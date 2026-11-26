package location

import "time"

// ================================================================
// CORE MODELS
// ================================================================

// LocationUpdate represents client location telemetry over WebSocket.
type LocationUpdate struct {
	Lat       float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lng       float64 `json:"lng" validate:"required,min=-180,max=180"`
	Heading   float64 `json:"heading,omitempty"`   // Arah hadap dalam derajat (0-360)
	Speed     float64 `json:"speed,omitempty"`     // Kecepatan dalam km/h
	Accuracy  float64 `json:"accuracy,omitempty"`  // Akurasi GPS dalam meter
	Timestamp int64   `json:"timestamp,omitempty"` // Unix timestamp dari device
}

// Dispatch to customer.
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

// SubscribeAreaRequest represents client viewport subscription bounds.
type SubscribeAreaRequest struct {
	Lat      float64 `json:"lat" validate:"required"`
	Lng      float64 `json:"lng" validate:"required"`
	RadiusKm float64 `json:"radius_km,omitempty"` // Default: 5 km
}

// ================================================================
// WEBSOCKET MESSAGE TYPES
// ================================================================

// WSIncomingMessage represents client WebSocket payload format.
type WSIncomingMessage struct {
	Type string      `json:"type"` // LOCATION_UPDATE, SUBSCRIBE_AREA, UNSUBSCRIBE
	Data interface{} `json:"data,omitempty"`
}

// WSLocationEvent represents location updates dispatched to subscribers.
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
	// Redis GEO key for active merchant coordinates.
	KeyMerchantLocations = "live:merchants"

	// Redis HASH key for merchant metadata.
	// Fields: business_name, logo_url, is_open, store_type, updated_at
	KeyMerchantMetaPrefix = "live:meta:"

	// Redis set key for subscribed customers.
	KeyCustomerSubPrefix = "live:sub:"

	// TTL for cached merchant metadata in Redis.
	MerchantMetaTTL = 30 * time.Minute
)
