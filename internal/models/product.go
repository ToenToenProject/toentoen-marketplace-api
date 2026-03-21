package models

import (
	"time"
	"gorm.io/gorm"
)

type Product struct {
	ID int64 `json:"id" gorm:"primaryKey;autoIncrement"`
	BusinessID int64 `json:"business_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Price float64 `json:"price" gorm:"type:decimal(15, 2)"`
	StockStatus string `json:"stock_status" gorm:"default:0"`
	SortOrder int `json:"sort_order" gorm:"default:0"`
	ViewCount int `json:"view_count" gorm:"default:0"`
	AverageRating float64 `json:"average_rating" gorm:"default:0.0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Images []ProductImage `json:"images" gorm:"foreignKey:ProductID"`
	Tags []Tag `json:"tags" gorm:"many2many:product_tags;"`
}

type ProductImage struct {
	ID int64 `json:"id" gorm:"primaryKey"`
	ProductID int64 `json:"product_id"`
	ImageURL string `json:"image_url"`
	SortOrder int `json:"sort_order" gorm:"default:0"`
}

type ProductTag struct {
	ID int64 `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}

type CreateProductInput struct {
	BusinessID int64 `json:"business_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Price float64 `json:"price"`
	StockStatus string `json:"stock_status"`
	SortOrder int `json:"sort_order"`
	Images []string `json:"images"`
	TagIDs []int64 `json:"tag_ids"`
}

func (Product) TableName() string {return "products"}
func (ProductImage) TableName() string {return "product_images"}