package models

import (
	"time"
)

// SearchSynonym stores dynamic synonyms in PostgreSQL for Meilisearch synchronization and SQL fallbacks.
type SearchSynonym struct {
	SynonymID     int64     `gorm:"primaryKey;autoIncrement;column:synonym_id" json:"synonym_id"`
	Term          string    `gorm:"column:term;uniqueIndex;not null" json:"term"`
	CanonicalTerm string    `gorm:"column:canonical_term;not null" json:"canonical_term"`
	Synonyms      []string  `gorm:"column:synonyms;type:jsonb;serializer:json;default:'[]'" json:"synonyms"`
	Dimension     string    `gorm:"column:dimension;default:'general'" json:"dimension"` // category, service, offering, facility, general
	IsActive      bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SearchSynonym) TableName() string {
	return "search_synonyms"
}

// SearchLog tracks user search queries for zero-result monitoring and CTR analytics.
type SearchLog struct {
	LogID             string                 `gorm:"primaryKey;column:log_id" json:"log_id"`
	AccountID         *string                `gorm:"column:account_id" json:"account_id,omitempty"`
	RawQuery          string                 `gorm:"column:raw_query;not null" json:"raw_query"`
	CleanQuery        string                 `gorm:"column:clean_query" json:"clean_query"`
	ParsedIntent      map[string]interface{} `gorm:"column:parsed_intent;type:jsonb;serializer:json;default:'{}'" json:"parsed_intent"`
	AppliedFilters    map[string]interface{} `gorm:"column:applied_filters;type:jsonb;serializer:json;default:'{}'" json:"applied_filters"`
	ResultCount       int                    `gorm:"column:result_count;default:0" json:"result_count"`
	IsZeroResult      bool                   `gorm:"column:is_zero_result;default:false" json:"is_zero_result"`
	ClickedBusinessID *string                `gorm:"column:clicked_business_id" json:"clicked_business_id,omitempty"`
	Latitude          *float64               `gorm:"column:latitude" json:"latitude,omitempty"`
	Longitude         *float64               `gorm:"column:longitude" json:"longitude,omitempty"`
	CreatedAt         time.Time              `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SearchLog) TableName() string {
	return "search_logs"
}

// SearchClickInput represents telemetry for search result clicks.
type SearchClickInput struct {
	SearchLogID string `json:"search_log_id" validate:"required"`
	BusinessID  string `json:"business_id" validate:"required"`
}

// UpsertSynonymInput represents administrator payload for managing dynamic search synonyms.
type UpsertSynonymInput struct {
	Term          string   `json:"term" validate:"required"`
	CanonicalTerm string   `json:"canonical_term" validate:"required"`
	Synonyms      []string `json:"synonyms" validate:"required"`
	Dimension     string   `json:"dimension"` // category, service, offering, facility, general
	IsActive      *bool    `json:"is_active"`
}
