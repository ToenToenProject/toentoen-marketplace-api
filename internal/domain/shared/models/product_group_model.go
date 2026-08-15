package models

import "time"

type ProductGroup struct {
	ProductGroupID string    `gorm:"primaryKey;column:product_group_id" json:"product_group_id"`
	BusinessID     string    `gorm:"column:business_id;index" json:"business_id"`
	Name           string    `gorm:"column:name" json:"name"`
	DisplayOrder   int       `gorm:"column:display_order;default:0" json:"display_order"`
	IsActive       bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
}

// DTO untuk Input
type CreateGroupInput struct {
	Name         string `json:"name" validate:"required"`
	DisplayOrder int    `json:"display_order"`
}

type UpdateGroupInput struct {
	Name         string `json:"name"`
	DisplayOrder *int   `json:"display_order"`
	IsActive     *bool  `json:"is_active"`
}

// DTO khusus untuk memanipulasi ISI dari Grup (Tambah/Hapus Produk)
type UpdateGroupContentsInput struct {
	AddProductIDs    []string `json:"add_product_ids"`    // ID produk yang mau dimasukkan ke grup ini
	RemoveProductIDs []string `json:"remove_product_ids"` // ID produk yang mau dikeluarkan (menjadi NULL)
}

type GroupOrderItem struct {
	ProductGroupID      string `json:"product_group_id" validate:"required"`
	DisplayOrder int    `json:"display_order" validate:"required"`
}

type ReorderGroupsInput struct {
	Orders []GroupOrderItem `json:"orders" validate:"required,min=1"`
}