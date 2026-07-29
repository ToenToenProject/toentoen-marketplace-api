package models

import "time"

// ================================================================
// DEMAND SIGNAL V1 (LEGACY — Tidak digunakan lagi, tabel tetap ada)
// ================================================================

type DemandSignal struct {
	SignalID    string    `gorm:"primaryKey;column:signal_id" json:"signal_id"`
	AccountID   string    `gorm:"not null" json:"account_id"`
	Category    string    `gorm:"not null" json:"category"`
	Description string    `json:"description"`
	MediaUrls   []string  `gorm:"type:json;serializer:json" json:"media_urls"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RadiusKm    int       `gorm:"default:5" json:"radius_km"`
	Status      string    `gorm:"default:'open'" json:"status"`
	IsFulfilled bool      `gorm:"default:false" json:"is_fulfilled"`
	FulfilledAt *time.Time `json:"fulfilled_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`

	Account   *Account         `json:"account,omitempty"`
	Responses []DemandResponse `gorm:"foreignKey:SignalID;references:SignalID" json:"responses,omitempty"`
}

type DemandResponse struct {
	ResponseID string    `gorm:"primaryKey;column:response_id" json:"response_id"`
	SignalID   string    `gorm:"not null" json:"signal_id"`
	BusinessID string    `gorm:"not null" json:"business_id"`
	Message    string    `json:"message"`
	MediaUrls  []string  `gorm:"type:json;serializer:json" json:"media_urls"`
	IsAccepted bool      `gorm:"default:false" json:"is_accepted"`
	CreatedAt  time.Time `json:"created_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
}

// --- LEGACY V1 INPUT STRUCTS (masih dipakai oleh package demand lama) ---

type CreateSignalInput struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	MediaUrls   []string `json:"media_urls"`
	ImageURLs   []string `json:"image_urls"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	RadiusKm    int      `json:"radius_km"`
	IsUrgent    bool     `json:"is_urgent"`
}

type CreateResponseInput struct {
	BusinessID string   `json:"business_id"`
	Message    string   `json:"message"`
	MediaUrls  []string `json:"media_urls"`
}

// ================================================================
// DEMAND SIGNAL V2 (BARU — Conversation-Based)
// ================================================================

// Signal adalah permintaan kebutuhan dari customer
type Signal struct {
	SignalID    string    `gorm:"primaryKey;column:signal_id" json:"signal_id"`
	CustomerID  string    `gorm:"not null;column:customer_id" json:"customer_id"`
	Description string    `gorm:"not null" json:"description"`
	Category    string    `gorm:"not null" json:"category"`
	Latitude    float64   `gorm:"not null" json:"latitude"`
	Longitude   float64   `gorm:"not null" json:"longitude"`
	RadiusKm    int       `gorm:"default:5" json:"radius_km"`
	IsUrgent    bool      `gorm:"default:false" json:"is_urgent"`
	Status      string    `gorm:"default:'active'" json:"status"` // active, completed, expired
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Kolom location (PostGIS geometry) dikelola via raw SQL, tidak oleh GORM
	// location geometry(Point, 4326)

	// Relations
	Account       *Account             `gorm:"foreignKey:CustomerID" json:"account,omitempty"`
	Images        []SignalImage        `gorm:"foreignKey:SignalID;references:SignalID" json:"images,omitempty"`
	Conversations []SignalConversation `gorm:"foreignKey:SignalID;references:SignalID" json:"conversations,omitempty"`
}

func (Signal) TableName() string { return "signals" }

// SignalImage adalah lampiran gambar pada signal
type SignalImage struct {
	ImageID  string `gorm:"primaryKey;column:image_id" json:"image_id"`
	SignalID string `gorm:"not null;column:signal_id" json:"signal_id"`
	ImageURL string `gorm:"not null;column:image_url" json:"image_url"`
}

func (SignalImage) TableName() string { return "signal_images" }

// SignalConversation adalah ruang negosiasi antara customer dan merchant
type SignalConversation struct {
	ConversationID string    `gorm:"primaryKey;column:conversation_id" json:"conversation_id"`
	SignalID       string    `gorm:"not null;column:signal_id" json:"signal_id"`
	MerchantID     string    `gorm:"not null;column:merchant_id" json:"merchant_id"`
	BusinessID     string    `gorm:"not null;column:business_id" json:"business_id"`
	Status         string    `gorm:"default:'open'" json:"status"` // open, locked
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Business *Business       `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Merchant *Account        `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
	Messages []SignalMessage `gorm:"foreignKey:ConversationID;references:ConversationID" json:"messages,omitempty"`
}

func (SignalConversation) TableName() string { return "signal_conversations" }

// SignalMessage adalah isi chat dalam conversation
type SignalMessage struct {
	MessageID      string    `gorm:"primaryKey;column:message_id" json:"message_id"`
	ConversationID string    `gorm:"not null;column:conversation_id" json:"conversation_id"`
	SenderType     string    `gorm:"not null;column:sender_type" json:"sender_type"` // "customer" atau "merchant"
	SenderID       string    `gorm:"not null;column:sender_id" json:"sender_id"`
	MessageText    string    `gorm:"column:message_text" json:"message_text"`
	AttachmentURL  *string   `gorm:"column:attachment_url" json:"attachment_url"`
	IsRead         bool      `gorm:"default:false;column:is_read" json:"is_read"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (SignalMessage) TableName() string { return "signal_messages" }

// ================================================================
// INPUT STRUCTS (V2) — CreateSignalInput sudah didefinisikan di atas (V1 struct)
// ================================================================

type InitiateConversationInput struct {
	BusinessID string `json:"business_id"`
	Message    string `json:"message"`
	ImageURL   string `json:"image_url"` // opsional, lampiran gambar produk
}

type SendMessageInput struct {
	MessageText   string `json:"message_text"`
	AttachmentURL string `json:"attachment_url"` // opsional
}

// ================================================================
// RESPONSE STRUCTS (V2)
// ================================================================

// SignalListItem untuk daftar signal (ringkas)
type SignalListItem struct {
	SignalID          string    `json:"signal_id"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	IsUrgent          bool      `json:"is_urgent"`
	Status            string    `json:"status"`
	ConversationCount int       `json:"conversation_count"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// SignalExploreItem untuk merchant explore (dengan jarak)
type SignalExploreItem struct {
	SignalID    string    `json:"signal_id"`
	CustomerID  string    `json:"customer_id"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	IsUrgent    bool      `json:"is_urgent"`
	RadiusKm    int       `json:"radius_km"`
	DistanceKm  float64   `json:"distance_km"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`

	CustomerName      string   `json:"customer_name"`
	CustomerAvatar    *string  `json:"customer_avatar"`
	Images            []string `json:"images"`
}