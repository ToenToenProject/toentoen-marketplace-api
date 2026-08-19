package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductInteractionRepository interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	CreateProductReview(review *models.ProductReview) error
	UpdateProductReview(review *models.ProductReview) error
	DeleteProductReview(reviewID string) error
	GetProductReviewByID(reviewID string) (*models.ProductReview, error)
	GetReviewsByProductID(productID string) ([]models.ProductReview, error)
	GetAllProductReviewsByBusinessID(businessID string) ([]models.ProductReview, error)
	GetBusinessIDByProductID(productID string) (string, error)
	HasUserReviewedProduct(productID, accountID string) (bool, error)

	ToggleReviewLike(reviewID, accountID string) (bool, error)
	CreateReviewReport(report *models.ProductReviewReport) error
	GetReviewLikes(reviewID string) ([]models.ProductReviewLike, error)
	RecalculateProductRating(productID string) error
}

type productInteractionRepository struct {
	db *gorm.DB
}

func NewProductInteractionRepository(db *gorm.DB) ProductInteractionRepository {
	return &productInteractionRepository{db}
}

// 1. TOGGLE LIKE PRODUK
func (r *productInteractionRepository) ToggleProductLike(productID, accountID string) (bool, error) {
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

// 2. CREATE REVIEW / COMMENT PRODUK
func (r *productInteractionRepository) CreateProductReview(review *models.ProductReview) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(review).Error; err != nil {
			return err
		}
		return r.recalculateProductRatingTx(tx, review.ProductID)
	})
}

// 3. UPDATE REVIEW / COMMENT PRODUK
func (r *productInteractionRepository) UpdateProductReview(review *models.ProductReview) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(review).Error; err != nil {
			return err
		}
		return r.recalculateProductRatingTx(tx, review.ProductID)
	})
}

// 4. DELETE REVIEW / COMMENT PRODUK
func (r *productInteractionRepository) DeleteProductReview(reviewID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var review models.ProductReview
		if err := tx.Where("review_id = ?", reviewID).First(&review).Error; err != nil {
			return err
		}

		if err := tx.Where("review_id = ?", reviewID).Delete(&models.ProductReviewLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("reply_to_review_id = ?", reviewID).Delete(&models.ProductReview{}).Error; err != nil {
			return err
		}
		if err := tx.Where("review_id = ?", reviewID).Delete(&models.ProductReview{}).Error; err != nil {
			return err
		}

		return r.recalculateProductRatingTx(tx, review.ProductID)
	})
}

func (r *productInteractionRepository) HasUserReviewedProduct(productID, accountID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.ProductReview{}).
		Where("product_id = ? AND account_id = ? AND reply_to_review_id IS NULL AND rating IS NOT NULL", productID, accountID).
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
		Where("product_id = ? AND reply_to_review_id IS NULL AND rating IS NOT NULL AND rating > 0", productID).
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

func (r *productInteractionRepository) GetProductReviewByID(reviewID string) (*models.ProductReview, error) {
	var review models.ProductReview
	err := r.db.Preload("Account").Preload("Product").Where("review_id = ?", reviewID).First(&review).Error
	return &review, err
}

func (r *productInteractionRepository) GetReviewsByProductID(productID string) ([]models.ProductReview, error) {
	var reviews []models.ProductReview
	err := r.db.Preload("Account").Where("product_id = ?", productID).Order("created_at ASC").Find(&reviews).Error
	return reviews, err
}

func (r *productInteractionRepository) GetAllProductReviewsByBusinessID(businessID string) ([]models.ProductReview, error) {
	var reviews []models.ProductReview
	err := r.db.Preload("Account").Preload("Product").Preload("Replies").
		Where("business_id = ? AND reply_to_review_id IS NULL", businessID).
		Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (r *productInteractionRepository) GetBusinessIDByProductID(productID string) (string, error) {
	var product models.Product
	err := r.db.Select("business_id").Where("product_id = ?", productID).First(&product).Error
	if err != nil {
		return "", err
	}
	return product.BusinessID, nil
}

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

func (r *productInteractionRepository) CreateReviewReport(report *models.ProductReviewReport) error {
	return r.db.Create(report).Error
}

func (r *productInteractionRepository) GetReviewLikes(reviewID string) ([]models.ProductReviewLike, error) {
	var likes []models.ProductReviewLike
	err := r.db.Where("review_id = ?", reviewID).Find(&likes).Error
	return likes, err
}