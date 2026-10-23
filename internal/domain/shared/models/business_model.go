package models

import (
	"encoding/json"
	"strings"
	"time"
)

type BusinessCertificate struct {
	Type                string `json:"type"`
	Name                string `json:"name,omitempty"`
	CertificateName     string `json:"certificate_name,omitempty"`
	CertificateType     string `json:"certificate_type,omitempty"`
	Number              string `json:"number"`
	CertificateNumber   string `json:"certificate_number,omitempty"`
	ImageURL            string `json:"image_url"`
	URL                 string `json:"url,omitempty"`
	CertificateImageURL string `json:"certificate_image_url,omitempty"`
}

func (b *BusinessCertificate) UnmarshalJSON(data []byte) error {
	// 1. Tangani jika payload berupa plain string URL (misal: "https://storage...")
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		b.ImageURL = strVal
		b.URL = strVal
		return nil
	}

	// 2. Tangani jika payload berupa object dengan berbagai kemungkinan alias key dari FE
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	getString := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok && v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		return ""
	}

	b.Type = getString("type", "name", "certificate_name", "certificate_type", "title", "jenis_sertifikat", "nama_sertifikat", "nama")
	b.Number = getString("number", "certificate_number", "cert_number", "nomor", "nomor_sertifikat", "no")
	b.ImageURL = getString("image_url", "imageUrl", "url", "image", "file_url", "certificate_image_url", "photo_url", "foto")

	// Set aliases agar konsisten
	b.Name = b.Type
	b.CertificateName = b.Type
	b.CertificateType = b.Type
	b.CertificateNumber = b.Number
	b.URL = b.ImageURL
	b.CertificateImageURL = b.ImageURL

	return nil
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

	BusinessImages            []string              `gorm:"type:jsonb;serializer:json;column:business_images" json:"business_images"`
	Certificates              []BusinessCertificate `gorm:"type:jsonb;serializer:json;column:certificates;default:'[]'" json:"certificates"`
	CertificatesAlias         []BusinessCertificate `gorm:"-" json:"certificate,omitempty"`
	BusinessCertificatesAlias []BusinessCertificate `gorm:"-" json:"business_certificates,omitempty"`
	BusinessDetails           *string               `gorm:"column:business_details" json:"business_details"`

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

	IsManuallyClosed bool     `gorm:"default:true;column:is_manually_closed" json:"is_manually_closed"`
	ConfirmedOpenDate     *string    `gorm:"type:date;column:confirmed_open_date" json:"confirmed_open_date,omitempty"`
	ConfirmedOvertimeDate *string    `gorm:"type:date;column:confirmed_overtime_date" json:"confirmed_overtime_date,omitempty"`
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
	IsClosed   bool    `gorm:"column:is_closed;default:false" json:"is_closed"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
}

func (BusinessHour) TableName() string { return "business_hours" }

type CreateBusinessInput struct {
	LogoURL      string `json:"logo_url"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`

	BusinessImages          []string              `json:"business_images" validate:"max=5"`
	Certificates            []BusinessCertificate `json:"certificates" validate:"max=5"`
	CertificatesAlias       []BusinessCertificate `json:"certificate" validate:"max=5"`
	BusinessCertificates    []BusinessCertificate `json:"business_certificates" validate:"max=5"`
	BusinessCertificateSing []BusinessCertificate `json:"business_certificate" validate:"max=5"`

	CategoryIDs     []int64  `json:"category_ids"`
	CustomCategory  *string  `json:"custom_category" validate:"omitempty,max=100"`
	Tags            []int64  `json:"tags"`
	CustomTags      []string `json:"custom_tags"`
	Hashtags        []string `json:"hashtags"`
	BusinessDetails string   `json:"business_details" validate:"max=300"`
	Description     string   `json:"description" validate:"max=300"` // Alias untuk business_details

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
	IsClosed  bool   `json:"is_closed"`
}

type UpdateBusinessInput struct {
	LogoURL      *string `json:"logo_url"`
	BusinessName *string `json:"business_name"`
	Name         *string `json:"name"` // Alias untuk business_name
	Handle       *string `json:"handle"`
	Email        *string `json:"email"`

	BusinessImages          *[]string              `json:"business_images" validate:"omitempty,max=5"`
	Certificates            *[]BusinessCertificate `json:"certificates" validate:"omitempty,max=5"`
	CertificatesAlias       *[]BusinessCertificate `json:"certificate" validate:"omitempty,max=5"`
	BusinessCertificates    *[]BusinessCertificate `json:"business_certificates" validate:"omitempty,max=5"`
	BusinessCertificateSing *[]BusinessCertificate `json:"business_certificate" validate:"omitempty,max=5"`

	CategoryIDs     *[]int64  `json:"category_ids"`
	CustomCategory  *string   `json:"custom_category" validate:"omitempty,max=100"`
	Tags            *[]int64  `json:"tags"`
	CustomTags      *[]string `json:"custom_tags"`
	Hashtags        *[]string `json:"hashtags"`
	BusinessDetails *string  `json:"business_details" validate:"omitempty,max=300"`
	Description     *string  `json:"description" validate:"omitempty,max=300"` // Alias untuk business_details

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
	IsOpenNow           bool    `json:"is_open_now"`
	IsManuallyClosed    bool    `json:"is_manually_closed"`
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

// ConfirmScheduleInput adalah payload dari notifikasi/popup konfirmasi jadwal buka/tutup
type ConfirmScheduleInput struct {
	Action string `json:"action" validate:"required,oneof=confirm_open confirm_overtime decline"`
}

// ==========================================
// STORE TYPE STANDARDIZATION & AUDIT HISTORY
// ==========================================

const (
	StoreTypeRumah        = "rumah"
	StoreTypeKeliling     = "keliling"
	StoreTypeGerobak      = "gerobak"
	StoreTypeToko         = "toko"
	StoreTypeRuko         = "ruko"
	StoreTypePasar        = "pasar"
	StoreTypeAlunAlun     = "alun_alun"
	StoreTypePusatBelanja = "pusat_belanja"
	StoreTypeAcaraMusiman = "acara_musiman"
)

var StoreTypeLabels = map[string]string{
	StoreTypeRumah:        "Rumah / Dapur Rumahan",
	StoreTypeKeliling:     "Pedagang Keliling",
	StoreTypeGerobak:      "Gerobak / Kaki Lima",
	StoreTypeToko:         "Toko / Warung Kelontong",
	StoreTypeRuko:         "Ruko / Kios Komersial",
	StoreTypePasar:        "Pasar Tradisional",
	StoreTypeAlunAlun:     "Alun-Alun / Lapangan Terbuka",
	StoreTypePusatBelanja: "Pusat Perbelanjaan / Mall",
	StoreTypeAcaraMusiman: "Acara / Musiman (Pop-up)",
	"regular":             "Regular",
}

func NormalizeStoreType(raw string) string {
	cleaned := strings.ToLower(strings.TrimSpace(raw))
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	for k := range StoreTypeLabels {
		if cleaned == k {
			return k
		}
	}
	return "regular"
}

func GetStoreTypeLabel(storeType string) string {
	norm := NormalizeStoreType(storeType)
	if label, ok := StoreTypeLabels[norm]; ok {
		return label
	}
	return "Regular"
}

type BusinessNameHistory struct {
	HistoryID  string    `gorm:"primaryKey;column:history_id" json:"history_id"`
	BusinessID string    `gorm:"column:business_id;index;not null" json:"business_id"`
	OldName    string    `gorm:"column:old_name;not null" json:"old_name"`
	NewName    string    `gorm:"column:new_name;not null" json:"new_name"`
	ChangedBy  string    `gorm:"column:changed_by;not null" json:"changed_by"`
	CreatedAt  time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
}

func (BusinessNameHistory) TableName() string {
	return "business_name_histories"
}