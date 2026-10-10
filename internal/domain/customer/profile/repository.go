package profile

import (
	"sort"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type PublicProfileRepository interface {
	GetAccountInfo(accountID string) (*models.Account, error)
	GetTotalCommentsAndReviews(accountID string) (int64, error)
	GetRecentReviews(accountID string, limit int) ([]models.RecentActivityResponse, error)
}

type publicProfileRepository struct {
	db *gorm.DB
}

func NewPublicProfileRepository(db *gorm.DB) PublicProfileRepository {
	return &publicProfileRepository{db}
}

func (r *publicProfileRepository) GetAccountInfo(accountID string) (*models.Account, error) {
	var account models.Account
	if err := r.db.Preload("Customer").Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *publicProfileRepository) GetTotalCommentsAndReviews(accountID string) (int64, error) {
	var total int64

	var count1 int64
	r.db.Model(&models.Review{}).Where("account_id = ?", accountID).Count(&count1)

	var count2 int64
	r.db.Model(&models.ProductReview{}).Where("account_id = ?", accountID).Count(&count2)

	var count3 int64
	r.db.Model(&models.BusinessReviewComment{}).Where("account_id = ?", accountID).Count(&count3)

	var count4 int64
	r.db.Model(&models.BusinessPostComment{}).Where("account_id = ?", accountID).Count(&count4)

	total = count1 + count2 + count3 + count4
	return total, nil
}

func (r *publicProfileRepository) GetRecentReviews(accountID string, limit int) ([]models.RecentActivityResponse, error) {
	var recentActivities []models.RecentActivityResponse

	// Get recent business reviews (Preload Business untuk nama toko)
	var businessReviews []models.Review
	r.db.Preload("Business").Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Find(&businessReviews)

	for _, review := range businessReviews {
		var bizName string
		if review.Business != nil && review.Business.BusinessName != nil {
			bizName = *review.Business.BusinessName
		}
		bizObj := review.Business
		if bizName == "" && review.BusinessID != "" {
			var fetchedBiz models.Business
			if err := r.db.Select("business_id, business_name, logo_url").Unscoped().Where("business_id = ?", review.BusinessID).First(&fetchedBiz).Error; err == nil {
				if fetchedBiz.BusinessName != nil {
					bizName = *fetchedBiz.BusinessName
				}
				bizObj = &fetchedBiz
			}
		}

		recentActivities = append(recentActivities, models.RecentActivityResponse{
			ReviewID:     review.ReviewID,
			ActivityType: "business_review",
			BusinessID:   review.BusinessID,
			BusinessName: bizName,
			StoreName:    bizName,
			Rating:       float64(review.Rating),
			Content:      review.Comment,
			MediaURLs:    review.ReviewImages,
			ReviewImages: review.ReviewImages,
			CreatedAt:    review.CreatedAt,
			Business:     bizObj,
			Store:        bizObj,
		})
	}

	// Get recent product reviews (Preload Business dan Product)
	var productReviews []models.ProductReview
	r.db.Preload("Business").Preload("Product").Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Find(&productReviews)

	for _, review := range productReviews {
		bizID := review.BusinessID
		if bizID == "" && review.Product != nil {
			bizID = review.Product.BusinessID
		}

		var bizName string
		if review.Business != nil && review.Business.BusinessName != nil {
			bizName = *review.Business.BusinessName
		}

		var prodName string
		if review.Product != nil {
			prodName = review.Product.ProductName
		}

		bizObj := review.Business
		if bizName == "" && bizID != "" {
			var fetchedBiz models.Business
			if err := r.db.Select("business_id, business_name, logo_url").Unscoped().Where("business_id = ?", bizID).First(&fetchedBiz).Error; err == nil {
				if fetchedBiz.BusinessName != nil {
					bizName = *fetchedBiz.BusinessName
				}
				bizObj = &fetchedBiz
			}
		}

		prodObj := review.Product
		if prodName == "" && review.ProductID != "" {
			var fetchedProd models.Product
			if err := r.db.Select("product_id, business_id, product_name").Unscoped().Where("product_id = ?", review.ProductID).First(&fetchedProd).Error; err == nil {
				prodName = fetchedProd.ProductName
				prodObj = &fetchedProd
				if bizID == "" {
					bizID = fetchedProd.BusinessID
				}
			}
		}

		recentActivities = append(recentActivities, models.RecentActivityResponse{
			ReviewID:     review.ReviewID,
			ActivityType: "product_review",
			BusinessID:   bizID,
			BusinessName: bizName,
			StoreName:    bizName,
			ProductID:    review.ProductID,
			ProductName:  prodName,
			Rating:       review.Rating,
			Content:      review.Comment,
			MediaURLs:    review.ReviewImages,
			ReviewImages: review.ReviewImages,
			CreatedAt:    review.CreatedAt,
			Business:     bizObj,
			Store:        bizObj,
			Product:      prodObj,
		})
	}

	// Urutkan berdasarkan waktu terbaru
	sort.Slice(recentActivities, func(i, j int) bool {
		return recentActivities[i].CreatedAt.After(recentActivities[j].CreatedAt)
	})

	if len(recentActivities) > limit {
		recentActivities = recentActivities[:limit]
	}

	return recentActivities, nil
}
