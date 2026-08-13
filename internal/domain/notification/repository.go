package notification

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type NotificationRepository interface {
	CreateNotification(notif *models.UserNotification) error
	GetUnreadNotificationByRef(recipientID, notifType, refID string) (*models.UserNotification, error)
	UpdateNotification(notif *models.UserNotification) error
	GetMyNotifications(accountID string, page, limit int) ([]models.UserNotification, int64, error)
	MarkAsRead(accountID, notifID string) error
	MarkAllAsRead(accountID string) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db}
}

func (r *notificationRepository) CreateNotification(notif *models.UserNotification) error {
	return r.db.Create(notif).Error
}

func (r *notificationRepository) GetUnreadNotificationByRef(recipientID, notifType, refID string) (*models.UserNotification, error) {
	var notif models.UserNotification
	err := r.db.Where("recipient_account_id = ? AND type = ? AND reference_id = ? AND is_read = false", recipientID, notifType, refID).
		Order("updated_at DESC").
		First(&notif).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return &notif, nil
}

func (r *notificationRepository) UpdateNotification(notif *models.UserNotification) error {
	return r.db.Save(notif).Error
}

func (r *notificationRepository) GetMyNotifications(accountID string, page, limit int) ([]models.UserNotification, int64, error) {
	var notifications []models.UserNotification
	var total int64

	query := r.db.Model(&models.UserNotification{}).Where("recipient_account_id = ?", accountID)
	
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&notifications).Error

	return notifications, total, err
}

func (r *notificationRepository) MarkAsRead(accountID, notifID string) error {
	result := r.db.Model(&models.UserNotification{}).
		Where("id = ? AND recipient_account_id = ?", notifID, accountID).
		Update("is_read", true)
		
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("notification not found or you don't have access")
	}
	return nil
}

func (r *notificationRepository) MarkAllAsRead(accountID string) error {
	return r.db.Model(&models.UserNotification{}).
		Where("recipient_account_id = ? AND is_read = false", accountID).
		Update("is_read", true).Error
}
