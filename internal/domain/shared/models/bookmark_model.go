package models

import "time"

type BusinessBookmark struct {
	AccountID  string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	BusinessID string    `gorm:"primaryKey;column:business_id" json:"business_id"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (BusinessBookmark) TableName() string {
	return "business_bookmarks"
}
