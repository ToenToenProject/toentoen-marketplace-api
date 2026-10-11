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

	Account   *Account         `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Responses []DemandResponse `gorm:"foreignKey:SignalID;references:SignalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"responses,omitempty"`
}

type DemandResponse struct {
	ResponseID string    `gorm:"primaryKey;column:response_id" json:"response_id"`
	SignalID   string    `gorm:"not null" json:"signal_id"`
	BusinessID string    `gorm:"not null" json:"business_id"`
	Message    string    `json:"message"`
	MediaUrls  []string  `gorm:"type:json;serializer:json" json:"media_urls"`
	IsAccepted bool      `gorm:"default:false" json:"is_accepted"`
	CreatedAt  time.Time `json:"created_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
}

// --- LEGACY V1 INPUT STRUCTS (masih dipakai oleh package demand lama) ---

type CreateSignalInput struct {
	CategoryID     *int64   `json:"category_id"`
	Category       string   `json:"category"`
	CategoryIDs    []int64  `json:"category_ids"`
	TagIDs         []int64  `json:"tag_ids"`
	Description    string   `json:"description"`
	MediaUrls      []string `json:"media_urls"`
	ImageURLs      []string `json:"image_urls"`
	Images         []string `json:"images"`
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	RadiusKm       int      `json:"radius_km"`
	IsUrgent       bool     `json:"is_urgent"`
	ExpiresInHours int      `json:"expires_in_hours"`
	MinBudget      *float64 `json:"min_budget"`
	MaxBudget      *float64 `json:"max_budget"`
	MaxOffers      *int     `json:"max_offers"`
}

type UpdateSignalInput struct {
	CategoryID     *int64   `json:"category_id"`
	Category       string   `json:"category"`
	CategoryIDs    []int64  `json:"category_ids"`
	TagIDs         []int64  `json:"tag_ids"`
	Description    string   `json:"description"`
	MediaUrls      []string `json:"media_urls"`
	ImageURLs      []string `json:"image_urls"`
	Images         []string `json:"images"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	RadiusKm       *int     `json:"radius_km"`
	IsUrgent       *bool    `json:"is_urgent"`
	ExpiresInHours *int     `json:"expires_in_hours"`
	MinBudget      *float64 `json:"min_budget"`
	MaxBudget      *float64 `json:"max_budget"`
	MaxOffers      *int     `json:"max_offers"`
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
	CategoryID  *int64    `gorm:"column:category_id" json:"category_id,omitempty"`
	Category    string    `gorm:"not null" json:"category"`
	CategoryIDs []int64   `gorm:"-" json:"category_ids"`
	Categories  []string  `gorm:"-" json:"categories"`
	Latitude    float64   `gorm:"not null" json:"latitude"`
	Longitude   float64   `gorm:"not null" json:"longitude"`
	RadiusKm    int       `gorm:"default:5" json:"radius_km"`
	IsUrgent    bool      `gorm:"default:false" json:"is_urgent"`
	Status      string    `gorm:"default:'active'" json:"status"` // active, completed, expired
	MinBudget   *float64  `gorm:"column:min_budget" json:"min_budget,omitempty"`
	MaxBudget   *float64  `gorm:"column:max_budget" json:"max_budget,omitempty"`
	MaxOffers   *int      `gorm:"column:max_offers;default:5" json:"max_offers,omitempty"`
	OfferCount  int       `gorm:"column:offer_count;default:0" json:"offer_count"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Kolom location (PostGIS geometry) dikelola via raw SQL, tidak oleh GORM
	// location geometry(Point, 4326)

	// Relations
	Account          *Account             `gorm:"foreignKey:CustomerID;references:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"account,omitempty"`
	CategoryObj      *Category            `gorm:"foreignKey:CategoryID;references:CategoryID" json:"category_detail,omitempty"`
	SignalCategories []SignalCategory     `gorm:"foreignKey:SignalID;references:SignalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"signal_categories,omitempty"`
	Tags             []SignalTag          `gorm:"foreignKey:SignalID;references:SignalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags,omitempty"`
	Images           []SignalImage        `gorm:"foreignKey:SignalID;references:SignalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"images,omitempty"`
	Conversations    []SignalConversation `gorm:"foreignKey:SignalID;references:SignalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"conversations,omitempty"`
}

func (Signal) TableName() string { return "signals" }

// PopulateCategories mengisi CategoryIDs dan Categories dari SignalCategories atau fallback ke CategoryID tunggal
func (s *Signal) PopulateCategories() {
	if len(s.SignalCategories) > 0 {
		s.CategoryIDs = make([]int64, 0, len(s.SignalCategories))
		s.Categories = make([]string, 0, len(s.SignalCategories))
		for _, sc := range s.SignalCategories {
			s.CategoryIDs = append(s.CategoryIDs, sc.CategoryID)
			if sc.Category != nil && sc.Category.Name != "" {
				s.Categories = append(s.Categories, sc.Category.Name)
			}
		}
	} else if s.CategoryID != nil && *s.CategoryID > 0 {
		s.CategoryIDs = []int64{*s.CategoryID}
		if s.Category != "" {
			s.Categories = []string{s.Category}
		} else {
			s.Categories = []string{}
		}
	} else {
		s.CategoryIDs = []int64{}
		s.Categories = []string{}
	}
}

// SignalCategory adalah relasi many-to-many antara signal dan kategori bisnis (maks 5 kategori)
type SignalCategory struct {
	SignalID   string    `gorm:"primaryKey;column:signal_id" json:"signal_id"`
	CategoryID int64     `gorm:"primaryKey;column:category_id" json:"category_id"`
	Category   *Category `gorm:"foreignKey:CategoryID;references:CategoryID" json:"category,omitempty"`
}

func (SignalCategory) TableName() string { return "signal_categories" }

// SignalTag adalah relasi antara signal dan tag bisnis yang dipilih customer
type SignalTag struct {
	SignalID string `gorm:"primaryKey;column:signal_id" json:"signal_id"`
	TagID    int64  `gorm:"primaryKey;column:tag_id" json:"tag_id"`

	Tag *Tag `gorm:"foreignKey:TagID;references:TagID" json:"tag,omitempty"`
}

func (SignalTag) TableName() string { return "signal_tags" }

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
	Business *Business       `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Merchant *Account        `gorm:"foreignKey:MerchantID;references:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"merchant,omitempty"`
	Messages []SignalMessage `gorm:"foreignKey:ConversationID;references:ConversationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"messages,omitempty"`
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

	Conversation *SignalConversation `gorm:"foreignKey:ConversationID;references:ConversationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"conversation,omitempty"`
}

func (SignalMessage) TableName() string { return "signal_messages" }

// ================================================================
// INPUT STRUCTS (V2)
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
	CategoryID        *int64    `json:"category_id,omitempty"`
	Category          string    `json:"category"`
	CategoryIDs       []int64   `json:"category_ids"`
	Categories        []string  `json:"categories"`
	IsUrgent          bool      `json:"is_urgent"`
	Status            string    `json:"status"`
	MinBudget         *float64  `json:"min_budget,omitempty"`
	MaxBudget         *float64  `json:"max_budget,omitempty"`
	MaxOffers         *int      `json:"max_offers,omitempty"`
	OfferCount        int       `json:"offer_count"`
	ConversationCount int       `json:"conversation_count"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// SignalExploreItem untuk merchant explore (dengan jarak dan relevansi tag)
type SignalExploreItem struct {
	SignalID        string    `json:"signal_id"`
	CustomerID      string    `json:"customer_id"`
	Description     string    `json:"description"`
	CategoryID      *int64    `json:"category_id,omitempty"`
	Category        string    `json:"category"`
	CategoryIDs     []int64   `json:"category_ids"`
	Categories      []string  `json:"categories"`
	IsUrgent        bool      `json:"is_urgent"`
	RadiusKm        int       `json:"radius_km"`
	DistanceKm      float64   `json:"distance_km"`
	MinBudget       *float64  `json:"min_budget,omitempty"`
	MaxBudget       *float64  `json:"max_budget,omitempty"`
	MaxOffers       *int      `json:"max_offers,omitempty"`
	OfferCount      int       `json:"offer_count"`
	HasReplied      bool      `json:"has_replied"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	CustomerName    string    `json:"customer_name"`
	CustomerAvatar  *string   `json:"customer_avatar"`
	Images          []string  `json:"images"`
	Tags            []string  `json:"tags"`          // Nama tag yang dipilih customer
	TagOverlapScore int       `json:"relevance_score"` // Jumlah tag overlap dengan bisnis merchant
}

// SignalMerchantComparison untuk matriks perbandingan toko penawar bagi customer
type SignalMerchantComparison struct {
	BusinessID        string    `json:"business_id"`
	BusinessName      string    `json:"business_name"`
	LogoURL           *string   `json:"logo_url"`
	DistanceKm        float64   `json:"distance_km"`
	AverageRating     float64   `json:"average_rating"`
	ReviewCount       int64     `json:"review_count"`
	ConversationID    string    `json:"conversation_id"`
	FirstOfferMessage string    `json:"first_offer_message"`
	LastMessageAt     time.Time `json:"last_message_at"`
}

// CustomerSignalDetailResponse menyertakan sinyal dan perbandingan merchant
type CustomerSignalDetailResponse struct {
	Signal      *Signal                    `json:"signal"`
	Comparisons []SignalMerchantComparison `json:"merchant_comparisons"`
}

// ConversationListItem untuk daftar roomchat (Merchant / Customer)
type ConversationListItem struct {
	ConversationID    string    `json:"conversation_id"`
	SignalID          string    `json:"signal_id"`
	SignalDescription string    `json:"signal_description"`
	CategoryID        *int64    `json:"category_id,omitempty"`
	SignalCategory    string    `json:"signal_category"`
	SignalStatus      string    `json:"signal_status"`
	IsUrgent          bool      `json:"is_urgent"`
	
	// Lawan bicara
	PeerID            string    `json:"peer_id"`
	PeerName          string    `json:"peer_name"`
	PeerAvatar        *string   `json:"peer_avatar"`
	
	// Bisnis (Toko)
	BusinessID        string    `json:"business_id"`
	BusinessName      string    `json:"business_name"`
	BusinessLogo      *string   `json:"business_logo"`
	
	// Pesan terakhir & status
	LastMessage       *string   `json:"last_message"`
	LastMessageSender string    `json:"last_message_sender"` // "customer" | "merchant"
	LastMessageAt     time.Time `json:"last_message_at"`
	UnreadCount       int64     `json:"unread_count"`
	Status            string    `json:"status"` // "open" | "locked"
	CreatedAt         time.Time `json:"created_at"`
}

// MerchantSignalDetailResponse menyertakan status balasan merchant yang sedang login
type MerchantSignalDetailResponse struct {
	Signal           *Signal             `json:"signal"`
	HasReplied       bool                `json:"has_replied"`
	MyConversationID *string             `json:"my_conversation_id"`
	MyConversation   *SignalConversation `json:"my_conversation"`
}
