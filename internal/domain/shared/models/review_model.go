package models

import "time"

type Review struct {
	ReviewID     string     `gorm:"primaryKey;column:review_id" json:"review_id"`
	BusinessID   string     `gorm:"column:business_id" json:"business_id"`
	AccountID    string     `gorm:"column:account_id" json:"account_id"` // Customer
	Rating       int        `gorm:"column:rating" json:"rating"`
	Comment      string     `gorm:"column:comment" json:"comment"`
	ReviewImages []string   `gorm:"column:review_images;serializer:json;default:'[]'" json:"review_images"`
	ReviewVideo  *string    `gorm:"column:review_video" json:"review_video,omitempty"`
	ReplyComment *string    `gorm:"column:reply_comment" json:"reply_comment,omitempty"`
	RepliedAt    *time.Time `gorm:"column:replied_at" json:"replied_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// Struct untuk Customer membuat review
type CreateReviewInput struct {
	BusinessID string  `json:"business_id" validate:"required"`
	Rating     int     `json:"rating" validate:"required,min=1,max=5"`
	Comment    string  `json:"comment" validate:"required"`
	ReviewImages []string `json:"review_images" validate:"max=5"` 
	ReviewVideo  *string  `json:"review_video"`
}

// Struct untuk Owner membalas review
type ReplyReviewInput struct {
	ReplyComment string `json:"reply_comment" validate:"required"`
}

// Tambahkan di bawah struct Review
type RatingDistribution struct {
	Star5 int64 `json:"5"`
	Star4 int64 `json:"4"`
	Star3 int64 `json:"3"`
	Star2 int64 `json:"2"`
	Star1 int64 `json:"1"`
}

type ReviewSummary struct {
	AverageRating float64            `json:"average_rating"`
	TotalReviews  int64              `json:"total_reviews"`
	Distribution  RatingDistribution `json:"distribution"`
}