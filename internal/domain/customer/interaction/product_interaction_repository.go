package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductInteractionRepository interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	CreateProductComment(comment *models.ProductComment) error
	GetProductCommentByID(commentID string) (*models.ProductComment, error)
	GetCommentsByProductID(productID string) ([]models.ProductComment, error)
	
	// 🪄 FUNGSI BARU: Untuk mencari BusinessID secara otomatis
	GetBusinessIDByProductID(productID string) (string, error) 
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

// 2. CREATE KOMENTAR / BALASAN PRODUK
func (r *productInteractionRepository) CreateProductComment(comment *models.ProductComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Simpan data komentar ke dalam tabel product_comments
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		
		// Ubah menjadi comment_count dan gunakan Updates dengan map
		return tx.Model(&models.Product{}).Where("product_id = ?", comment.ProductID).
			Updates(map[string]interface{}{
				"comment_count": gorm.Expr("comment_count + 1"),
			}).Error
	})
}

// 3. AMBIL DATA SATU KOMENTAR BERDASARKAN ID
func (r *productInteractionRepository) GetProductCommentByID(commentID string) (*models.ProductComment, error) {
	var comment models.ProductComment
	err := r.db.Preload("Account").Where("comment_id = ?", commentID).First(&comment).Error
	return &comment, err
}

// 4. AMBIL SEMUA KOMENTAR DALAM SATU PRODUK
func (r *productInteractionRepository) GetCommentsByProductID(productID string) ([]models.ProductComment, error) {
	var comments []models.ProductComment
	err := r.db.Preload("Account").Where("product_id = ?", productID).Order("created_at ASC").Find(&comments).Error
	return comments, err
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