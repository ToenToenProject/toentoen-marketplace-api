package models

import "time"

// 1. Model untuk tabel PostgreSQL
type MerchantAnalytics struct {
	LogID       int64     `gorm:"primaryKey;autoIncrement;column:log_id" json:"log_id"`
	BusinessID  string    `gorm:"not null;index;column:business_id" json:"business_id"`
	EventType   string    `gorm:"not null;column:event_type" json:"event_type"` 
	SourceType  string    `gorm:"not null;column:source_type" json:"source_type"` 
	SearchedTag *string   `gorm:"column:searched_tag" json:"searched_tag,omitempty"` 
	ProductID   *string   `gorm:"column:product_id" json:"product_id,omitempty"`     
	CreatedAt   time.Time `gorm:"index;column:created_at" json:"created_at"`
}

func (MerchantAnalytics) TableName() string {
	return "merchant_analytics"
}

// 2. Struct untuk menyusun JSON Response
type DailyAnalyticsResponse struct {
	TotalVisitors   int64               `json:"total_visitors"`
	TotalImpression int64               `json:"total_impression"`
	SignalReceived  int64               `json:"signal_received"`
	SignalResponded int64               `json:"signal_responded"`
	TrafficSources  []TrafficSourceStat `json:"traffic_sources"`
	VisitorByTags   []TagStat           `json:"visitor_by_tags"`
	VisitorByProd   []ProductStat       `json:"visitor_by_products"`
}

type TrafficSourceStat struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}
type TagStat struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}
type ProductStat struct {
	ProductName string `json:"product_name"`
	Count       int64  `json:"count"`
}

// Struct Response untuk History Page
type HistoryAnalyticsResponse struct {
	TotalVisitors    int64               `json:"total_visitors"`
	VisitorsGrowth   float64             `json:"visitors_growth"`   // Contoh: 80.5 (berarti +80.5%)
	TotalImpression  int64               `json:"total_impression"`
	ImpressionGrowth float64             `json:"impression_growth"` 
	TrafficSources   []TrafficSourceStat `json:"traffic_sources"`
	TrafficGrowth    float64             `json:"traffic_growth"`
	VisitorByTags    []TagStat           `json:"visitor_by_tags"`
	VisitorByProd    []ProductStat       `json:"visitor_by_products"`
}

// 5. Struct Response untuk Explore Page
type ExploreAnalyticsResponse struct {
	TagsSimilarities    []TagSimilarity     `json:"tags_similarities"`
	ProductSimilarities []ProductSimilarity `json:"product_similarities"`
	CompetitorsRelated  []CompetitorStat    `json:"competitors_related"`
}

type TagSimilarity struct {
	Tag           string `json:"tag"`
	AudienceCount int64  `json:"audience_count"` // Berapa banyak orang yang nyari tag ini di radius 5km
}

type ProductSimilarity struct {
	ProductName   string `json:"product_name"`
	AudienceCount int64  `json:"audience_count"`
}

type CompetitorStat struct {
	BusinessID    string `json:"business_id"`
	BusinessName  string `json:"business_name"`
	AudienceCount int64  `json:"audience_count"`
}