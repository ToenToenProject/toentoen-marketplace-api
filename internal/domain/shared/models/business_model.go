package models

import (
	"time"
)

type BusinessCertificate struct {
	Type     string `json:"type"`
	Number   string `json:"number"`
	ImageURL string `json:"image_url"`
}

type EventSchedule struct {
	Date      string `json:"date"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

type Business struct {
	BusinessID string `gorm:"primaryKey;column:business_id" json:"business_id"`
	AccountID  string `gorm:"column:account_id" json:"account_id"`

	LogoURL           *string  `gorm:"column:logo_url" json:"logo_url"`
	BusinessName      *string  `gorm:"not null;column:business_name" json:"name"`
	BusinessNameAlias *string  `gorm:"-" json:"business_name,omitempty"`
	Handle            *string  `gorm:"unique;column:handle" json:"handle"`
	Email             *string  `gorm:"not null;column:email" json:"email"`

	Status           string `gorm:"default:'draft';column:status" json:"status"`
	SubscriptionTier string `gorm:"default:'free';column:subscription_tier" json:"subscription_tier"`

	BusinessImages          []string              `gorm:"type:jsonb;serializer:json;column:business_images" json:"business_images"`
	Certificates            []BusinessCertificate `gorm:"type:jsonb;serializer:json;column:certificates;default:'[]'" json:"certificates"`
	BusinessDetails         *string               `gorm:"column:business_details" json:"business_details"`

	NIB           *string  `gorm:"column:nib" json:"nib"`
	Country       *string  `gorm:"column:country;default:'Indonesia'" json:"country"`
	Province      *string  `gorm:"column:province" json:"province"`
	City          *string  `gorm:"column:city" json:"city"`
	AddressDetail  *string  `gorm:"column:address_detail" json:"address_detail"`
	GoogleMapsLink *string  `gorm:"column:google_maps_link" json:"google_maps_link"`
	Latitude       *float64 `gorm:"column:latitude" json:"latitude"`
	Longitude      *float64 `gorm:"column:longitude" json:"longitude"`

	IsLocationShared bool    `gorm:"column:is_location_shared;default:false" json:"is_location_shared"`
	VenueName        *string `gorm:"column:venue_name" json:"venue_name"`
	EventStartDate   *time.Time `gorm:"column:event_start_date" json:"event_start_date"`
	EventEndDate     *time.Time `gorm:"column:event_end_date" json:"event_end_date"`
	EventSchedules   []EventSchedule `gorm:"type:jsonb;serializer:json;column:event_schedules;default:'[]'" json:"event_schedules"`

	StoreType *string `gorm:"default:'regular';column:store_type" json:"store_type"`

	IsManuallyClosed bool     `gorm:"default:false;column:is_manually_closed" json:"is_manually_closed"`
	IsOpenNow        bool `gorm:"-" json:"is_open_now"`
	IsLive           bool     `gorm:"default:false;column:is_live" json:"is_live"`
	LivePlatform     *string  `gorm:"column:live_platform" json:"live_platform,omitempty"`
	LiveStartedAt    *time.Time `gorm:"column:live_started_at" json:"live_started_at,omitempty"`
	LastLiveNotifiedAt *time.Time `gorm:"column:last_live_notified_at" json:"last_live_notified_at,omitempty"`

	PhoneContact        *string  `gorm:"column:phone_contact" json:"phone_contact"`
	WhatsappContact     *string  `gorm:"column:whatsapp_contact" json:"whatsapp_contact"`
	SocialInstagram     *string  `gorm:"column:social_instagram" json:"social_instagram"`
	SocialTiktok        *string  `gorm:"column:social_tiktok" json:"social_tiktok"`
	SocialWebsites      []string `gorm:"type:jsonb;serializer:json;column:social_website" json:"websites"`
	SocialWebsitesAlias []string `gorm:"-" json:"social_websites,omitempty"`

	AverageRating float64   `gorm:"default:0;column:average_rating" json:"average_rating"`
	ReviewCount   int64     `gorm:"default:0;column:review_count" json:"review_count"`
	IsActive      bool      `gorm:"default:true;column:is_active" json:"is_active"`	
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`

	Hours      []BusinessHour `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"hours"`
	Categories []Category     `gorm:"many2many:business_has_categories;foreignKey:BusinessID;joinForeignKey:BusinessID;References:CategoryID;joinReferences:CategoryID" json:"categories"`
	Tags           []Tag          `gorm:"many2many:business_has_tags;foreignKey:BusinessID;joinForeignKey:BusinessID;References:TagID;joinReferences:TagID" json:"tags"` // mapped to old tags table
	CustomTags     []string       `gorm:"type:jsonb;serializer:json;column:custom_tags;default:'[]'" json:"custom_tags"`
	Hashtags       []string       `gorm:"type:jsonb;serializer:json;column:hashtags;default:'[]'" json:"hashtags"`
	CustomCategory *string        `gorm:"column:custom_category" json:"custom_category,omitempty"`
}

type BusinessHour struct {
	BusinessID string  `gorm:"primaryKey;column:business_id" json:"business_id"`
	DayOfWeek  int     `gorm:"primaryKey;column:day_of_week" json:"day_of_week"`
	OpenTime   *string `gorm:"column:open_time" json:"open_time"`
	CloseTime  *string `gorm:"column:close_time" json:"close_time"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
}

func (BusinessHour) TableName() string { return "business_hours" }

type CreateBusinessInput struct {
	LogoURL      string `json:"logo_url"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`

	BusinessImages []string `json:"business_images" validate:"max=5"`
	Certificates   []BusinessCertificate `json:"certificates" validate:"max=5"`

	CategoryIDs     []int64  `json:"category_ids"`
	CustomCategory  *string  `json:"custom_category" validate:"omitempty,max=100"`
	Tags            []int64  `json:"tags"`
	CustomTags      []string `json:"custom_tags"`
	Hashtags        []string `json:"hashtags"`
	BusinessDetails string   `json:"business_details" validate:"max=2000"`

	NIB             string  `json:"nib"`
	Country         string  `json:"country"`
	Province        string  `json:"province"`
	City            string  `json:"city"`
	AddressDetail   string  `json:"address_detail"`
	GoogleMapsLink  string  `json:"google_maps_link"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	
	IsLocationShared bool    `json:"is_location_shared"`
	VenueName        string  `json:"venue_name"`
	EventStartDate   string  `json:"event_start_date"`
	EventEndDate     string  `json:"event_end_date"`
	EventSchedules   []EventSchedule `json:"event_schedules"`

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
}

type UpdateBusinessInput struct {
	LogoURL      *string `json:"logo_url"`
	BusinessName *string `json:"business_name"`
	Name         *string `json:"name"` // Alias untuk business_name
	Handle       *string `json:"handle"`
	Email        *string `json:"email"`

	BusinessImages *[]string              `json:"business_images" validate:"omitempty,max=5"`
	Certificates   *[]BusinessCertificate `json:"certificates" validate:"omitempty,max=5"`

	CategoryIDs     *[]int64  `json:"category_ids"`
	CustomCategory  *string   `json:"custom_category" validate:"omitempty,max=100"`
	Tags            *[]int64  `json:"tags"`
	CustomTags      *[]string `json:"custom_tags"`
	Hashtags        *[]string `json:"hashtags"`
	BusinessDetails *string  `json:"business_details" validate:"omitempty,max=2000"`

	NIB            *string  `json:"nib"`
	Country        *string  `json:"country"`
	Province       *string  `json:"province"`
	City           *string  `json:"city"`
	AddressDetail  *string  `json:"address_detail"`
	Address        *string  `json:"address"` // Alias untuk address_detail
	GoogleMapsLink *string  `json:"google_maps_link"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`

	IsLocationShared *bool            `json:"is_location_shared"`
	VenueName        *string          `json:"venue_name"`
	EventStartDate   *string          `json:"event_start_date"`
	EventEndDate     *string          `json:"event_end_date"`
	EventSchedules   *[]EventSchedule `json:"event_schedules"`

	StoreType *string              `json:"store_type"`
	Hours     *[]BusinessHourInput `json:"hours"`

	PhoneContact    *string   `json:"phone_contact"`
	WhatsappContact *string   `json:"whatsapp_contact"`
	SocialInstagram *string   `json:"social_instagram"`
	SocialTiktok    *string   `json:"social_tiktok"`
	SocialWebsites  *[]string `json:"social_websites"`
	Websites        *[]string `json:"websites"` // Alias untuk social_websites
}

type BusinessListItem struct {
	BusinessID          string  `json:"business_id"`
	BusinessName        string  `json:"business_name"`
	OwnerName           string  `json:"owner_name"` // Membutuhkan join/preload ke tabel Account
	LogoURL             string  `json:"logo_url"`
	Address             string  `json:"address"` // Gabungan AddressDetail, City, Province
	AverageRating       float64 `json:"average_rating"`
	TotalProductReviews int64   `json:"total_product_reviews"`
}

// GetBusinessListResponse adalah bungkus utama paginasi
type GetBusinessListResponse struct {
	Businesses []BusinessListItem `json:"businesses"`
	Pagination PaginationMeta     `json:"pagination"`
}

type BusinessWithPreview struct {
	Business
	OwnerName           string
	AverageRating       float64
	TotalProductReviews int64
}

// ToggleLiveInput adalah payload untuk mengaktifkan / menonaktifkan live streaming
type ToggleLiveInput struct {
	IsLive   bool    `json:"is_live"`
	Platform *string `json:"platform"` // "tiktok" atau "instagram"
}

// LiveStatusResponse adalah response setelah mengubah status live streaming
type LiveStatusResponse struct {
	BusinessID    string     `json:"business_id"`
	IsLive        bool       `json:"is_live"`
	LivePlatform  *string    `json:"live_platform,omitempty"`
	LiveStartedAt *time.Time `json:"live_started_at,omitempty"`
	SocialURL     *string    `json:"social_url,omitempty"`
}