package my_activity

import (
	"sort"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
)

type MyActivityRepository interface {
	GetCustomerInfo(accountID string) (*models.Account, error)
	GetTotalReviews(accountID string) (int64, error)
	GetTotalComments(accountID string) (int64, error)
	
	GetMyComments(accountID string, limit, offset int) ([]models.BusinessPostComment, error)
	GetMyBusinessReviews(accountID string, limit, offset int, reviewType ...string) ([]models.MyActivityReviewResponse, error)
	
	GetMyBookmarks(accountID string, limit, offset int) ([]models.BusinessBookmark, error)
	GetMyWishlists(accountID string, limit, offset int) ([]models.ProductWishlist, error)
	ToggleWishlist(accountID, productID string) (bool, error)
}

type myActivityRepository struct {
	db *gorm.DB
}

func NewMyActivityRepository(db *gorm.DB) MyActivityRepository {
	return &myActivityRepository{db: db}
}

func (r *myActivityRepository) GetCustomerInfo(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Preload("Customer").Where("account_id = ?", accountID).First(&account).Error
	return &account, err
}

func (r *myActivityRepository) GetTotalReviews(accountID string) (int64, error) {
	var businessReviewCount, productReviewCount int64
	r.db.Model(&models.Review{}).Where("account_id = ?", accountID).Count(&businessReviewCount)
	r.db.Model(&models.ProductReview{}).Where("account_id = ?", accountID).Count(&productReviewCount)
	return businessReviewCount + productReviewCount, nil
}

func (r *myActivityRepository) GetTotalComments(accountID string) (int64, error) {
	var commentCount int64
	r.db.Model(&models.BusinessPostComment{}).Where("account_id = ?", accountID).Count(&commentCount)
	return commentCount, nil
}

func (r *myActivityRepository) GetMyComments(accountID string, limit, offset int) ([]models.BusinessPostComment, error) {
	var comments []models.BusinessPostComment
	err := r.db.Preload("Post").Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Offset(offset).Find(&comments).Error
	return comments, err
}

func (r *myActivityRepository) GetMyBusinessReviews(accountID string, limit, offset int, reviewType ...string) ([]models.MyActivityReviewResponse, error) {
	rType := "all"
	if len(reviewType) > 0 && reviewType[0] != "" {
		rType = strings.ToLower(strings.TrimSpace(reviewType[0]))
	}

	fetchLimit := limit + offset
	if fetchLimit < 20 {
		fetchLimit = 20
	}

	var businessReviews []models.Review
	if rType == "all" || rType == "business" || rType == "store" {
		err := r.db.Preload("Business").
			Where("account_id = ?", accountID).
			Order("created_at desc").
			Limit(fetchLimit).
			Find(&businessReviews).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	var productReviews []models.ProductReview
	if rType == "all" || rType == "product" {
		err := r.db.Preload("Business").
			Preload("Product").
			Preload("Comments", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at asc")
			}).
			Where("account_id = ?", accountID).
			Order("created_at desc").
			Limit(fetchLimit).
			Find(&productReviews).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	var unified []models.MyActivityReviewResponse

	for _, br := range businessReviews {
		unified = append(unified, models.MyActivityReviewResponse{
			ReviewID:     br.ReviewID,
			ReviewType:   "business",
			BusinessID:   br.BusinessID,
			AccountID:    br.AccountID,
			Rating:       float64(br.Rating),
			Comment:      br.Comment,
			ReviewImages: br.ReviewImages,
			ReviewVideo:  br.ReviewVideo,
			ReplyComment: br.ReplyComment,
			RepliedAt:    br.RepliedAt,
			LikeCount:    br.LikeCount,
			CommentCount: br.CommentCount,
			CreatedAt:    br.CreatedAt,
			UpdatedAt:    br.UpdatedAt,
			Business:     br.Business,
			Store:        br.Business,
		})
	}

	for _, pr := range productReviews {
		prodID := pr.ProductID
		var replyComment *string
		var repliedAt *time.Time
		for _, c := range pr.Comments {
			if c.AccountID != pr.AccountID {
				replyComment = &c.Content
				repliedAt = &c.CreatedAt
				break
			}
		}
		if replyComment == nil && len(pr.Comments) > 0 {
			replyComment = &pr.Comments[0].Content
			repliedAt = &pr.Comments[0].CreatedAt
		}

		unified = append(unified, models.MyActivityReviewResponse{
			ReviewID:     pr.ReviewID,
			ReviewType:   "product",
			BusinessID:   pr.BusinessID,
			ProductID:    &prodID,
			AccountID:    pr.AccountID,
			Rating:       pr.Rating,
			Comment:      pr.Comment,
			ReviewImages: pr.ReviewImages,
			ReviewVideo:  pr.ReviewVideo,
			ReplyComment: replyComment,
			RepliedAt:    repliedAt,
			LikeCount:    pr.LikeCount,
			CommentCount: pr.CommentCount,
			CreatedAt:    pr.CreatedAt,
			UpdatedAt:    pr.UpdatedAt,
			Business:     pr.Business,
			Store:        pr.Business,
			Product:      pr.Product,
		})
	}

	sort.Slice(unified, func(i, j int) bool {
		return unified[i].CreatedAt.After(unified[j].CreatedAt)
	})

	if offset >= len(unified) {
		return []models.MyActivityReviewResponse{}, nil
	}

	end := offset + limit
	if end > len(unified) {
		end = len(unified)
	}

	return unified[offset:end], nil
}

func (r *myActivityRepository) GetMyBookmarks(accountID string, limit, offset int) ([]models.BusinessBookmark, error) {
	var bookmarks []models.BusinessBookmark
	// In the future, we might want to preload the Business, but GORM might not know the relation unless defined.
	// Since BusinessBookmark has AccountID and BusinessID, we can fetch the businesses in service or preload if set up.
	err := r.db.Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Offset(offset).Find(&bookmarks).Error
	return bookmarks, err
}

func (r *myActivityRepository) GetMyWishlists(accountID string, limit, offset int) ([]models.ProductWishlist, error) {
	var wishlists []models.ProductWishlist
	err := r.db.Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Offset(offset).Find(&wishlists).Error
	return wishlists, err
}

func (r *myActivityRepository) ToggleWishlist(accountID, productID string) (bool, error) {
	var wishlist models.ProductWishlist
	err := r.db.Where("account_id = ? AND product_id = ?", accountID, productID).First(&wishlist).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create
			newWishlist := models.ProductWishlist{
				AccountID: accountID,
				ProductID: productID,
			}
			errCreate := r.db.Create(&newWishlist).Error
			return true, errCreate
		}
		return false, err
	}
	// Delete
	errDelete := r.db.Delete(&wishlist).Error
	return false, errDelete
}
