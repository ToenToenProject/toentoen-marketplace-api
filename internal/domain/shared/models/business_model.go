package models

import (
	"time"
)

type Business struct {
	BusinessID string `gorm:"primaryKey;column:business_id" json:"business_id"`
	AccountID  string `gorm:"column:account_id" json:"account_id"`

	LogoURL      *string `gorm:"column:logo_url" json:"logo_url"`
	BusinessName *string `gorm:"not null;column:business_name" json:"name"`
	Handle       *string `gorm:"unique;column:handle" json:"handle"`
	Email        *string `gorm:"not null;column:email" json:"email"`

	Status           string `gorm:"default:'draft';column:status" json:"status"`
	SubscriptionTier string `gorm:"default:'free';column:subscription_tier" json:"subscription_tier"`

	BusinessImages          []string `gorm:"type:jsonb;serializer:json;column:business_images" json:"business_images"`
	BusinessDetails *string  `gorm:"column:business_details" json:"business_details"`

	Province      *string  `gorm:"column:province" json:"province"`
	City          *string  `gorm:"column:city" json:"city"`
	AddressDetail *string  `gorm:"column:address_detail" json:"address_detail"`
	Latitude      *float64 `gorm:"column:latitude" json:"latitude"`
	Longitude     *float64 `gorm:"column:longitude" json:"longitude"`

	StoreType *string `gorm:"default:'regular';column:store_type" json:"store_type"`

	IsManuallyClosed bool     `gorm:"default:false;column:is_manually_closed" json:"is_manually_closed"`
	IsOpenNow        bool `gorm:"-" json:"is_open_now"`
	PhoneContact     *string  `gorm:"column:phone_contact" json:"phone_contact"`
	WhatsappContact  *string  `gorm:"column:whatsapp_contact" json:"whatsapp_contact"`
	SocialInstagram  *string  `gorm:"column:social_instagram" json:"social_instagram"`
	SocialTiktok     *string  `gorm:"column:social_tiktok" json:"social_tiktok"`
	SocialWebsites   []string `gorm:"type:jsonb;serializer:json;column:social_website" json:"websites"`

	AverageRating float64   `gorm:"default:0;column:average_rating" json:"average_rating"`
	ReviewCount   int64     `gorm:"default:0;column:review_count" json:"review_count"`
	IsActive      bool      `gorm:"default:true;column:is_active" json:"is_active"`	
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`

	Hours      []BusinessHour `gorm:"foreignKey:BusinessID" json:"hours"`
	Categories []Category     `gorm:"many2many:business_categories;foreignKey:BusinessID;joinForeignKey:BusinessID;References:CategoryID;joinReferences:CategoryID" json:"categories"`
	Tags       []Tag          `gorm:"many2many:business_tags;foreignKey:BusinessID;joinForeignKey:BusinessID;References:TagID;joinReferences:TagID" json:"tags"`
}

type BusinessHour struct {
	BusinessID string  `gorm:"primaryKey;column:business_id" json:"business_id"`
	DayOfWeek  int     `gorm:"primaryKey;column:day_of_week" json:"day_of_week"`
	OpenTime   *string `gorm:"column:open_time" json:"open_time"`
	CloseTime  *string `gorm:"column:close_time" json:"close_time"`
	IsClosed   bool    `gorm:"default:false;column:is_closed" json:"is_closed"`
}

func (BusinessHour) TableName() string { return "business_hours" }

type CreateBusinessInput struct {
	LogoURL      string `json:"logo_url"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`

	BusinessImages []string `json:"business_images"`

	CategoryIDs     []int64 `json:"category_ids"`
	TagIDs          []int64 `json:"tag_ids"`
	BusinessDetails string  `json:"business_details"`

	Province      string  `json:"province"`
	City          string  `json:"city"`
	AddressDetail string  `json:"address_detail"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`

	StoreType string              `json:"store_type"`
	Hours     []BusinessHourInput `json:"hours"`

	PhoneContact    string   `json:"phone_contact"`
	WhatsappContact string   `json:"whatsapp_contact"`
	SocialInstagram string   `json:"social_instagram"`
	SocialTiktok    string   `json:"social_tiktok"`
	SocialWebsites  []string `json:"social_websites"`
}

type BusinessHourInput struct {
	DayOfWeek int    `json:"day_of_week"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed  bool   `json:"is_closed"`
}

type UpdateBusinessInput struct {
	LogoURL      string `json:"logo_url"`
	BusinessName string `json:"business_name"`
	Handle       string `json:"handle"`
	Email        string `json:"email"`

	BusinessImages []string `json:"business_images"`

	CategoryIDs     []int64 `json:"category_ids"`
	TagIDs          []int64 `json:"tag_ids"`
	BusinessDetails string  `json:"business_details"`

	Province      string  `json:"province"`
	City          string  `json:"city"`
	AddressDetail string  `json:"address_detail"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`

	StoreType string              `json:"store_type"`
	Hours     []BusinessHourInput `json:"hours"`

	PhoneContact    string   `json:"phone_contact"`
	WhatsappContact string   `json:"whatsapp_contact"`
	SocialInstagram string   `json:"social_instagram"`
	SocialTiktok    string   `json:"social_tiktok"`
	SocialWebsites  []string `json:"social_websites"`
}

type BusinessListItem struct {
	BusinessID   string `json:"business_id"`
	BusinessName string `json:"business_name"`
	OwnerName    string `json:"owner_name"` // Membutuhkan join/preload ke tabel Account
	LogoURL      string `json:"logo_url"`
	Address      string `json:"address"`    // Gabungan AddressDetail, City, Province

	// 🪄 4 Gambar preview produk yang diambil dari tabel products
	PreviewImages []string `json:"preview_images"` 
}

// GetBusinessListResponse adalah bungkus utama paginasi
type GetBusinessListResponse struct {
	Businesses []BusinessListItem `json:"businesses"`
	Pagination PaginationMeta     `json:"pagination"`
}

type BusinessWithPreview struct {
	Business
	OwnerName     string
	PreviewImages []string
}