package models

import "time"

type ReferralCode struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`
	Code        string    `gorm:"uniqueIndex;column:code;not null" json:"code"`
	Description *string   `gorm:"column:description" json:"description,omitempty"`
	IsActive    bool      `gorm:"default:true;column:is_active" json:"is_active"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (ReferralCode) TableName() string {
	return "referral_codes"
}
