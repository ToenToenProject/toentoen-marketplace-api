package my_activity

import (
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
)

type MyActivityRepository interface {
	GetCustomerInfo(accountID string) (*models.Account, error)
	GetTotalReviews(accountID string) (int64, error)
	GetTotalComments(accountID string) (int64, error)
	
	GetMyComments(accountID string, limit, offset int) ([]models.BusinessPostComment, error)
	GetMyBusinessReviews(accountID string, limit, offset int) ([]models.Review, error)
	
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
	r.db.Model(&models.ProductComment{}).Where("account_id = ?", accountID).Count(&productReviewCount)
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

func (r *myActivityRepository) GetMyBusinessReviews(accountID string, limit, offset int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Business").Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Offset(offset).Find(&reviews).Error
	return reviews, err
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
