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

	var count5 int64
	r.db.Model(&models.ProductComment{}).Where("account_id = ?", accountID).Count(&count5)

	total = count1 + count2 + count3 + count4 + count5
	return total, nil
}

func (r *publicProfileRepository) GetRecentReviews(accountID string, limit int) ([]models.RecentActivityResponse, error) {
	var recentActivities []models.RecentActivityResponse

	// 1. Get recent business reviews (Preload Business untuk nama toko)
	var businessReviews []models.Review
	r.db.Preload("Business").Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Find(&businessReviews)

	for _, review := range businessReviews {
		var bizName string
		if review.Business != nil && review.Business.BusinessName != nil {
			bizName = *review.Business.BusinessName
		}
		if bizName == "" && review.BusinessID != "" {
			var fetchedBiz models.Business
			if err := r.db.Select("business_id, business_name").Unscoped().Where("business_id = ?", review.BusinessID).First(&fetchedBiz).Error; err == nil {
				if fetchedBiz.BusinessName != nil {
					bizName = *fetchedBiz.BusinessName
				}
			}
		}

		recentActivities = append(recentActivities, models.RecentActivityResponse{
			ReviewID:     review.ReviewID,
			ActivityType: "business_review",
			Content:      review.Comment,
			Rating:       float64(review.Rating),
			CreatedAt:    review.CreatedAt,
			BusinessName: bizName,
		})
	}

	// 2. Get recent product reviews (Preload Product untuk nama produk)
	var productReviews []models.ProductReview
	r.db.Preload("Product").Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Find(&productReviews)

	for _, review := range productReviews {
		var prodName string
		if review.Product != nil {
			prodName = review.Product.ProductName
		}
		if prodName == "" && review.ProductID != "" {
			var fetchedProd models.Product
			if err := r.db.Select("product_id, product_name").Unscoped().Where("product_id = ?", review.ProductID).First(&fetchedProd).Error; err == nil {
				prodName = fetchedProd.ProductName
			}
		}

		recentActivities = append(recentActivities, models.RecentActivityResponse{
			ReviewID:     review.ReviewID,
			ActivityType: "product_review",
			Content:      review.Comment,
			Rating:       review.Rating,
			CreatedAt:    review.CreatedAt,
			ProductName:  prodName,
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
