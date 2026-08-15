package models

import "time"

type BusinessBookmark struct {
	AccountID  string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	BusinessID string    `gorm:"primaryKey;column:business_id" json:"business_id"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Account  *Account  `gorm:"foreignKey:AccountID;references:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"account,omitempty"`
}

func (BusinessBookmark) TableName() string {
	return "business_bookmarks"
}

type ProductWishlist struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	ProductID string    `gorm:"primaryKey;column:product_id" json:"product_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	Product *Product `gorm:"foreignKey:ProductID;references:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"product,omitempty"`
	Account *Account `gorm:"foreignKey:AccountID;references:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"account,omitempty"`
}

func (ProductWishlist) TableName() string {
	return "product_wishlists"
}
