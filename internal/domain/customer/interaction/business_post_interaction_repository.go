package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type PostInteractionRepository interface {
	TogglePostLike(postID, accountID string) (bool, error)
	CreatePostComment(comment *models.BusinessPostComment) error
	GetPostCommentByID(commentID string) (*models.BusinessPostComment, error)
	GetCommentsByPostID(postID string) ([]models.BusinessPostComment, error)
	GetBusinessIDByPostID(postID string) (string, error)
}

type postInteractionRepository struct {
	db *gorm.DB
}

func NewPostInteractionRepository(db *gorm.DB) PostInteractionRepository {
	return &postInteractionRepository{db}
}

// 1. TOGGLE LIKE POSTINGAN
func (r *postInteractionRepository) TogglePostLike(postID, accountID string) (bool, error) {
	var like models.BusinessPostLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Cek apakah akun ini sudah pernah menyukai postingan ini
		result := tx.Where("post_id = ? AND account_id = ?", postID, accountID).First(&like)

		if result.Error == nil {
			// JIKA SUDAH ADA: Lakukan UNLIKE (Hapus baris data)
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			// Kurangi kolom like_count di tabel posts secara atomik
			if err := tx.Model(&models.BusinessPost{}).Where("post_id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// JIKA BELUM ADA: Lakukan LIKE (Buat baris data baru)
			newLike := models.BusinessPostLike{
				PostID:    postID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			// Tambah kolom like_count di tabel posts secara atomik
			if err := tx.Model(&models.BusinessPost{}).Where("post_id = ?", postID).
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

// 2. CREATE KOMENTAR / BALASAN POSTINGAN
func (r *postInteractionRepository) CreatePostComment(comment *models.BusinessPostComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Simpan data komentar ke dalam tabel post_comments
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		// Tambah kolom comment_count di tabel posts secara atomik
		return tx.Model(&models.BusinessPost{}).Where("post_id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}

// 3. AMBIL DATA SATU KOMENTAR BERDASARKAN ID
// Fungsi ini sangat krusial untuk Service guna mengecek level kedalaman parent-nya
func (r *postInteractionRepository) GetPostCommentByID(commentID string) (*models.BusinessPostComment, error) {
	var comment models.BusinessPostComment
	// Ambil juga data dasar Account untuk mendapatkan nama/ID acuan mention secara otomatis
	err := r.db.Preload("Account").Where("comment_id = ?", commentID).First(&comment).Error
	return &comment, err
}

// 4. AMBIL SEMUA KOMENTAR DALAM SATU POSTINGAN
func (r *postInteractionRepository) GetCommentsByPostID(postID string) ([]models.BusinessPostComment, error) {
	var comments []models.BusinessPostComment
	// Tarik data datar diurutkan dari yang paling lama (ASC) agar urutan diskusi runut
	// Gunakan Preload("Account") untuk mendapatkan informasi user yang menulis komentar
	err := r.db.Preload("Account").Where("post_id = ?", postID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *postInteractionRepository) GetBusinessIDByPostID(postID string) (string, error) {
	var post models.BusinessPost
	
	// Cari postingan berdasarkan ID, tapi HANYA tarik kolom business_id saja untuk hemat memori
	err := r.db.Select("business_id").Where("post_id = ?", postID).First(&post).Error
	if err != nil {
		return "", err
	}
	
	return post.BusinessID, nil
}