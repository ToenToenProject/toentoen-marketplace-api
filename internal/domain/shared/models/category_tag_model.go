package models

import "time"

type Category struct {
	CategoryID int64   `gorm:"primaryKey;autoIncrement;column:category_id" json:"category_id"`
	Name       string  `gorm:"not null;column:name" json:"name"`
	IconURL    *string `gorm:"column:icon_url" json:"icon_url"`
	ParentID   *int64  `gorm:"column:parent_id" json:"parent_id"`

	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

type Tag struct {
	TagID      int64   `gorm:"primaryKey;autoIncrement;column:tag_id" json:"tag_id"`
	Name       string  `gorm:"not null;column:name" json:"name"`
	CategoryID *int64  `gorm:"column:category_id" json:"category_id"`
	ParentID   *int64  `gorm:"column:parent_id" json:"parent_id"`

	Children []Tag `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

type UserPreferredCategory struct {
	AccountID  string     `gorm:"primaryKey;column:account_id" json:"account_id"`
	CategoryID string     `gorm:"primaryKey;column:category_id" json:"category_id"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type BusinessCategory struct {
	BusinessID string `gorm:"primaryKey;column:business_id" json:"business_id"`
	CategoryID int64 `gorm:"primaryKey;column:category_id" json:"category_id"`
}

type BusinessTag struct {
	BusinessID string `gorm:"primaryKey;column:business_id" json:"business_id"`
	TagID      int64 `gorm:"primaryKey;column:tag_id" json:"tag_id"`
}