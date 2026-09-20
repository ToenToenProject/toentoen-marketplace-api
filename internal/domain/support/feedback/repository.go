package feedback

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type FeedbackRepository interface {
	GetAccountByID(accountID string) (*models.Account, error)
}

type feedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) FeedbackRepository {
	return &feedbackRepository{db: db}
}

func (r *feedbackRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	if err := r.db.Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
