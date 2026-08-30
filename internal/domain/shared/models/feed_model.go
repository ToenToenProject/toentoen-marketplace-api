package models

import "time"

// Action types untuk interaksi feed
const (
	ActionImpression   = "impression"   // Muncul di layar
	ActionDwell        = "dwell"        // Berhenti melihat (> 2 detik)
	ActionSkip         = "skip"         // Scroll cepat tanpa melihat (< 0.5 detik)
	ActionClickPost    = "click_post"   // Buka detail postingan
	ActionClickProduct = "click_product" // Buka produk yang di-tag
	ActionClickChat    = "click_chat"   // Buka chat toko dari postingan
	ActionClickStore   = "click_store"  // Buka profil toko dari postingan
	ActionClickMap     = "click_map"    // Buka rute/peta toko
	ActionLike         = "like"         // Like postingan
	ActionBookmark     = "bookmark"     // Simpan postingan
	ActionShare        = "share"        // Bagikan postingan
)

// UserPostInteraction mencatat log interaksi pengguna terhadap postingan feed
type UserPostInteraction struct {
	InteractionID string    `gorm:"primaryKey;column:interaction_id" json:"interaction_id"`
	AccountID     string    `gorm:"column:account_id;index" json:"account_id"`
	PostID        string    `gorm:"column:post_id;index" json:"post_id"`
	BusinessID    string    `gorm:"column:business_id;index" json:"business_id"`
	CategoryID    *int64    `gorm:"column:category_id;index" json:"category_id"`
	ActionType    string    `gorm:"column:action_type;index" json:"action_type"`
	DwellTimeMs   int       `gorm:"default:0;column:dwell_time_ms" json:"dwell_time_ms"`
	CreatedAt     time.Time `gorm:"column:created_at;index" json:"created_at"`

	Account  *Account      `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Post     *BusinessPost `gorm:"foreignKey:PostID;references:PostID" json:"post,omitempty"`
	Business *Business     `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
}

func (UserPostInteraction) TableName() string {
	return "user_post_interactions"
}

// UserCategoryAffinity mencatat skor preferensi minat kategori per akun pengguna
type UserCategoryAffinity struct {
	AccountID        string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CategoryID       int64     `gorm:"primaryKey;column:category_id" json:"category_id"`
	AffinityScore    float64   `gorm:"default:1.0;column:affinity_score" json:"affinity_score"`
	InteractionCount int       `gorm:"default:1;column:interaction_count" json:"interaction_count"`
	LastInteractedAt time.Time `gorm:"column:last_interacted_at" json:"last_interacted_at"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`

	Account  *Account  `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID;references:CategoryID" json:"category,omitempty"`
}

func (UserCategoryAffinity) TableName() string {
	return "user_category_affinities"
}

// DTO untuk Batching Events dari Front End Mobile
type FeedEventInput struct {
	PostID      string `json:"post_id" validate:"required"`
	BusinessID  string `json:"business_id"`
	CategoryID  *int64 `json:"category_id"`
	ActionType  string `json:"action_type" validate:"required"` // impression, dwell, click_product, click_chat, etc.
	DwellTimeMs int    `json:"dwell_time_ms"`                   // durasi lihat dalam milidetik
}

type FeedEventBatchInput struct {
	Events []FeedEventInput `json:"events" validate:"required,min=1,max=100"`
}

// DTO untuk Query Timeline / Feed Rekomendasi
type TimelineQueryInput struct {
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	RadiusKm   float64 `json:"radius_km"`
	CategoryID int64   `json:"category_id"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
}
