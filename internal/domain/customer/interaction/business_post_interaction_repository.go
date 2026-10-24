package interaction

import (
	"errors"
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type PostInteractionRepository interface {
	TogglePostLike(postID, accountID string) (bool, error)
	CreatePostComment(comment *models.BusinessPostComment) error
	UpdatePostComment(comment *models.BusinessPostComment) error
	DeletePostComment(commentID string) error
	GetPostCommentByID(commentID string) (*models.BusinessPostComment, error)
	GetCommentsByPostID(postID string) ([]models.BusinessPostComment, error)
	GetBusinessIDByPostID(postID string) (string, error)
	TogglePostCommentLike(commentID, accountID string) (bool, error)
	GetPostCommentLikes(commentID string) ([]models.BusinessPostCommentLike, error)
	CreatePostReport(report *models.BusinessPostReport) error
}

type postInteractionRepository struct {
	db *gorm.DB
}

func NewPostInteractionRepository(db *gorm.DB) PostInteractionRepository {
	return &postInteractionRepository{db}
}

// 1. TOGGLE LIKE POSTINGAN
func (r *postInteractionRepository) TogglePostLike(postID, accountID string) (bool, error) {
	postID = strings.TrimSpace(postID)
	accountID = strings.TrimSpace(accountID)
	var like models.BusinessPostLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("TRIM(LOWER(post_id)) = TRIM(LOWER(?)) AND TRIM(LOWER(account_id)) = TRIM(LOWER(?))", postID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessPost{}).Where("TRIM(LOWER(post_id)) = TRIM(LOWER(?))", postID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.BusinessPostLike{
				PostID:    postID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessPost{}).Where("TRIM(LOWER(post_id)) = TRIM(LOWER(?))", postID).
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
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		var actualCount int64
		if err := tx.Model(&models.BusinessPostComment{}).Where("post_id = ?", comment.PostID).Count(&actualCount).Error; err != nil {
			return err
		}
		return tx.Model(&models.BusinessPost{}).Where("post_id = ?", comment.PostID).
			UpdateColumn("comment_count", actualCount).Error
	})
}

// 3. UPDATE KOMENTAR POSTINGAN
func (r *postInteractionRepository) UpdatePostComment(comment *models.BusinessPostComment) error {
	return r.db.Save(comment).Error
}

// 4. DELETE KOMENTAR POSTINGAN
func (r *postInteractionRepository) DeletePostComment(commentID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var comment models.BusinessPostComment
		if err := tx.Where("comment_id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}

		if err := tx.Where("comment_id = ?", commentID).Delete(&models.BusinessPostCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("reply_to_comment_id = ?", commentID).Delete(&models.BusinessPostComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id = ?", commentID).Delete(&models.BusinessPostComment{}).Error; err != nil {
			return err
		}

		var actualCount int64
		if err := tx.Model(&models.BusinessPostComment{}).Where("post_id = ?", comment.PostID).Count(&actualCount).Error; err != nil {
			return err
		}
		return tx.Model(&models.BusinessPost{}).Where("post_id = ?", comment.PostID).
			UpdateColumn("comment_count", actualCount).Error
	})
}

func (r *postInteractionRepository) GetPostCommentByID(commentID string) (*models.BusinessPostComment, error) {
	var comment models.BusinessPostComment
	err := r.db.Preload("Account").Where("comment_id = ?", commentID).First(&comment).Error
	return &comment, err
}

func (r *postInteractionRepository) GetCommentsByPostID(postID string) ([]models.BusinessPostComment, error) {
	var comments []models.BusinessPostComment
	err := r.db.Preload("Account").Where("post_id = ?", postID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *postInteractionRepository) GetBusinessIDByPostID(postID string) (string, error) {
	var post models.BusinessPost
	err := r.db.Select("business_id").Where("post_id = ?", postID).First(&post).Error
	if err != nil {
		return "", err
	}
	return post.BusinessID, nil
}

func (r *postInteractionRepository) TogglePostCommentLike(commentID, accountID string) (bool, error) {
	commentID = strings.TrimSpace(commentID)
	accountID = strings.TrimSpace(accountID)
	var like models.BusinessPostCommentLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?)) AND TRIM(LOWER(account_id)) = TRIM(LOWER(?))", commentID, accountID).First(&like)

		if result.Error == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessPostComment{}).Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", commentID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
			isLiked = false
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newLike := models.BusinessPostCommentLike{
				CommentID: commentID,
				AccountID: accountID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BusinessPostComment{}).Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", commentID).
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

func (r *postInteractionRepository) GetPostCommentLikes(commentID string) ([]models.BusinessPostCommentLike, error) {
	var likes []models.BusinessPostCommentLike
	err := r.db.Where("TRIM(LOWER(comment_id)) = TRIM(LOWER(?))", strings.TrimSpace(commentID)).Find(&likes).Error
	return likes, err
}

func (r *postInteractionRepository) CreatePostReport(report *models.BusinessPostReport) error {
	return r.db.Create(report).Error
}