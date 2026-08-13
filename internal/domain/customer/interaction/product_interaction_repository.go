package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductInteractionRepository interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	CreateProductReview(review *models.ProductReview) error
	GetProductReviewByID(reviewID string) (*models.ProductReview, error)
	GetReviewsByProductID(productID string) ([]models.ProductReview, error)
	GetAllProductReviewsByBusinessID(businessID string) ([]models.ProductReview, error)
	GetBusinessIDByProductID(productID string) (string, error)

	ToggleReviewLike(reviewID, accountID string) (bool, error)
	CreateReviewReport(report *models.ProductReviewReport) error
	GetReviewLikes(reviewID string) ([]models.ProductReviewLike, error)
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
		// Cek apakah akun ini sudah pernah menyukai produk ini
		result := tx.Where("product_id = ? AND account_id = ?", productID, accountID).First(&like)

		if result.Error == nil {
			// JIKA SUDAH ADA: Lakukan UNLIKE (Hapus baris data)
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			
			// Gunakan Updates dengan map agar gorm.Expr dieksekusi dengan benar
			if err := tx.Model(&models.Product{}).Where("product_id = ?", productID).
				Updates(map[string]interface{}{
					"like_count": gorm.Expr("like_count - 1"),
				}).Error; err != nil {
				return err
			}
			isLiked = false
			
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// JIKA BELUM ADA: Lakukan LIKE (Buat baris data baru)
			newLike := models.ProductLike{
				ProductID: productID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			
			// Gunakan Updates dengan map
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

// 2. CREATE REVIEW / BALASAN PRODUK
func (r *productInteractionRepository) CreateProductReview(review *models.ProductReview) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Simpan data review ke dalam tabel product_reviews
		if err := tx.Create(review).Error; err != nil {
			return err
		}
		
		updateFields := map[string]interface{}{}

		if review.Rating != nil && *review.Rating > 0 {
			// Jika ada rating, kita update review_count dan kalkulasi ulang average_rating
			// 1. Dapatkan produk saat ini
			var prod models.Product
			if err := tx.Select("average_rating", "review_count").Where("product_id = ?", review.ProductID).First(&prod).Error; err != nil {
				return err
			}
			
			newReviewCount := prod.ReviewCount + 1
			newAverage := ((prod.AverageRating * float64(prod.ReviewCount)) + *review.Rating) / float64(newReviewCount)
			
			updateFields["review_count"] = newReviewCount
			updateFields["average_rating"] = newAverage
		}

		if len(updateFields) > 0 {
			return tx.Model(&models.Product{}).Where("product_id = ?", review.ProductID).
				Updates(updateFields).Error
		}

		return nil
	})
}

// 3. AMBIL DATA SATU REVIEW BERDASARKAN ID
func (r *productInteractionRepository) GetProductReviewByID(reviewID string) (*models.ProductReview, error) {
	var review models.ProductReview
	err := r.db.Preload("Account.Customer").Where("review_id = ?", reviewID).First(&review).Error
	return &review, err
}

// 4. AMBIL SEMUA REVIEW DALAM SATU PRODUK
func (r *productInteractionRepository) GetReviewsByProductID(productID string) ([]models.ProductReview, error) {
	var reviews []models.ProductReview
	err := r.db.Preload("Account.Customer").Where("product_id = ?", productID).Order("created_at ASC").Find(&reviews).Error
	return reviews, err
}

func (r *productInteractionRepository) GetAllProductReviewsByBusinessID(businessID string) ([]models.ProductReview, error) {
	var reviews []models.ProductReview
	// We only want root reviews (which are the actual product reviews, typically ReplyToReviewID is null)
	err := r.db.Preload("Account.Customer").Preload("Product").Preload("Replies").Where("business_id = ? AND reply_to_review_id IS NULL", businessID).Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

// 5. AMBIL BUSINESS ID BERDASARKAN PRODUCT ID (Mencegah Error Foreign Key)
func (r *productInteractionRepository) GetBusinessIDByProductID(productID string) (string, error) {
	var product models.Product
	
	// Cari produk berdasarkan ID, tapi HANYA tarik kolom business_id saja untuk hemat memori
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
					"like_count": gorm.Expr("like_count - 1"),
				}).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.ProductReviewLike{
				ReviewID: reviewID,
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