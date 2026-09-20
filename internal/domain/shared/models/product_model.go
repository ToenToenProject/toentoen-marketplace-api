package models

import (
	"time"

	"gorm.io/gorm"
)

// ==========================================
// 1. ENTITAS DATABASE (GORM MODELS)
// ==========================================

type Product struct {
	ProductID      string  `gorm:"primaryKey;column:product_id" json:"product_id"`
	BusinessID     string  `gorm:"column:business_id;index" json:"business_id"`
	ProductGroupID *string `gorm:"column:product_group_id;index" json:"product_group_id"` // Pointer karena bisa NULL

	// Serializer JSON otomatis dari GORM
	ProductPreview     []string `gorm:"column:product_preview;type:jsonb;serializer:json;default:'[]'" json:"product_preview"`
	ProductName        string   `gorm:"column:product_name" json:"product_name"`
	ProductOrigin      string   `gorm:"column:product_origin;default:'domestic'" json:"product_origin"`
	OriginCountry      *string  `gorm:"column:origin_country" json:"origin_country,omitempty"`
	ProductDescription string   `gorm:"column:product_description" json:"product_description"`

	Price           float64    `gorm:"column:price" json:"price"`
	DiscountType    string     `gorm:"column:discount_type;default:'nominal'" json:"discount_type"`
	Discount        float64    `gorm:"column:discount;default:0" json:"discount"`
	MaxDiscount     *float64   `gorm:"column:max_discount" json:"max_discount,omitempty"`
	DiscountStartAt *time.Time `gorm:"column:discount_start_at" json:"discount_start_at"`
	DiscountEndAt   *time.Time `gorm:"column:discount_end_at" json:"discount_end_at"`

	// 🪄 KOLOM VIRTUAL HASIL SUBQUERY (Aman, tidak memicu error migrasi/gorm insert)
	HasVariants bool    `gorm:"->;column:has_variants" json:"has_variants"`
	MinPrice    float64 `gorm:"->;column:min_price" json:"min_price"`
	MaxPrice    float64 `gorm:"->;column:max_price" json:"max_price"`

	StockType         string `gorm:"column:stock_type;default:'in_stock'" json:"stock_type"`
	StockQuantity     int    `gorm:"column:stock_quantity;default:0" json:"stock_quantity"`
	LowStockThreshold int    `gorm:"column:low_stock_threshold;default:5" json:"low_stock_threshold"`

	Tags      []string `gorm:"column:tags;type:jsonb;serializer:json;default:'[]'" json:"tags"`
	IsActive  bool     `gorm:"column:is_active;default:true" json:"is_active"`
	SortOrder int      `gorm:"column:sort_order;default:0" json:"sort_order"`

	ViewCount     int     `gorm:"column:view_count;default:0" json:"view_count"`
	SoldCount     int     `gorm:"column:sold_count;default:0" json:"sold_count"`
	AverageRating float64 `gorm:"column:average_rating;default:0" json:"average_rating"`
	ReviewCount   int     `gorm:"column:review_count;default:0" json:"review_count"`
	LikeCount     int     `gorm:"column:like_count;default:0" json:"like_count"`

	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"` // Disembunyikan dari JSON response

	// Relasi GORM (Otomatis ditarik saat Preload)
	Business *Business        `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Group    *ProductGroup    `gorm:"foreignKey:ProductGroupID;references:ProductGroupID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"product_group,omitempty"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID;references:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"variants,omitempty"`
}

type ProductVariant struct {
	VariantID       string    `gorm:"primaryKey;column:variant_id" json:"variant_id"`
	ProductID       string    `gorm:"column:product_id;index" json:"product_id"`
	SKU             string    `gorm:"column:sku" json:"sku"`
	VariantName     string    `gorm:"column:variant_name" json:"variant_name"`
	AdditionalPrice float64   `gorm:"column:additional_price;default:0" json:"additional_price"`
	StockQuantity   int       `gorm:"column:stock_quantity;default:0" json:"stock_quantity"`
	IsActive        bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// ==========================================
// DTO UNTUK PRODUCT VARIANTS
// ==========================================

type CreateVariantInput struct {
	SKU             string  `json:"sku"`
	VariantName     string  `json:"variant_name" validate:"required"`
	AdditionalPrice float64 `json:"additional_price"`
	StockQuantity   int     `json:"stock_quantity"`
	IsActive        *bool   `json:"is_active"`
}

type UpdateVariantInput struct {
	SKU             *string  `json:"sku"`
	VariantName     *string  `json:"variant_name"`
	AdditionalPrice *float64 `json:"additional_price"`
	StockQuantity   *int     `json:"stock_quantity"`
	IsActive        *bool    `json:"is_active"`
}

// ==========================================
// 2. DTO UNTUK INPUT (REQUEST)
// ==========================================

type CreateProductInput struct {
	ProductGroupID     *string    `json:"product_group_id"`
	ProductPreview     []string   `json:"product_preview" validate:"max=10"`
	ProductName        string     `json:"product_name" validate:"required"`
	ProductOrigin      string     `json:"product_origin"`
	OriginCountry      *string    `json:"origin_country"`
	ProductDescription string     `json:"product_description" validate:"max=300"`
	Price              float64    `json:"price" validate:"required,min=0"`
	DiscountType       *string    `json:"discount_type"` // "nominal" atau "percentage"
	Discount           float64    `json:"discount"`
	MaxDiscount        *float64   `json:"max_discount"`  // Maksimal potongan dalam Rupiah (opsional)
	DiscountStartAt    *time.Time `json:"discount_start_at"`
	DiscountEndAt      *time.Time `json:"discount_end_at"`

	StockType         string `json:"stock_type"` // in_stock, out_of_stock, limited
	StockQuantity     *int   `json:"stock_quantity"`
	LowStockThreshold *int   `json:"low_stock_threshold"`

	Tags     []string `json:"tags"`
	IsActive *bool    `json:"is_active"`
}

type UpdateProductInput struct {
	ProductGroupID     *string    `json:"product_group_id"`
	ProductPreview     *[]string  `json:"product_preview" validate:"omitempty,max=10"`
	ProductName        *string    `json:"product_name"`
	ProductOrigin      *string    `json:"product_origin"`
	OriginCountry      *string    `json:"origin_country"`
	ProductDescription *string    `json:"product_description" validate:"omitempty,max=300"`
	Price              *float64   `json:"price"`
	DiscountType       *string    `json:"discount_type"`
	Discount           *float64   `json:"discount"`
	MaxDiscount        *float64   `json:"max_discount"`
	DiscountStartAt    *time.Time `json:"discount_start_at"`
	DiscountEndAt      *time.Time `json:"discount_end_at"`
	StockType          *string    `json:"stock_type"`
	StockQuantity      *int       `json:"stock_quantity"`
	LowStockThreshold  *int       `json:"low_stock_threshold"`
	IsActive           *bool      `json:"is_active"`
}

// ==========================================
// 3. DTO UNTUK OUTPUT (RESPONSE CATALOG & PAGINASI)
// ==========================================

type PaginationMeta struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

type ProductSummary struct {
	TotalAll       int64 `json:"total_all"`
	TotalInStock   int64 `json:"total_in_stock"`
	TotalPreOrder  int64 `json:"total_pre_order"`
	TotalLimited   int64 `json:"total_limited"`
	TotalOutOfStock int64 `json:"total_out_of_stock"`
}



type ProductListItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	ImageURL    string `json:"image_url"`
	HasVariants bool   `json:"has_variants"`

	// HARGA CORET (Sebelum Diskon)
	OriginalMinPrice float64 `json:"original_min_price"`
	OriginalMaxPrice float64 `json:"original_max_price"`

	// HARGA UTAMA / BAYAR (Setelah Diskon)
	FinalMinPrice float64 `json:"final_min_price"`
	FinalMaxPrice float64 `json:"final_max_price"`

	// INFO DISKON
	IsDiscounted    bool       `json:"is_discounted"`
	DiscountType    string     `json:"discount_type"`          // "nominal" atau "percentage"
	Discount        float64    `json:"discount"`               // Nilai diskon input (misal: 10000 atau 20)
	DiscountAmount  float64    `json:"discount_amount"`        // Potongan harga riil dalam Rupiah setelah max_discount
	DiscountPct     float64    `json:"discount_pct"`           // Persentase diskon (misal: 20 = 20%)
	MaxDiscount     *float64   `json:"max_discount,omitempty"` // Batas maksimal diskon jika ada
	DiscountStartAt *time.Time `json:"discount_start_at"`      // null jika tidak ada batas waktu mulai
	DiscountEndAt   *time.Time `json:"discount_end_at"`        // null jika tidak ada batas waktu akhir

	StockType     string   `json:"stock_type"`
	ProductOrigin string   `json:"product_origin,omitempty"`
	OriginCountry *string  `json:"origin_country,omitempty"`
	Rating        float64  `json:"rating"`
	LikeCount     int      `json:"like_count"`
	ReviewCount   int      `json:"review_count"`

	// Digunakan jika detail varian ingin disertakan pada response tertentu
	Variants []ProductVariant `json:"variants,omitempty"`
}

type GetProductsResponse struct {
	Products   []ProductListItem `json:"products"`
	Summary    ProductSummary    `json:"summary"`
	Pagination PaginationMeta    `json:"pagination"`
}

// Taruh ini di internal/models/models.go
type ProductDetailResponse struct {
	ProductID          string   `json:"product_id"`
	BusinessID         string   `json:"business_id"`
	ProductGroupID     *string  `json:"product_group_id"`
	ProductPreview     []string `json:"product_preview"`
	ProductName        string   `json:"product_name"`
	ProductOrigin      string   `json:"product_origin"`
	OriginCountry      *string  `json:"origin_country,omitempty"`
	ProductDescription string   `json:"product_description"`

	Price           float64    `json:"price"` // Harga dasar di DB (opsional untuk referensi)
	DiscountType    string     `json:"discount_type"`
	Discount        float64    `json:"discount"`
	DiscountAmount  float64    `json:"discount_amount"`
	DiscountPct     float64    `json:"discount_pct"`
	MaxDiscount     *float64   `json:"max_discount,omitempty"`
	DiscountStartAt *time.Time `json:"discount_start_at"`
	DiscountEndAt   *time.Time `json:"discount_end_at"`

	// --- 🪄 VARIABEL HASIL KALKULASI DISKON & VARIAN ---
	HasVariants      bool    `json:"has_variants"`
	OriginalMinPrice float64 `json:"original_min_price"` // Harga coret termurah
	OriginalMaxPrice float64 `json:"original_max_price"` // Harga coret termahal
	FinalMinPrice    float64 `json:"final_min_price"`    // Harga bayar termurah
	FinalMaxPrice    float64 `json:"final_max_price"`    // Harga bayar termahal
	IsDiscounted     bool    `json:"is_discounted"`

	StockType         string `json:"stock_type"`
	StockQuantity     int    `json:"stock_quantity"`
	LowStockThreshold int    `json:"low_stock_threshold"`

	Tags         []string         `json:"tags"`
	IsActive     bool             `json:"is_active"`
	Rating       float64          `json:"rating"`
	LikeCount    int              `json:"like_count"`
	ReviewCount  int              `json:"review_count"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Group        *ProductGroup    `json:"product_group,omitempty"`
	Variants     []ProductVariant `json:"variants"`
}
