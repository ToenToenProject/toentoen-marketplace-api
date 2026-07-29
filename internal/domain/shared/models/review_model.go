package models

import (
	"time"
	"gorm.io/gorm"
)

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
	LikeCount    int        `gorm:"default:0;column:like_count" json:"like_count"`
	CommentCount int        `gorm:"default:0;column:comment_count" json:"comment_count"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`

	Account  *Account                `json:"account,omitempty"`
	Business *Business               `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Comments []BusinessReviewComment `gorm:"foreignKey:ReviewID;references:ReviewID" json:"comments,omitempty"`
}

type BusinessReviewLike struct {
	ReviewID  string    `gorm:"primaryKey;column:review_id" json:"review_id"`
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type BusinessReviewComment struct {
	CommentID            string         `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	ReviewID             string         `gorm:"column:review_id;index" json:"review_id"`
	BusinessID           string         `gorm:"column:business_id;index" json:"business_id"`
	AccountID            string         `gorm:"column:account_id" json:"account_id"`
	Content              string         `gorm:"column:content" json:"content"`
	ReplyToCommentID     *string        `gorm:"column:reply_to_comment_id;index" json:"reply_to_comment_id"`
	RepliedToAccountID   *string        `gorm:"column:replied_to_account_id" json:"replied_to_account_id"`
	RepliedToAccountName *string        `gorm:"column:replied_to_account_name" json:"replied_to_account_name"`
	LikeCount            int            `gorm:"default:0;column:like_count" json:"like_count"`
	CreatedAt            time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	Account *Account                `json:"account,omitempty"`
	Replies []BusinessReviewComment `gorm:"foreignKey:ReplyToCommentID;references:CommentID" json:"replies,omitempty"`
}

type BusinessReviewCommentLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CommentID string    `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
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