package models

import "time"

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

	PromotionTypeCatalog = "catalog"
	PromotionTypePost    = "post"
)

type Promotion struct {
	PromotionID     string     `gorm:"primaryKey;column:promotion_id" json:"promotion_id"`
	BusinessID      string     `gorm:"column:business_id;index" json:"business_id"`
	Caption         string     `gorm:"column:caption;size:150" json:"caption"`
	MainProductID   string     `gorm:"column:main_product_id" json:"main_product_id"`
	PostID          *string    `gorm:"column:post_id" json:"post_id,omitempty"`
	StartAt         time.Time  `gorm:"column:start_at" json:"start_at"`
	EndAt           time.Time  `gorm:"column:end_at" json:"end_at"`
	Status          string     `gorm:"column:status;default:'scheduled'" json:"status"`
	NotifiedAt      *time.Time `gorm:"column:notified_at" json:"notified_at,omitempty"`
	StoppedAt       *time.Time `gorm:"column:stopped_at" json:"stopped_at,omitempty"`
	ImpressionCount int        `gorm:"column:impression_count;default:0" json:"impression_count"`
	ClickCount      int        `gorm:"column:click_count;default:0" json:"click_count"`
	CreatedBy       string     `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`

	// Relasi
	Business    *Business          `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	MainProduct *Product           `gorm:"foreignKey:MainProductID;references:ProductID" json:"main_product,omitempty"`
	Post        *BusinessPost      `gorm:"foreignKey:PostID;references:PostID" json:"post,omitempty"`
	Products    []PromotionProduct `gorm:"foreignKey:PromotionID;references:PromotionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"products,omitempty"`
}

func (Promotion) TableName() string {
	return "promotions"
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
// 2. DTO UNTUK REQUEST & RESPONSE (MERCHANT)
// ==========================================

type EligibleProductItem struct {
	ProductID              string     `json:"product_id"`
	ProductName            string     `json:"product_name"`
	ImageURL               string     `json:"image_url"`
	Price                  float64    `json:"price"`
	IsEligible             bool       `json:"is_eligible"`
	IneligibleReason       string     `json:"ineligible_reason,omitempty"`
	HasExistingDiscount    bool       `json:"has_existing_discount"`
	CurrentDiscountType    string     `json:"current_discount_type,omitempty"`
	CurrentDiscount        float64    `json:"current_discount,omitempty"`
	CurrentMaxDiscount     *float64   `json:"current_max_discount,omitempty"`
	CurrentDiscountStartAt *time.Time `json:"current_discount_start_at,omitempty"`
	CurrentDiscountEndAt   *time.Time `json:"current_discount_end_at,omitempty"`
}

type CreatePromotionItemInput struct {
	ProductID            string   `json:"product_id" validate:"required"`
	DiscountType         *string  `json:"discount_type"` // "percentage" atau "nominal"
	Discount             *float64 `json:"discount"`
	MaxDiscount          *float64 `json:"max_discount"`
	KeepExistingDiscount *bool    `json:"keep_existing_discount"`
}

type CreatePromotionInput struct {
	Caption       string                     `json:"caption" validate:"required,min=3,max=150"`
	MainProductID string                     `json:"main_product_id" validate:"required"`
	StartAt       time.Time                  `json:"start_at" validate:"required"`
	EndAt         time.Time                  `json:"end_at" validate:"required"`
	Products      []CreatePromotionItemInput `json:"products" validate:"required,min=1,max=20"`
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
	PromotionID     string                    `json:"promotion_id"`
	PromotionType   string                    `json:"promotion_type"` // "catalog" | "post"
	PostID          *string                   `json:"post_id"`
	Caption         string                    `json:"caption"`
	MainProduct     *PromotionMainProductCard `json:"main_product"`
	Post            *PromotionPostCard        `json:"post"`
	ProductCount    int                       `json:"product_count"`
	RemainingDays   int                       `json:"remaining_days"` // Dibulatkan ke atas
	StartAt         time.Time                 `json:"start_at"`
	EndAt           time.Time                 `json:"end_at"`
	Status          string                    `json:"status"`
	ImpressionCount int                       `json:"impression_count"`
	ClickCount      int                       `json:"click_count"`
	CreatedAt       time.Time                 `json:"created_at"`
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
	PromotionID     string                       `json:"promotion_id"`
	PromotionType   string                       `json:"promotion_type"` // "catalog" | "post"
	BusinessID      string                       `json:"business_id"`
	Caption         string                       `json:"caption"`
	MainProductID   *string                      `json:"main_product_id,omitempty"`
	PostID          *string                      `json:"post_id,omitempty"`
	Post            *PromotionPostCard           `json:"post,omitempty"`
	StartAt         time.Time                    `json:"start_at"`
	EndAt           time.Time                    `json:"end_at"`
	Status          string                       `json:"status"`
	RemainingDays   int                          `json:"remaining_days"`
	ImpressionCount int                          `json:"impression_count"`
	ClickCount      int                          `json:"click_count"`
	CreatedAt       time.Time                    `json:"created_at"`
	Products        []PromotionProductDetailItem `json:"products"`
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
