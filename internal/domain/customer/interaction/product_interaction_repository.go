package interaction

import (
	"errors"
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductInteractionRepository interface {
	// 1. General & Like Produk
	GetBusinessIDByProductID(productID string) (string, error)
	ToggleProductLike(productID, accountID string) (bool, error)

	// 2. Review Produk (Ulasan Berbintang)
	CreateProductReview(review *models.ProductReview) error
	UpdateProductReview(review *models.ProductReview) error
	DeleteProductReview(reviewID string) error
	GetProductReviewByID(reviewID string) (*models.ProductReview, error)
	GetProductReviewsByProductID(productID string, rating *int) ([]models.ProductReview, error)
	HasUserReviewedProduct(productID, accountID string) (bool, error)
	RecalculateProductRating(productID string) error

	// 3. Like Review Produk
	ToggleReviewLike(reviewID, accountID string) (bool, error)
	GetReviewLikes(reviewID string) ([]models.ProductReviewLike, error)

	// 4. Balasan Review Produk (Replies)
	CreateReviewReply(comment *models.ProductReviewComment) error
	DeleteReviewReply(commentID string) error
	GetReviewReplyByID(commentID string) (*models.ProductReviewComment, error)
	ToggleReviewReplyLike(commentID, accountID string) (bool, error)
	GetReviewReplyLikes(commentID string) ([]models.ProductReviewCommentLike, error)

	// 5. Moderasi Review & Balasan
	CreateReviewReport(report *models.ProductReviewReport) error
	CreateReviewReplyReport(report *models.ProductReviewCommentReport) error

	// 6. Komentar Diskusi / Q&A Produk
	CreateProductComment(comment *models.ProductComment) error
	UpdateProductComment(comment *models.ProductComment) error
	DeleteProductComment(commentID string) error
	GetProductCommentByID(commentID string) (*models.ProductComment, error)
	GetProductCommentsByProductID(productID string) ([]models.ProductComment, error)
	ToggleProductCommentLike(commentID, accountID string) (bool, error)
	GetProductCommentLikes(commentID string) ([]models.ProductCommentLike, error)
	CreateProductCommentReport(report *models.ProductCommentReport) error
}

type productInteractionRepository struct {
	db *gorm.DB
}

func NewProductInteractionRepository(db *gorm.DB) ProductInteractionRepository {
	return &productInteractionRepository{db}
}

// ----------------------------------------------------
// 1. GENERAL & LIKE PRODUK
// ----------------------------------------------------

func (r *productInteractionRepository) GetBusinessIDByProductID(productID string) (string, error) {
	var product models.Product
	err := r.db.Select("business_id").Where("TRIM(LOWER(product_id)) = TRIM(LOWER(?))", strings.TrimSpace(productID)).First(&product).Error
	if err != nil {
		return "", err
	}
	return product.BusinessID, nil
}

func (r *productInteractionRepository) ToggleProductLike(productID, accountID string) (bool, error) {
	productID = strings.TrimSpace(productID)
	accountID = strings.TrimSpace(accountID)

	var product models.Product
	if err := r.db.Select("product_id").Where("TRIM(LOWER(product_id)) = TRIM(LOWER(?))", productID).First(&product).Error; err != nil {
		return false, errors.New("produk tidak ditemukan")
	}
	productID = product.ProductID

	var like models.ProductLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("product_id = ? AND account_id = ?", productID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Product{}).Where("product_id = ?", productID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("GREATEST(like_count - 1, 0)"),
				}).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.ProductLike{
				ProductID: productID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Product{}).Where("product_id = ?", productID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("like_count + 1"),
				}).Error; err != nil {
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

// ----------------------------------------------------
// 2. REVIEW PRODUK (ULASAN BERBINTANG)
// ----------------------------------------------------

func (r *productInteractionRepository) CreateProductReview(review *models.ProductReview) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(review).Error; err != nil {
			return err
		}
		return r.recalculateProductRatingTx(tx, review.ProductID)
	})
}

func (r *productInteractionRepository) UpdateProductReview(review *models.ProductReview) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(review).Error; err != nil {
			return err
		}
		return r.recalculateProductRatingTx(tx, review.ProductID)
	})
}

func (r *productInteractionRepository) DeleteProductReview(reviewID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var review models.ProductReview
		if err := tx.Where("review_id = ?", reviewID).First(&review).Error; err != nil {
			return err
		}
		if err := tx.Delete(&review).Error; err != nil {
			return err
		}
		return r.recalculateProductRatingTx(tx, review.ProductID)
	})
}

func (r *productInteractionRepository) GetProductReviewByID(reviewID string) (*models.ProductReview, error) {
	var review models.ProductReview
	err := r.db.Preload("Account").Preload("Product").Preload("Comments", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Account").Order("created_at ASC")
	}).Where("review_id = ?", reviewID).First(&review).Error
	return &review, err
}

func (r *productInteractionRepository) GetProductReviewsByProductID(productID string, rating *int) ([]models.ProductReview, error) {
	var reviews []models.ProductReview
	query := r.db.Preload("Account").Preload("Comments", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Account").Order("created_at ASC")
	}).Where("TRIM(LOWER(product_id)) = TRIM(LOWER(?))", strings.TrimSpace(productID))

	if rating != nil && *rating >= 1 && *rating <= 5 {
		query = query.Where("rating >= ? AND rating < ?", float64(*rating), float64(*rating)+1.0)
	}

	err := query.Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (r *productInteractionRepository) HasUserReviewedProduct(productID, accountID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.ProductReview{}).
		Where("product_id = ? AND account_id = ?", productID, accountID).
		Count(&count).Error
	return count > 0, err
}

func (r *productInteractionRepository) recalculateProductRatingTx(tx *gorm.DB, productID string) error {
	type AggResult struct {
		Count int64
		Avg   float64
	}
	var res AggResult
	err := tx.Model(&models.ProductReview{}).
		Select("COUNT(*) as count, COALESCE(AVG(rating), 0) as avg").
		Where("product_id = ? AND rating > 0", productID).
		Scan(&res).Error
	if err != nil {
		return err
	}

	roundedAvg := float64(int(res.Avg*10)) / 10
	return tx.Model(&models.Product{}).Where("product_id = ?", productID).
		Updates(map[string]interface{}{
			"review_count":   res.Count,
			"average_rating": roundedAvg,
		}).Error
}

func (r *productInteractionRepository) RecalculateProductRating(productID string) error {
	return r.recalculateProductRatingTx(r.db, productID)
}

// ----------------------------------------------------
// 3. LIKE REVIEW PRODUK
// ----------------------------------------------------

func (r *productInteractionRepository) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	var like models.ProductReviewLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("review_id = ? AND account_id = ?", reviewID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ProductReview{}).Where("review_id = ?", reviewID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("GREATEST(like_count - 1, 0)"),
				}).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.ProductReviewLike{
				ReviewID:  reviewID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ProductReview{}).Where("review_id = ?", reviewID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("like_count + 1"),
				}).Error; err != nil {
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

func (r *productInteractionRepository) GetReviewLikes(reviewID string) ([]models.ProductReviewLike, error) {
	var likes []models.ProductReviewLike
	err := r.db.Where("review_id = ?", reviewID).Find(&likes).Error
	return likes, err
}

// ----------------------------------------------------
// 4. BALASAN REVIEW PRODUK (REPLIES)
// ----------------------------------------------------

func (r *productInteractionRepository) CreateReviewReply(comment *models.ProductReviewComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&models.ProductReview{}).Where("review_id = ?", comment.ReviewID).
			Updates(map[string]interface{}{
				"comment_count": gorm.Expr("comment_count + 1"),
			}).Error
	})
}

func (r *productInteractionRepository) DeleteReviewReply(commentID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var comment models.ProductReviewComment
		if err := tx.Where("comment_id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}
		return tx.Model(&models.ProductReview{}).Where("review_id = ?", comment.ReviewID).
			Updates(map[string]interface{}{
				"comment_count": gorm.Expr("GREATEST(comment_count - 1, 0)"),
			}).Error
	})
}

func (r *productInteractionRepository) GetReviewReplyByID(commentID string) (*models.ProductReviewComment, error) {
	var comment models.ProductReviewComment
	err := r.db.Preload("Account").Where("comment_id = ?", commentID).First(&comment).Error
	return &comment, err
}

func (r *productInteractionRepository) ToggleReviewReplyLike(commentID, accountID string) (bool, error) {
	var like models.ProductReviewCommentLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("comment_id = ? AND account_id = ?", commentID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ProductReviewComment{}).Where("comment_id = ?", commentID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("GREATEST(like_count - 1, 0)"),
				}).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.ProductReviewCommentLike{
				CommentID: commentID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ProductReviewComment{}).Where("comment_id = ?", commentID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("like_count + 1"),
				}).Error; err != nil {
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

func (r *productInteractionRepository) GetReviewReplyLikes(commentID string) ([]models.ProductReviewCommentLike, error) {
	var likes []models.ProductReviewCommentLike
	err := r.db.Where("comment_id = ?", commentID).Find(&likes).Error
	return likes, err
}

// ----------------------------------------------------
// 5. MODERASI REVIEW
// ----------------------------------------------------

func (r *productInteractionRepository) CreateReviewReport(report *models.ProductReviewReport) error {
	return r.db.Create(report).Error
}

func (r *productInteractionRepository) CreateReviewReplyReport(report *models.ProductReviewCommentReport) error {
	return r.db.Create(report).Error
}

// ----------------------------------------------------
// 6. KOMENTAR DISKUSI / Q&A PRODUK
// ----------------------------------------------------

func (r *productInteractionRepository) CreateProductComment(comment *models.ProductComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		if comment.ReplyToCommentID != nil {
			return tx.Model(&models.ProductComment{}).Where("comment_id = ?", *comment.ReplyToCommentID).
				Updates(map[string]interface{}{
					"replies_count": gorm.Expr("replies_count + 1"),
				}).Error
		}
		return nil
	})
}

func (r *productInteractionRepository) UpdateProductComment(comment *models.ProductComment) error {
	return r.db.Save(comment).Error
}

func (r *productInteractionRepository) DeleteProductComment(commentID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var comment models.ProductComment
		if err := tx.Where("comment_id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}
		if comment.ReplyToCommentID != nil {
			return tx.Model(&models.ProductComment{}).Where("comment_id = ?", *comment.ReplyToCommentID).
				Updates(map[string]interface{}{
					"replies_count": gorm.Expr("GREATEST(replies_count - 1, 0)"),
				}).Error
		}
		return nil
	})
}

func (r *productInteractionRepository) GetProductCommentByID(commentID string) (*models.ProductComment, error) {
	var comment models.ProductComment
	err := r.db.Preload("Account").Where("comment_id = ?", commentID).First(&comment).Error
	return &comment, err
}

func (r *productInteractionRepository) GetProductCommentsByProductID(productID string) ([]models.ProductComment, error) {
	var comments []models.ProductComment
	err := r.db.Preload("Account").Where("TRIM(LOWER(product_id)) = TRIM(LOWER(?))", strings.TrimSpace(productID)).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *productInteractionRepository) ToggleProductCommentLike(commentID, accountID string) (bool, error) {
	var like models.ProductCommentLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("comment_id = ? AND account_id = ?", commentID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ProductComment{}).Where("comment_id = ?", commentID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("GREATEST(like_count - 1, 0)"),
				}).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.ProductCommentLike{
				CommentID: commentID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ProductComment{}).Where("comment_id = ?", commentID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("like_count + 1"),
				}).Error; err != nil {
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

func (r *productInteractionRepository) GetProductCommentLikes(commentID string) ([]models.ProductCommentLike, error) {
	var likes []models.ProductCommentLike
	err := r.db.Where("comment_id = ?", commentID).Find(&likes).Error
	return likes, err
}

func (r *productInteractionRepository) CreateProductCommentReport(report *models.ProductCommentReport) error {
	return r.db.Create(report).Error
}