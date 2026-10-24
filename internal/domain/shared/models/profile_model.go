package models

import "time"

type RecentActivityResponse struct {
	ReviewID     string    `json:"review_id,omitempty"`
	ActivityType string    `json:"activity_type"` // "product_review" or "business_review"
	Content      string    `json:"content"`
	Rating       float64   `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	BusinessName string    `json:"business_name,omitempty"`
	ProductName  string    `json:"product_name,omitempty"`

	BusinessID   string    `json:"business_id,omitempty"`
	ProductID    string    `json:"product_id,omitempty"`
	StoreName    string    `json:"store_name,omitempty"` // alias untuk kemudahan FE
	MediaURLs    []string  `json:"media_urls,omitempty"`
	ReviewImages []string  `json:"review_images,omitempty"`

	Business *Business `json:"business,omitempty"`
	Store    *Business `json:"store,omitempty"`
	Product  *Product  `json:"product,omitempty"`
}

type PublicProfileSummaryResponse struct {
	AccountID            string                   `json:"account_id"`
	Username             string                   `json:"username"`
	FullName             string                   `json:"full_name"`
	Gender               string                   `json:"gender"`
	ProfilePictureURL    *string                  `json:"profile_picture_url"`
	TotalCommentsReviews int64                    `json:"total_comments_reviews"`
	RecentReviews        []RecentActivityResponse `json:"recent_reviews"`
}
