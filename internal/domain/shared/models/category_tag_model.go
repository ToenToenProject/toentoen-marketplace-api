package models

import "time"

type Category struct {
	CategoryID int64  `gorm:"primaryKey;autoIncrement;column:category_id" json:"category_id"`
	Slug       string `gorm:"column:slug;uniqueIndex" json:"slug,omitempty"`
	Name       string `gorm:"not null;column:name" json:"name"`
	ParentID   *int64 `gorm:"column:parent_id" json:"parent_id"`
	SortOrder  int    `gorm:"column:sort_order;default:0" json:"sort_order"`
	IsActive   bool   `gorm:"column:is_active;default:true" json:"is_active"`

	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (Category) TableName() string {
	return "business_categories"
}

type TagGroup struct {
	GroupID   int64  `gorm:"primaryKey;autoIncrement;column:group_id" json:"group_id"`
	Slug      string `gorm:"column:slug;uniqueIndex;not null" json:"slug"`
	Name      string `gorm:"column:name;not null" json:"name"`
	TagType   string `gorm:"column:tag_type;default:'characteristic'" json:"tag_type"`
	SortOrder int    `gorm:"column:sort_order;default:0" json:"sort_order"`
	IsActive  bool   `gorm:"column:is_active;default:true" json:"is_active"`
}

func (TagGroup) TableName() string {
	return "tag_groups"
}

type Tag struct {
	TagID      int64  `gorm:"primaryKey;autoIncrement;column:tag_id" json:"tag_id"`
	Slug       string `gorm:"column:slug;uniqueIndex" json:"slug,omitempty"`
	Name       string `gorm:"not null;column:name" json:"name"`
	TagType    string `gorm:"column:tag_type;default:'characteristic'" json:"tag_type,omitempty"`
	GroupID    *int64 `gorm:"column:group_id;index" json:"group_id,omitempty"`
	GroupSlug  string `gorm:"-" json:"group_slug,omitempty"`
	CategoryID *int64 `gorm:"column:category_id" json:"category_id"`
	ParentID   *int64 `gorm:"column:parent_id" json:"parent_id"`
	IsGlobal   bool   `gorm:"column:is_global;default:false" json:"is_global"`
	SortOrder  int    `gorm:"column:sort_order;default:0" json:"sort_order"`
	IsActive   bool   `gorm:"column:is_active;default:true" json:"is_active"`

	Children []Tag `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (Tag) TableName() string {
	return "business_tags"
}

type CategoryTagRule struct {
	CategoryID int64 `gorm:"primaryKey;column:category_id" json:"category_id"`
	TagID      int64 `gorm:"primaryKey;column:tag_id" json:"tag_id"`
	SortOrder  int   `gorm:"column:sort_order;default:0" json:"sort_order"`
	IsActive   bool  `gorm:"column:is_active;default:true" json:"is_active"`
}

func (CategoryTagRule) TableName() string {
	return "category_tag_rules"
}

type UserPreferredCategory struct {
	AccountID  string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CategoryID string    `gorm:"primaryKey;column:category_id" json:"category_id"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (UserPreferredCategory) TableName() string {
	return "customer_preferred_categories"
}

type BusinessHasCategory struct {
	BusinessID string `gorm:"primaryKey;column:business_id" json:"business_id"`
	CategoryID int64 `gorm:"primaryKey;column:category_id" json:"category_id"`
}

func (BusinessHasCategory) TableName() string {
	return "business_has_categories"
}

type BusinessHasTag struct {
	BusinessID string `gorm:"primaryKey;column:business_id" json:"business_id"`
	CategoryID int64  `gorm:"primaryKey;column:category_id;default:0" json:"category_id"`
	TagID      int64  `gorm:"primaryKey;column:tag_id" json:"tag_id"`
}

func (BusinessHasTag) TableName() string {
	return "business_has_tags"
}