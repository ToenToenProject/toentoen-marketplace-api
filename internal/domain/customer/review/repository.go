package review

import (
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
	"errors"
)

type ReviewRepository interface {
	CreateReview(review *models.Review) error
	GetReviewByID(reviewID string) (*models.Review, error)
	UpdateReview(review *models.Review) error
	GetAllReviews(businessID string) ([]models.Review, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)

	ToggleReviewLike(reviewID, accountID string) (bool, error)
	CreateReviewComment(comment *models.BusinessReviewComment) error
	GetReviewComments(reviewID string) ([]models.BusinessReviewComment, error)
	GetReviewCommentByID(commentID string) (*models.BusinessReviewComment, error)
	ToggleReviewCommentLike(commentID, accountID string) (bool, error)
	CreateReviewReport(report *models.BusinessReviewReport) error
	CreateReviewCommentReport(report *models.BusinessReviewCommentReport) error
	GetReviewCommentLikes(commentID string) ([]models.BusinessReviewCommentLike, error)
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
	err := r.db.Where("review_id = ?", reviewID).First(&review).Error
	return &review, err
}

func (r *reviewRepository) UpdateReview(review *models.Review) error {
	return r.db.Save(review).Error
}

func (r *reviewRepository) GetAllReviews(businessID string) ([]models.Review, error) {
	var reviews []models.Review
	query := r.db.Model(&models.Review{})

	if businessID != "" {
		query = query.Where("business_id = ?", businessID)
	}

	err := query.Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

// --- Implementasinya ---
func (r *reviewRepository) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	var summary models.ReviewSummary
	var total float64 // Untuk menghitung rata-rata

	// Struct sementara untuk menangkap hasil GROUP BY SQL
	type Result struct {
		Rating int
		Count  int64
	}
	var results []Result

	// Query 1: Review Bisnis
	err := r.db.Model(&models.Review{}).
		Select("rating, count(*) as count").
		Where("business_id = ?", businessID).
		Group("rating").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Petakan hasil ke dalam struct Distribution
	for _, res := range results {
		summary.TotalReviews += res.Count
		total += float64(res.Rating) * float64(res.Count)

		switch res.Rating {
		case 5: summary.Distribution.Star5 += res.Count
		case 4: summary.Distribution.Star4 += res.Count
		case 3: summary.Distribution.Star3 += res.Count
		case 2: summary.Distribution.Star2 += res.Count
		case 1: summary.Distribution.Star1 += res.Count
		}
	}

	if summary.TotalReviews > 0 {
		summary.AverageRating = total / float64(summary.TotalReviews)
		summary.AverageRating = float64(int(summary.AverageRating*10)) / 10 
	}

	return &summary, nil
}

func (r *reviewRepository) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	var like models.BusinessReviewLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("review_id = ? AND account_id = ?", reviewID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Review{}).Where("review_id = ?", reviewID).
				Updates(map[string]interface{}{"like_count": gorm.Expr("like_count - 1")}).Error; err != nil {
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
			if err := tx.Model(&models.Review{}).Where("review_id = ?", reviewID).
				Updates(map[string]interface{}{"like_count": gorm.Expr("like_count + 1")}).Error; err != nil {
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
		return tx.Model(&models.Review{}).Where("review_id = ?", comment.ReviewID).
			Updates(map[string]interface{}{"comment_count": gorm.Expr("comment_count + 1")}).Error
	})
}

func (r *reviewRepository) GetReviewComments(reviewID string) ([]models.BusinessReviewComment, error) {
	var comments []models.BusinessReviewComment
	err := r.db.Preload("Account.Customer").Where("review_id = ?", reviewID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *reviewRepository) GetReviewCommentByID(commentID string) (*models.BusinessReviewComment, error) {
	var comment models.BusinessReviewComment
	err := r.db.Preload("Account.Customer").Where("comment_id = ?", commentID).First(&comment).Error
	return &comment, err
}

func (r *reviewRepository) ToggleReviewCommentLike(commentID, accountID string) (bool, error) {
	var like models.BusinessReviewCommentLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("comment_id = ? AND account_id = ?", commentID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessReviewComment{}).Where("comment_id = ?", commentID).
				Updates(map[string]interface{}{"like_count": gorm.Expr("like_count - 1")}).Error; err != nil {
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
			if err := tx.Model(&models.BusinessReviewComment{}).Where("comment_id = ?", commentID).
				Updates(map[string]interface{}{"like_count": gorm.Expr("like_count + 1")}).Error; err != nil {
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