package models

import "time"

type DemandSignal struct {
	SignalID    string    `gorm:"primaryKey;column:signal_id" json:"signal_id"`
	AccountID   string    `gorm:"not null" json:"account_id"`
	Category    string    `gorm:"not null" json:"category"`
	Description string    `json:"description"`
	MediaUrls   []string  `gorm:"type:json;serializer:json" json:"media_urls"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RadiusKm    int       `gorm:"default:5" json:"radius_km"`
	Status      string    `gorm:"default:'open'" json:"status"` // open, processing, completed
	IsFulfilled bool      `gorm:"default:false" json:"is_fulfilled"`
	FulfilledAt *time.Time `json:"fulfilled_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`

	Account   *Account         `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
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

// --- INPUT STRUCTS ---

type CreateSignalInput struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	MediaUrls   []string `json:"media_urls"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	RadiusKm    int      `json:"radius_km"`
}

type CreateResponseInput struct {
	BusinessID string   `json:"business_id"` // Owner merespons mewakili toko mana
	Message    string   `json:"message"`
	MediaUrls  []string `json:"media_urls"`
}