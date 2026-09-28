package review

import (
	"errors"
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ReviewRepository interface {
	CreateReview(review *models.Review) error
	GetReviewByID(reviewID string) (*models.Review, error)
	UpdateReview(review *models.Review) error
	DeleteReview(reviewID string) error
	HasUserReviewedBusiness(businessID, accountID string) (bool, error)
	GetLastDeletedReview(businessID, accountID string) (*models.Review, error)
	GetReviewByBusinessAndAccount(businessID, accountID string) (*models.Review, error)
	GetReviewsPaginated(businessID string, ratingFilter int, sort string, page, limit int) ([]models.Review, int64, error)
	GetAllReviews(businessID string) ([]models.Review, error)
	GetEligibleReviewsForSmartDisplay(businessID string) ([]models.Review, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)

	GetReviewLikes(reviewID string) ([]models.BusinessReviewLike, error)
	ToggleReviewLike(reviewID, accountID string) (bool, error)
	CreateReviewComment(comment *models.BusinessReviewComment) error
	GetReviewComments(reviewID string) ([]models.BusinessReviewComment, error)
	GetReviewCommentByID(commentID string) (*models.BusinessReviewComment, error)
	DeleteReviewComment(commentID string) error
	ToggleReviewCommentLike(commentID, accountID string) (bool, error)
	GetReviewCommentLikes(commentID string) ([]models.BusinessReviewCommentLike, error)
	CreateReviewReport(report *models.BusinessReviewReport) error
	CreateReviewCommentReport(report *models.BusinessReviewCommentReport) error
}

type reviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &reviewRepository{db}
}

func (r *reviewRepository) CreateReview(review *models.Review) error {
	return r.db.Create(review).Error
}

func (r *reviewRepository) GetReviewByID(reviewID string) (*models.Review, error) {
	var review models.Review
	err := r.db.Preload("Account").Preload("Business").Where("review_id = ?", reviewID).First(&review).Error
	return &review, err
}

func (r *reviewRepository) UpdateReview(review *models.Review) error {
	return r.db.Save(review).Error
}

func (r *reviewRepository) DeleteReview(reviewID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("review_id = ?", reviewID).Delete(&models.BusinessReviewLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("review_id = ?", reviewID).Delete(&models.BusinessReviewComment{}).Error; err != nil {
			return err
		}
		return tx.Where("review_id = ?", reviewID).Delete(&models.Review{}).Error
	})
}

func (r *reviewRepository) HasUserReviewedBusiness(businessID, accountID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Review{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count).Error
	return count > 0, err
}

func (r *reviewRepository) GetLastDeletedReview(businessID, accountID string) (*models.Review, error) {
	var review models.Review
	err := r.db.Unscoped().
		Where("business_id = ? AND account_id = ? AND deleted_at IS NOT NULL", businessID, accountID).
		Order("deleted_at DESC").
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) GetReviewByBusinessAndAccount(businessID, accountID string) (*models.Review, error) {
	var review models.Review
	err := r.db.Preload("Account").Where("business_id = ? AND account_id = ?", businessID, accountID).First(&review).Error
	return &review, err
}

func (r *reviewRepository) GetReviewsPaginated(businessID string, ratingFilter int, sort string, page, limit int) ([]models.Review, int64, error) {
	var reviews []models.Review
	var total int64

	query := r.db.Model(&models.Review{}).Where("business_id = ?", businessID)

	if ratingFilter >= 1 && ratingFilter <= 5 {
		query = query.Where("rating = ?", ratingFilter)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	switch sort {
	case "highest":
		query = query.Order("rating DESC, created_at DESC")
	case "lowest":
		query = query.Order("rating ASC, created_at DESC")
	case "most_liked":
		query = query.Order("like_count DESC, created_at DESC")
	case "oldest":
		query = query.Order("created_at ASC")
	default: // "newest"
		query = query.Order("created_at DESC")
	}

	offset := (page - 1) * limit
	err := query.Preload("Account").Offset(offset).Limit(limit).Find(&reviews).Error
	return reviews, total, err
}

func (r *reviewRepository) GetAllReviews(businessID string) ([]models.Review, error) {
	var reviews []models.Review
	query := r.db.Model(&models.Review{})

	if businessID != "" {
		query = query.Where("business_id = ?", businessID)
	}

	err := query.Preload("Account").Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (r *reviewRepository) GetEligibleReviewsForSmartDisplay(businessID string) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Account").
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *reviewRepository) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	var summary models.ReviewSummary
	var total float64

	type Result struct {
		Rating int
		Count  int64
	}
	var results []Result

	err := r.db.Model(&models.Review{}).
		Select("rating, count(*) as count").
		Where("business_id = ?", businessID).
		Group("rating").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		summary.TotalReviews += res.Count
		total += float64(res.Rating) * float64(res.Count)

		switch res.Rating {
		case 5:
			summary.Distribution.Star5 += res.Count
		case 4:
			summary.Distribution.Star4 += res.Count
		case 3:
			summary.Distribution.Star3 += res.Count
		case 2:
			summary.Distribution.Star2 += res.Count
		case 1:
			summary.Distribution.Star1 += res.Count
		}
	}

	if summary.TotalReviews > 0 {
		summary.AverageRating = total / float64(summary.TotalReviews)
		summary.AverageRating = float64(int(summary.AverageRating*10)) / 10
	}

	return &summary, nil
}

func (r *reviewRepository) GetReviewLikes(reviewID string) ([]models.BusinessReviewLike, error) {
	var likes []models.BusinessReviewLike
	err := r.db.Where("review_id = ?", reviewID).Find(&likes).Error
	return likes, err
}

func (r *reviewRepository) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	reviewID = strings.TrimSpace(reviewID)
	accountID = strings.TrimSpace(accountID)
	var like models.BusinessReviewLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?)) AND TRIM(LOWER(account_id)) = TRIM(LOWER(?))", reviewID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Review{}).Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", reviewID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.BusinessReviewLike{
				ReviewID:  reviewID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Review{}).Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", reviewID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			isLiked = true
		} else {
			return result.Error
		}
		return nil
	})

	return isLiked, err
}

func (r *reviewRepository) CreateReviewComment(comment *models.BusinessReviewComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&models.Review{}).Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", comment.ReviewID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}

func (r *reviewRepository) GetReviewComments(reviewID string) ([]models.BusinessReviewComment, error) {
	var comments []models.BusinessReviewComment
	err := r.db.Preload("Account").Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", strings.TrimSpace(reviewID)).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *reviewRepository) GetReviewCommentByID(commentID string) (*models.BusinessReviewComment, error) {
	var comment models.BusinessReviewComment
	err := r.db.Preload("Account").Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", strings.TrimSpace(commentID)).First(&comment).Error
	return &comment, err
}

func (r *reviewRepository) DeleteReviewComment(commentID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var comment models.BusinessReviewComment
		if err := tx.Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", strings.TrimSpace(commentID)).First(&comment).Error; err != nil {
			return err
		}
		if err := tx.Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", strings.TrimSpace(commentID)).Delete(&models.BusinessReviewCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", strings.TrimSpace(commentID)).Delete(&models.BusinessReviewComment{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Review{}).Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", comment.ReviewID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - 1, 0)")).Error
	})
}

func (r *reviewRepository) ToggleReviewCommentLike(commentID, accountID string) (bool, error) {
	commentID = strings.TrimSpace(commentID)
	accountID = strings.TrimSpace(accountID)
	var like models.BusinessReviewCommentLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?)) AND TRIM(LOWER(account_id)) = TRIM(LOWER(?))", commentID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessReviewComment{}).Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", commentID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.BusinessReviewCommentLike{
				CommentID: commentID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessReviewComment{}).Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			isLiked = true
		} else {
			return result.Error
		}
		return nil
	})

	return isLiked, err
}

func (r *reviewRepository) CreateReviewReport(report *models.BusinessReviewReport) error {
	return r.db.Create(report).Error
}

func (r *reviewRepository) CreateReviewCommentReport(report *models.BusinessReviewCommentReport) error {
	return r.db.Create(report).Error
}

func (r *reviewRepository) GetReviewCommentLikes(commentID string) ([]models.BusinessReviewCommentLike, error) {
	var likes []models.BusinessReviewCommentLike
	err := r.db.Where("comment_id = ?", commentID).Find(&likes).Error
	return likes, err
}