package report

import (
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"gorm.io/gorm"
)

type Repository interface {
	CreateReport(report *models.BusinessReport) error
	CreateUniversalReport(report *models.Report) error
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

	switch targetType {
	case "business":
		err = r.db.Table("businesses").Where("business_id = ?", targetID).Count(&count).Error
	case "product":
		err = r.db.Table("products").Where("product_id = ?", targetID).Count(&count).Error
	case "review":
		// Bisa di product_reviews atau reviews (toko)
		var pCount, bCount int64
		_ = r.db.Table("product_reviews").Where("review_id = ?", targetID).Count(&pCount).Error
		_ = r.db.Table("reviews").Where("review_id = ?", targetID).Count(&bCount).Error
		count = pCount + bCount
	case "comment":
		// Bisa di product_comments, product_review_comments, business_review_comments, business_post_comments
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
