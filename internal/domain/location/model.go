package location

import "time"

// ================================================================
// CORE MODELS
// ================================================================

// LocationUpdate adalah payload yang dikirim oleh customer via WebSocket
type LocationUpdate struct {
	Lat       float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lng       float64 `json:"lng" validate:"required,min=-180,max=180"`
	Heading   float64 `json:"heading,omitempty"`   // Arah hadap dalam derajat (0-360)
	Speed     float64 `json:"speed,omitempty"`     // Kecepatan dalam km/h
	Accuracy  float64 `json:"accuracy,omitempty"`  // Akurasi GPS dalam meter
	Timestamp int64   `json:"timestamp,omitempty"` // Unix timestamp dari device
}

// NearbyMerchant adalah hasil query merchant terdekat yang dikirim ke customer
type NearbyMerchant struct {
	BusinessID   string  `json:"business_id"`
	BusinessName string  `json:"business_name"`
	LogoURL      string  `json:"logo_url,omitempty"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	DistanceKm   float64 `json:"distance_km"`
	IsOpen       bool    `json:"is_open"`
	StoreType    string  `json:"store_type,omitempty"`
}

// SubscribeAreaRequest adalah payload customer untuk subscribe update di area tertentu
type SubscribeAreaRequest struct {
	Lat      float64 `json:"lat" validate:"required"`
	Lng      float64 `json:"lng" validate:"required"`
	RadiusKm float64 `json:"radius_km,omitempty"` // Default: 5 km
}

// ================================================================
// WEBSOCKET MESSAGE TYPES
// ================================================================

// WSIncomingMessage adalah format pesan masuk dari client WebSocket
type WSIncomingMessage struct {
	Type string      `json:"type"` // LOCATION_UPDATE, SUBSCRIBE_AREA, UNSUBSCRIBE
	Data interface{} `json:"data,omitempty"`
}

// WSLocationEvent adalah event yang dikirim ke subscriber saat ada perubahan lokasi
type WSLocationEvent struct {
	Type string      `json:"type"` // NEARBY_UPDATE, MERCHANT_ONLINE, MERCHANT_OFFLINE
	Data interface{} `json:"data"`
}

// ================================================================
// THROTTLE STATE (Server-Side per Connection)
// ================================================================

// ThrottleState melacak status throttle per koneksi customer
type ThrottleState struct {
	LastLat         float64
	LastLng         float64
	LastSentAt      time.Time
	IsStationary    bool
	StationarySince time.Time
	UpdateCount     int64 // Total update yang diterima (untuk rate limiting)
	LastUpdateAt    time.Time
}

// ================================================================
// REDIS KEY PATTERNS
// ================================================================

const (
	// Redis GEO key untuk lokasi semua merchant aktif
	KeyMerchantLocations = "live:merchants"

	// Redis HASH key untuk metadata merchant (prefix + business_id)
	// Fields: business_name, logo_url, is_open, store_type, updated_at
	KeyMerchantMetaPrefix = "live:meta:"

	// Redis key untuk menyimpan set customer yang sedang subscribe
	KeyCustomerSubPrefix = "live:sub:"

	// TTL untuk metadata merchant di Redis (sync ulang dari DB jika expired)
	MerchantMetaTTL = 30 * time.Minute
)
