package models

import (
	"time"

	"gorm.io/datatypes"
)

type UserNotification struct {
	ID                 string         `gorm:"primaryKey;column:id" json:"notification_id"`
	RecipientAccountID string         `gorm:"column:recipient_account_id" json:"recipient_account_id"`
	ActorIDs           datatypes.JSON `gorm:"column:actor_ids;default:'[]'" json:"actor_ids"`
	Type               string         `gorm:"column:type" json:"type"`
	ReferenceID        *string        `gorm:"column:reference_id" json:"reference_id"`
	Title              string         `gorm:"column:title" json:"title"`
	Body               string         `gorm:"column:body" json:"body"`
	Data               datatypes.JSON `gorm:"column:data;default:'{}'" json:"data"`
	IsRead             bool           `gorm:"column:is_read;default:false" json:"is_read"`
	CreatedAt          time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at" json:"updated_at"`

	// Relasi (Opsional, jika ingin Preload nama akun penerima)
	Recipient *Account `gorm:"foreignKey:RecipientAccountID;references:AccountID" json:"-"`
}

func (UserNotification) TableName() string {
	return "user_notifications"
}

// Untuk response API List Notifikasi
type NotificationResponse struct {
	NotificationID string                 `json:"notification_id"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	IsRead         bool                   `json:"is_read"`
	Data           map[string]interface{} `json:"data"`
	TargetType     string                 `json:"target_type,omitempty"`
	TargetID       string                 `json:"target_id,omitempty"`
	ActorCount     int                    `json:"actor_count"`
	CreatedAt      time.Time              `json:"created_at"`
}

