package post

import (
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
)

type BusinessPostRepository interface {
	CreatePost(post *models.BusinessPost) error
	GetPostsByBusinessID(businessID string, page, limit int) ([]models.BusinessPost, int64, error)
	UpdatePost(post *models.BusinessPost) error
	DeletePost(postID, businessID string) error
	ToggleLike(postID, accountID string) (string, error)
	GetPostByID(postID string) (*models.BusinessPost, error)
}

type businessPostRepository struct {
	db *gorm.DB
}

func NewBusinessPostRepository(db *gorm.DB) BusinessPostRepository {
	return &businessPostRepository{db}
}

func (r *businessPostRepository) CreatePost(post *models.BusinessPost) error {
	return r.db.Create(post).Error
}

func (r *businessPostRepository) GetPostsByBusinessID(businessID string, page, limit int) ([]models.BusinessPost, int64, error) {
	var posts []models.BusinessPost
	var totalRows int64

	query := r.db.Model(&models.BusinessPost{}).Where("business_id = ?", businessID)
	query.Count(&totalRows)

	offset := (page - 1) * limit
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error

	return posts, totalRows, err
}

func (r *businessPostRepository) UpdatePost(post *models.BusinessPost) error {
	return r.db.Save(post).Error
}

func (r *businessPostRepository) DeletePost(postID, businessID string) error {
	return r.db.Where("post_id = ? AND business_id = ?", postID, businessID).Delete(&models.BusinessPost{}).Error
}

// Fitur Anti-Spam Like & Kalkulasi Otomatis
func (r *businessPostRepository) ToggleLike(postID, accountID string) (string, error) {
	var status string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existingLike models.BusinessPostLike
		result := tx.Where("post_id = ? AND account_id = ?", postID, accountID).First(&existingLike)

		if result.Error == nil {
			// Jika sudah ada, berarti user melakukan UNLIKE
			if err := tx.Delete(&existingLike).Error; err != nil { return err }
			if err := tx.Model(&models.BusinessPost{}).Where("post_id = ?", postID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil { return err }
			status = "unliked"
		} else {
			// Jika belum ada, berarti user melakukan LIKE
			newLike := models.BusinessPostLike{PostID: postID, AccountID: accountID}
			if err := tx.Create(&newLike).Error; err != nil { return err }
			if err := tx.Model(&models.BusinessPost{}).Where("post_id = ?", postID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil { return err }
			status = "liked"
		}
		return nil
	})
	return status, err
}

func (r *businessPostRepository) GetPostByID(postID string) (*models.BusinessPost, error) {
	var post models.BusinessPost
	// Preload "Business" jika Anda butuh info nama tokonya sekalian
	err := r.db.Where("post_id = ?", postID).First(&post).Error
	return &post, err
}