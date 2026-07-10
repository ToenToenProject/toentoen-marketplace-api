package models

import "time"

type RecentActivityResponse struct {
	ActivityType string    `json:"activity_type"` // "product_review" or "business_review"
	Content      string    `json:"content"`
	MediaURLs    []string  `json:"media_urls"`
	CreatedAt    time.Time `json:"created_at"`
}

type PublicProfileSummaryResponse struct {
	AccountID            string                   `json:"account_id"`
	Username             string                   `json:"username"`
	ProfilePictureURL    *string                  `json:"profile_picture_url"`
	TotalCommentsReviews int64                    `json:"total_comments_reviews"`
	RecentReviews        []RecentActivityResponse `json:"recent_reviews"`
}
