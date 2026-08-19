package profile

import (
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

	// Get recent business reviews
	var businessReviews []models.Review
	r.db.Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Find(&businessReviews)

	for _, review := range businessReviews {
		recentActivities = append(recentActivities, models.RecentActivityResponse{
			ActivityType: "business_review",
			Content:      review.Comment,
			MediaURLs:    review.ReviewImages,
			CreatedAt:    review.CreatedAt,
		})
	}

	// Get recent product reviews
	var productReviews []models.ProductReview
	r.db.Where("account_id = ?", accountID).Order("created_at desc").Limit(limit).Find(&productReviews)

	for _, review := range productReviews {
		recentActivities = append(recentActivities, models.RecentActivityResponse{
			ActivityType: "product_review",
			Content:      review.Comment,
			MediaURLs:    review.ReviewImages,
			CreatedAt:    review.CreatedAt,
		})
	}

	// Sort and limit again in memory
	if len(recentActivities) > limit {
		// Bubble sort since we just want the top 2 and the array size is at most 2*limit
		for i := 0; i < len(recentActivities)-1; i++ {
			for j := i + 1; j < len(recentActivities); j++ {
				if recentActivities[j].CreatedAt.After(recentActivities[i].CreatedAt) {
					temp := recentActivities[i]
					recentActivities[i] = recentActivities[j]
					recentActivities[j] = temp
				}
			}
		}
		recentActivities = recentActivities[:limit]
	}

	return recentActivities, nil
}
