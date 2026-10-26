package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ==========================================
// 1. ENTITAS DATABASE (GORM MODELS)
// ==========================================

const (
	PromotionStatusScheduled = "scheduled"
	PromotionStatusActive    = "active"
	PromotionStatusEnded     = "ended"
	PromotionStatusStopped   = "stopped"

	PromotionDiscountSourceNew  = "promotion"
	PromotionDiscountSourceKept = "existing_kept"

	PromotionEventImpression = "impression"
	PromotionEventClick      = "click"

	PromotionEventSourcePost            = "post"
	PromotionEventSourceBusinessProfile = "business_profile"
	PromotionEventSourceNotification    = "notification"
	PromotionEventSourceProductGroup    = "product_group"

	PromotionTypeCatalog    = "catalog"
	PromotionTypePost       = "post"
	PromotionTypePerProduct = "per_product"
	PromotionTypeBundling   = "bundling"
)

type Promotion struct {
	PromotionID        string     `gorm:"primaryKey;column:promotion_id" json:"promotion_id"`
	BusinessID         string     `gorm:"column:business_id;index" json:"business_id"`
	Caption            string     `gorm:"column:caption;size:150" json:"caption"`
	MainProductID      string     `gorm:"column:main_product_id" json:"main_product_id"`
	PostID             *string    `gorm:"column:post_id" json:"post_id,omitempty"`
	StartAt            time.Time  `gorm:"column:start_at" json:"start_at"`
	EndAt              time.Time  `gorm:"column:end_at" json:"end_at"`
	Status             string     `gorm:"column:status;default:'scheduled'" json:"status"`
	NotifiedAt         *time.Time `gorm:"column:notified_at" json:"notified_at,omitempty"`
	StoppedAt          *time.Time `gorm:"column:stopped_at" json:"stopped_at,omitempty"`
	ImpressionCount    int        `gorm:"column:impression_count;default:0" json:"impression_count"`
	ClickCount         int        `gorm:"column:click_count;default:0" json:"click_count"`
	CreatedBy          string     `gorm:"column:created_by" json:"created_by"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`

	// Kolom khusus tipe Bundling
	PromotionType      string   `gorm:"column:promotion_type;default:'per_product'" json:"promotion_type"`
	PackageName        *string  `gorm:"column:package_name" json:"package_name,omitempty"`
	PackagePrice       *float64 `gorm:"column:package_price" json:"package_price,omitempty"`
	PackageImage       *string  `gorm:"column:package_image" json:"package_image,omitempty"`
	OriginalTotalPrice *float64 `gorm:"column:original_total_price" json:"original_total_price,omitempty"`
	SavingsAmount      *float64 `gorm:"column:savings_amount" json:"savings_amount,omitempty"`
	SavingsPercentage  *float64 `gorm:"column:savings_percentage" json:"savings_percentage,omitempty"`

	// Relasi
	Business    *Business              `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	MainProduct *Product               `gorm:"foreignKey:MainProductID;references:ProductID" json:"main_product,omitempty"`
	Post        *BusinessPost          `gorm:"foreignKey:PostID;references:PostID" json:"post,omitempty"`
	Products    []PromotionProduct     `gorm:"foreignKey:PromotionID;references:PromotionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"products,omitempty"`
	BundleItems []PromotionBundleItem `gorm:"foreignKey:PromotionID;references:PromotionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"bundle_items,omitempty"`
}

func (Promotion) TableName() string {
	return "promotions"
}

type PromotionBundleItem struct {
	PromotionID     string    `gorm:"primaryKey;column:promotion_id" json:"promotion_id"`
	ProductID       string    `gorm:"primaryKey;column:product_id" json:"product_id"`
	Quantity        int       `gorm:"column:quantity;default:1" json:"quantity"`
	ItemNormalPrice float64   `gorm:"column:item_normal_price;default:0" json:"item_normal_price"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`

	// Relasi
	Promotion *Promotion `gorm:"foreignKey:PromotionID;references:PromotionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"promotion,omitempty"`
	Product   *Product   `gorm:"foreignKey:ProductID;references:ProductID" json:"product,omitempty"`
}

func (PromotionBundleItem) TableName() string {
	return "promotion_bundle_items"
}

type PromotionProduct struct {
	PromotionID string `gorm:"primaryKey;column:promotion_id" json:"promotion_id"`
	ProductID   string `gorm:"primaryKey;column:product_id" json:"product_id"`

	DiscountSource string   `gorm:"column:discount_source;default:'promotion'" json:"discount_source"`
	DiscountType   string   `gorm:"column:discount_type" json:"discount_type"`
	Discount       float64  `gorm:"column:discount;default:0" json:"discount"`
	MaxDiscount    *float64 `gorm:"column:max_discount" json:"max_discount,omitempty"`

	// Cadangan diskon lama untuk rollback
	HadPreviousDiscount bool       `gorm:"column:had_previous_discount;default:false" json:"had_previous_discount"`
	PrevDiscountType    *string    `gorm:"column:prev_discount_type" json:"prev_discount_type,omitempty"`
	PrevDiscount        *float64   `gorm:"column:prev_discount" json:"prev_discount,omitempty"`
	PrevMaxDiscount     *float64   `gorm:"column:prev_max_discount" json:"prev_max_discount,omitempty"`
	PrevDiscountStartAt *time.Time `gorm:"column:prev_discount_start_at" json:"prev_discount_start_at,omitempty"`
	PrevDiscountEndAt   *time.Time `gorm:"column:prev_discount_end_at" json:"prev_discount_end_at,omitempty"`

	// Snapshot harga untuk riwayat promosi
	SnapshotOriginalPrice float64 `gorm:"column:snapshot_original_price;default:0" json:"snapshot_original_price"`
	SnapshotFinalPrice    float64 `gorm:"column:snapshot_final_price;default:0" json:"snapshot_final_price"`

	// Relasi
	Product *Product `gorm:"foreignKey:ProductID;references:ProductID" json:"product,omitempty"`
}

func (PromotionProduct) TableName() string {
	return "promotion_products"
}

type PromotionEvent struct {
	EventID     string    `gorm:"primaryKey;column:event_id" json:"event_id"`
	PromotionID string    `gorm:"column:promotion_id;index" json:"promotion_id"`
	ProductID   *string   `gorm:"column:product_id" json:"product_id,omitempty"`
	AccountID   *string   `gorm:"column:account_id" json:"account_id,omitempty"`
	EventType   string    `gorm:"column:event_type;index" json:"event_type"` // "impression" | "click"
	Source      string    `gorm:"column:source" json:"source"`               // "post" | "business_profile" | "notification" | "product_group"
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

func (PromotionEvent) TableName() string {
	return "promotion_events"
}

// ==========================================
// 2. DTO UNTUK REQUEST & RESPONSE (MERCHANT & CUSTOMER)
// ==========================================

type BusinessActivePromotionItem struct {
	PromotionID   string    `json:"promotion_id"`
	Title         string    `json:"title"`
	Caption       string    `json:"caption"`
	BannerURL     *string   `json:"banner_url"`
	PromotionType string    `json:"promotion_type,omitempty"`
	DiscountPct   *float64  `json:"discount_pct,omitempty"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
}

type ActivePromotionInfo struct {
	PromotionID   string `json:"promotion_id"`
	PromotionType string `json:"promotion_type"` // "per_product" | "bundling"
	Status        string `json:"status"`         // "active" | "scheduled"
}

type EligibleProductItem struct {
	ProductID              string                `json:"product_id"`
	ProductName            string                `json:"product_name"`
	ImageURL               string                `json:"image_url"`
	Price                  float64               `json:"price"`
	IsEligible             bool                  `json:"is_eligible"`
	IneligibleReason       string                `json:"ineligible_reason,omitempty"`
	HasExistingDiscount    bool                  `json:"has_existing_discount"`
	CurrentDiscountType    string                `json:"current_discount_type,omitempty"`
	CurrentDiscount        float64               `json:"current_discount,omitempty"`
	CurrentMaxDiscount     *float64              `json:"current_max_discount,omitempty"`
	CurrentDiscountStartAt *time.Time            `json:"current_discount_start_at,omitempty"`
	CurrentDiscountEndAt   *time.Time            `json:"current_discount_end_at,omitempty"`
	InPerProductPromotion  bool                  `json:"in_per_product_promotion"`
	InBundlingPromotion    bool                  `json:"in_bundling_promotion"`
	ActivePromotions       []ActivePromotionInfo `json:"active_promotions,omitempty"`
}

type CreatePromotionItemInput struct {
	ProductID            string   `json:"product_id" validate:"required"`
	DiscountType         *string  `json:"discount_type"` // "percentage" atau "nominal"
	Discount             *float64 `json:"discount"`
	MaxDiscount          *float64 `json:"max_discount"`
	KeepExistingDiscount *bool    `json:"keep_existing_discount"`
}

// Helper parser waktu fleksibel (RFC3339, +07:00, Z, format SQL, datetime-local, timestamp)
func parseFlexibleTime(val string) (time.Time, error) {
	val = strings.TrimSpace(val)
	if val == "" || val == "null" {
		return time.Time{}, nil
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999Z07:00",
		"2006-01-02T15:04:05.99Z07:00",
		"2006-01-02T15:04:05.9Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05+07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	loc := time.FixedZone("WIB", 7*3600)
	for _, layout := range formats {
		if t, err := time.ParseInLocation(layout, val, loc); err == nil {
			return t, nil
		}
	}

	if epoch, err := strconv.ParseInt(val, 10, 64); err == nil {
		if epoch > 1e11 {
			return time.UnixMilli(epoch), nil
		}
		return time.Unix(epoch, 0), nil
	}

	return time.Parse(time.RFC3339, val)
}

type CreatePromotionInput struct {
	Caption       string                     `json:"caption"`
	MainProductID string                     `json:"main_product_id"`
	StartAt       time.Time                  `json:"start_at"`
	EndAt         time.Time                  `json:"end_at"`
	Products      []CreatePromotionItemInput `json:"products" validate:"required,min=1,max=20"`
}

func (c *CreatePromotionInput) UnmarshalJSON(data []byte) error {
	type Alias CreatePromotionInput
	aux := &struct {
		RawStartAt interface{} `json:"start_at"`
		RawEndAt   interface{} `json:"end_at"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.RawStartAt != nil {
		startStr := strings.Trim(fmt.Sprintf("%v", aux.RawStartAt), "\"")
		if parsed, err := parseFlexibleTime(startStr); err == nil {
			c.StartAt = parsed
		}
	}

	if aux.RawEndAt != nil {
		endStr := strings.Trim(fmt.Sprintf("%v", aux.RawEndAt), "\"")
		if parsed, err := parseFlexibleTime(endStr); err == nil {
			c.EndAt = parsed
		}
	}

	// Auto-fallback main_product_id jika kosong tapi ada produk yang dipilih
	if strings.TrimSpace(c.MainProductID) == "" && len(c.Products) > 0 {
		c.MainProductID = c.Products[0].ProductID
	}

	// Auto-fallback caption jika kosong
	if strings.TrimSpace(c.Caption) == "" {
		c.Caption = "Promo Spesial Toko"
	}

	return nil
}

type BundlingProductItemInput struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"omitempty,min=1"`
}

type CreateBundlingPromotionInput struct {
	PackageName  string                     `json:"package_name" validate:"required,min=3,max=100"`
	Caption      string                     `json:"caption"`
	ImageURL     *string                    `json:"image_url"`
	PackagePrice float64                    `json:"package_price" validate:"required,gt=0"`
	StartAt      time.Time                  `json:"start_at"`
	EndAt        *time.Time                 `json:"end_at"`
	DurationDays *int                       `json:"duration_days" validate:"omitempty,min=1,max=90"`
	Products     []BundlingProductItemInput `json:"products" validate:"required,min=2,max=20"`
}

func (c *CreateBundlingPromotionInput) UnmarshalJSON(data []byte) error {
	type Alias CreateBundlingPromotionInput
	aux := &struct {
		RawStartAt interface{} `json:"start_at"`
		RawEndAt   interface{} `json:"end_at"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.RawStartAt != nil {
		startStr := strings.Trim(fmt.Sprintf("%v", aux.RawStartAt), "\"")
		if parsed, err := parseFlexibleTime(startStr); err == nil {
			c.StartAt = parsed
		}
	}

	if aux.RawEndAt != nil {
		endStr := strings.Trim(fmt.Sprintf("%v", aux.RawEndAt), "\"")
		if parsed, err := parseFlexibleTime(endStr); err == nil {
			c.EndAt = &parsed
		}
	}

	// Auto-fallback caption jika kosong
	if strings.TrimSpace(c.Caption) == "" {
		pkgName := strings.TrimSpace(c.PackageName)
		if pkgName == "" {
			pkgName = "Paket Hemat"
		}
		c.Caption = fmt.Sprintf("Promo Paket: %s", pkgName)
	}

	return nil
}

type PromotionBundleItemDetail struct {
	ProductID           string  `json:"product_id"`
	ProductName         string  `json:"product_name"`
	ImageURL            string  `json:"image_url"`
	Quantity            int     `json:"quantity"`
	NormalPrice         float64 `json:"normal_price"`
	SubtotalNormalPrice float64 `json:"subtotal_normal_price"`
}

type ExtendPromotionInput struct {
	AddDays *int       `json:"add_days" validate:"omitempty,min=1,max=90"`
	EndAt   *time.Time `json:"end_at"`
}

type PromotionSummary struct {
	ActiveCount      int64 `json:"active_count"`
	TotalImpressions int64 `json:"total_impressions"`
	TotalClicks      int64 `json:"total_clicks"`
}

type PromotionMainProductCard struct {
	ProductID      string  `json:"product_id"`
	ProductName    string  `json:"product_name"`
	ImageURL       string  `json:"image_url"`
	OriginalPrice  float64 `json:"original_price"`
	FinalPrice     float64 `json:"final_price"`
	DiscountType   string  `json:"discount_type"`
	Discount       float64 `json:"discount"`
	DiscountAmount float64 `json:"discount_amount"`
	DiscountPct    float64 `json:"discount_pct"`
}

type PromotionPostCard struct {
	PostID       string   `json:"post_id"`
	Content      string   `json:"content"`
	Images       []string `json:"images"`
	VideoURL     *string  `json:"video_url,omitempty"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
}

type PromotionListItem struct {
	PromotionID        string                    `json:"promotion_id"`
	PromotionType      string                    `json:"promotion_type"` // "catalog" | "post" | "bundling" | "per_product"
	PostID             *string                   `json:"post_id"`
	Caption            string                    `json:"caption"`
	MainProduct        *PromotionMainProductCard `json:"main_product,omitempty"`
	Post               *PromotionPostCard        `json:"post,omitempty"`
	PackageName        *string                   `json:"package_name,omitempty"`
	PackagePrice       *float64                  `json:"package_price,omitempty"`
	PackageImage       *string                   `json:"package_image,omitempty"`
	OriginalTotalPrice *float64                  `json:"original_total_price,omitempty"`
	SavingsAmount      *float64                  `json:"savings_amount,omitempty"`
	SavingsPercentage  *float64                  `json:"savings_percentage,omitempty"`
	BundleItemCount    *int                      `json:"bundle_item_count,omitempty"`
	ProductCount       int                       `json:"product_count"`
	RemainingDays      int                       `json:"remaining_days"` // Dibulatkan ke atas
	StartAt            time.Time                 `json:"start_at"`
	EndAt              time.Time                 `json:"end_at"`
	Status             string                    `json:"status"`
	ImpressionCount    int                       `json:"impression_count"`
	ClickCount         int                       `json:"click_count"`
	CreatedAt          time.Time                 `json:"created_at"`
}

type PromotionProductDetailItem struct {
	ProductID      string   `json:"product_id"`
	ProductName    string   `json:"product_name"`
	ImageURL       string   `json:"image_url"`
	OriginalPrice  float64  `json:"original_price"`
	FinalPrice     float64  `json:"final_price"`
	DiscountSource string   `json:"discount_source"`
	DiscountType   string   `json:"discount_type"`
	Discount       float64  `json:"discount"`
	DiscountAmount float64  `json:"discount_amount"`
	DiscountPct    float64  `json:"discount_pct"`
	MaxDiscount    *float64 `json:"max_discount,omitempty"`
	IsMainProduct  bool     `json:"is_main_product"`
}

type PromotionDetailResponse struct {
	PromotionID        string                       `json:"promotion_id"`
	PromotionType      string                       `json:"promotion_type"` // "catalog" | "post" | "bundling" | "per_product"
	BusinessID         string                       `json:"business_id"`
	Caption            string                       `json:"caption"`
	MainProductID      *string                      `json:"main_product_id,omitempty"`
	PostID             *string                      `json:"post_id,omitempty"`
	Post               *PromotionPostCard           `json:"post,omitempty"`
	PackageName        *string                      `json:"package_name,omitempty"`
	PackagePrice       *float64                     `json:"package_price,omitempty"`
	PackageImage       *string                      `json:"package_image,omitempty"`
	OriginalTotalPrice *float64                     `json:"original_total_price,omitempty"`
	SavingsAmount      *float64                     `json:"savings_amount,omitempty"`
	SavingsPercentage  *float64                     `json:"savings_percentage,omitempty"`
	StartAt            time.Time                    `json:"start_at"`
	EndAt              time.Time                    `json:"end_at"`
	Status             string                       `json:"status"`
	RemainingDays      int                          `json:"remaining_days"`
	ImpressionCount    int                          `json:"impression_count"`
	ClickCount         int                          `json:"click_count"`
	CreatedAt          time.Time                    `json:"created_at"`
	Products           []PromotionProductDetailItem `json:"products,omitempty"`
	BundleItems        []PromotionBundleItemDetail  `json:"bundle_items,omitempty"`
}

type GetPromotionsResponse struct {
	Summary    PromotionSummary    `json:"summary"`
	Promotions []PromotionListItem `json:"promotions"`
	Pagination PaginationMeta      `json:"pagination"`
}

// ==========================================
// 3. DTO UNTUK CUSTOMER EVENTS & TRACKING
// ==========================================

type RecordPromotionEventInput struct {
	PromotionID string  `json:"promotion_id" validate:"required"`
	ProductID   *string `json:"product_id"`
	EventType   string  `json:"event_type" validate:"required,oneof=impression click"`
	Source      string  `json:"source" validate:"required,oneof=post business_profile notification product_group"`
}

type RecordPromotionEventsBatchInput struct {
	Events []RecordPromotionEventInput `json:"events" validate:"required,min=1,max=100"`
}
