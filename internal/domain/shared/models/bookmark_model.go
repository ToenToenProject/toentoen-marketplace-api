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

type ProductWishlist struct {
	AccountID  string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	ProductID  string    `gorm:"primaryKey;column:product_id" json:"product_id"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ProductWishlist) TableName() string {
	return "product_wishlists"
}
