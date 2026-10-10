package models

import "time"

type RecentActivityResponse struct {
	ReviewID     string    `json:"review_id,omitempty"`
	ActivityType string    `json:"activity_type"` // "product_review" or "business_review"
	BusinessID   string    `json:"business_id"`
	BusinessName string    `json:"business_name"`
	StoreName    string    `json:"store_name"` // alias untuk kemudahan FE
	ProductID    string    `json:"product_id,omitempty"`
	ProductName  string    `json:"product_name,omitempty"`
	Rating       float64   `json:"rating"`
	Content      string    `json:"content"`
	MediaURLs    []string  `json:"media_urls"`
	ReviewImages []string  `json:"review_images,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	Business *Business `json:"business,omitempty"`
	Store    *Business `json:"store,omitempty"`
	Product  *Product  `json:"product,omitempty"`
}

type PublicProfileSummaryResponse struct {
	AccountID            string                   `json:"account_id"`
	Username             string                   `json:"username"`
	ProfilePictureURL    *string                  `json:"profile_picture_url"`
	TotalCommentsReviews int64                    `json:"total_comments_reviews"`
	RecentReviews        []RecentActivityResponse `json:"recent_reviews"`
}
