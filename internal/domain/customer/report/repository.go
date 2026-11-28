package report

import (
	"strings"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"gorm.io/gorm"
)

type Repository interface {
	CreateReport(report *models.BusinessReport) error
	CreateUniversalReport(report *models.Report) error
	HasUserReportedTarget(accountID, targetID string) (bool, error)
	CountActiveReportsForTarget(targetID string) (int64, error)
	GetReportsPaginated(filter models.ReportFilterQuery) ([]models.Report, int64, error)
	GetUserReports(accountID string, page, limit int) ([]models.Report, int64, error)
	GetReportByID(reportID string) (*models.Report, error)
	DeleteReport(reportID string) error
	UpdateReportStatus(reportID, status string) error
	UpdateReportResolution(reportID, status, adminNote, resolvedBy string) error
	TakedownTargetContent(targetType, targetID string) error
	CheckBusinessExists(businessID string) (bool, error)
	CheckTargetExists(targetType, targetID string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository() Repository {
	return &repository{
		db: database.DB,
	}
}

func (r *repository) CreateReport(report *models.BusinessReport) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(report).Error
	})
}

func (r *repository) CreateUniversalReport(report *models.Report) error {
	return r.db.Create(report).Error
}

func (r *repository) HasUserReportedTarget(accountID, targetID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Report{}).
		Where("account_id = ? AND target_id = ?", strings.TrimSpace(accountID), strings.TrimSpace(targetID)).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) CountActiveReportsForTarget(targetID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Report{}).
		Where("target_id = ? AND status NOT IN ('rejected', 'dismissed')", strings.TrimSpace(targetID)).
		Count(&count).Error
	return count, err
}

func (r *repository) GetReportsPaginated(filter models.ReportFilterQuery) ([]models.Report, int64, error) {
	var reports []models.Report
	var total int64

	query := r.db.Model(&models.Report{}).Preload("Account")

	if filter.Status != "" {
		query = query.Where("status = ?", strings.ToLower(strings.TrimSpace(filter.Status)))
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", strings.ToLower(strings.TrimSpace(filter.Priority)))
	}
	if filter.TargetType != "" {
		query = query.Where("target_type = ?", strings.ToLower(strings.TrimSpace(filter.TargetType)))
	}
	if filter.TargetID != "" {
		query = query.Where("target_id = ?", strings.TrimSpace(filter.TargetID))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&reports).Error
	return reports, total, err
}

func (r *repository) GetUserReports(accountID string, page, limit int) ([]models.Report, int64, error) {
	var reports []models.Report
	var total int64

	query := r.db.Model(&models.Report{}).Preload("Account").
		Where("account_id = ?", strings.TrimSpace(accountID))

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&reports).Error
	return reports, total, err
}

func (r *repository) GetReportByID(reportID string) (*models.Report, error) {
	var report models.Report
	err := r.db.Preload("Account").
		Where("report_id = ?", strings.TrimSpace(reportID)).
		First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *repository) DeleteReport(reportID string) error {
	return r.db.Where("report_id = ?", strings.TrimSpace(reportID)).
		Delete(&models.Report{}).Error
}

func (r *repository) UpdateReportStatus(reportID, status string) error {
	return r.db.Model(&models.Report{}).
		Where("report_id = ?", strings.TrimSpace(reportID)).
		UpdateColumn("status", strings.ToLower(strings.TrimSpace(status))).Error
}

func (r *repository) UpdateReportResolution(reportID, status, adminNote, resolvedBy string) error {
	updates := map[string]interface{}{
		"status": strings.ToLower(strings.TrimSpace(status)),
	}

	// resolved_at is set for terminal statuses (resolved, rejected, dismissed).
	terminalStatuses := map[string]bool{"resolved": true, "rejected": true, "dismissed": true}
	if terminalStatuses[strings.ToLower(strings.TrimSpace(status))] {
		updates["resolved_at"] = gorm.Expr("NOW()")
	}
	if strings.TrimSpace(adminNote) != "" {
		updates["admin_note"] = strings.TrimSpace(adminNote)
	}
	if strings.TrimSpace(resolvedBy) != "" {
		updates["resolved_by"] = strings.TrimSpace(resolvedBy)
	}

	return r.db.Model(&models.Report{}).
		Where("report_id = ?", strings.TrimSpace(reportID)).
		Updates(updates).Error
}

func (r *repository) TakedownTargetContent(targetType, targetID string) error {
	targetID = strings.TrimSpace(targetID)
	targetType = strings.ToLower(strings.TrimSpace(targetType))

	return r.db.Transaction(func(tx *gorm.DB) error {
		switch targetType {
		case "comment":
			_ = tx.Where("comment_id = ?", targetID).Delete(&models.ProductComment{})
			_ = tx.Where("comment_id = ?", targetID).Delete(&models.ProductReviewComment{})
			_ = tx.Where("comment_id = ?", targetID).Delete(&models.BusinessReviewComment{})
			_ = tx.Where("comment_id = ?", targetID).Delete(&models.BusinessPostComment{})
		case "review":
			_ = tx.Table("business_review_likes").Where("review_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Table("business_review_comments").Where("review_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Where("review_id = ?", targetID).Delete(&models.Review{})

			_ = tx.Table("product_review_likes").Where("review_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Table("product_review_comments").Where("review_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Where("review_id = ?", targetID).Delete(&models.ProductReview{})
		case "post":
			_ = tx.Table("business_post_likes").Where("post_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Table("business_post_comments").Where("post_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Where("post_id = ?", targetID).Delete(&models.BusinessPost{})
		case "product":
			_ = tx.Table("product_likes").Where("product_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Table("product_comments").Where("product_id = ?", targetID).Delete(map[string]interface{}{})
			_ = tx.Where("product_id = ?", targetID).Delete(&models.Product{})
		}
		return nil
	})
}

func (r *repository) CheckBusinessExists(businessID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Business{}).Where("business_id = ?", businessID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) CheckTargetExists(targetType, targetID string) (bool, error) {
	var count int64
	var err error

	targetID = strings.TrimSpace(targetID)

	switch targetType {
	case "business":
		err = r.db.Table("businesses").Where("business_id = ?", targetID).Count(&count).Error
	case "product":
		err = r.db.Table("products").Where("product_id = ?", targetID).Count(&count).Error
	case "review":
		var pCount, bCount int64
		_ = r.db.Table("product_reviews").Where("review_id = ?", targetID).Count(&pCount).Error
		_ = r.db.Table("reviews").Where("review_id = ?", targetID).Count(&bCount).Error
		count = pCount + bCount
	case "comment":
		var c1, c2, c3, c4 int64
		_ = r.db.Table("product_comments").Where("comment_id = ?", targetID).Count(&c1).Error
		_ = r.db.Table("product_review_comments").Where("comment_id = ?", targetID).Count(&c2).Error
		_ = r.db.Table("business_review_comments").Where("comment_id = ?", targetID).Count(&c3).Error
		_ = r.db.Table("business_post_comments").Where("comment_id = ?", targetID).Count(&c4).Error
		count = c1 + c2 + c3 + c4
	case "post":
		err = r.db.Table("business_posts").Where("post_id = ?", targetID).Count(&count).Error
	default:
		return true, nil
	}

	if err != nil {
		return false, err
	}
	return count > 0, nil
}
