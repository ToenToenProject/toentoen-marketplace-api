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

	Account  *Account                `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Business *Business               `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Comments []BusinessReviewComment `gorm:"foreignKey:ReviewID;references:ReviewID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"comments,omitempty"`
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

	Account  *Account                `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Business *Business               `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Replies  []BusinessReviewComment `gorm:"foreignKey:ReplyToCommentID;references:CommentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"replies,omitempty"`
}

type BusinessReviewCommentLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CommentID string    `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// Struct untuk Customer membuat review bisnis
type CreateReviewInput struct {
	BusinessID   string   `json:"business_id" validate:"required"`
	Rating       int      `json:"rating" validate:"required,min=1,max=5"`
	Comment      string   `json:"comment" validate:"required"`
	ReviewImages []string `json:"review_images" validate:"max=5"`
	ReviewVideo  *string  `json:"review_video"`
}

// Struct untuk Customer mengedit review bisnis
type UpdateReviewInput struct {
	Rating       int      `json:"rating" validate:"required,min=1,max=5"`
	Comment      string   `json:"comment" validate:"required"`
	ReviewImages []string `json:"review_images" validate:"max=5"`
	ReviewVideo  *string  `json:"review_video"`
}

// Struct untuk membalas review bisnis
type CreateReviewReplyInput struct {
	Content              string  `json:"content" validate:"required"`
	ReplyToCommentID     *string `json:"reply_to_comment_id"`
	RepliedToAccountID   *string `json:"replied_to_account_id"`
	RepliedToAccountName *string `json:"replied_to_account_name"`
}

// Struct untuk Owner membalas review langsung
type ReplyReviewInput struct {
	ReplyComment string `json:"reply_comment" validate:"required"`
}

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

// ==========================================
// 4. MODEL GORM UNTUK REVIEW PRODUK
// ==========================================

type ProductReview struct {
	ReviewID     string     `gorm:"primaryKey;column:review_id" json:"review_id"`
	ProductID    string     `gorm:"column:product_id;index" json:"product_id"`
	BusinessID   string     `gorm:"column:business_id;index" json:"business_id"`
	AccountID    string     `gorm:"column:account_id;index" json:"account_id"`
	Rating       float64    `gorm:"column:rating" json:"rating"`
	Comment      string     `gorm:"column:comment" json:"comment"`
	ReviewImages []string   `gorm:"column:review_images;serializer:json;default:'[]'" json:"review_images"`
	ReviewVideo  *string    `gorm:"column:review_video" json:"review_video,omitempty"`
	LikeCount    int        `gorm:"default:0;column:like_count" json:"like_count"`
	CommentCount int        `gorm:"default:0;column:comment_count" json:"comment_count"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`

	Account  *Account               `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Product  *Product               `gorm:"foreignKey:ProductID;references:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"product,omitempty"`
	Business *Business              `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Comments []ProductReviewComment `gorm:"foreignKey:ReviewID;references:ReviewID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"comments,omitempty"`
}

type ProductReviewLike struct {
	ReviewID  string    `gorm:"primaryKey;column:review_id" json:"review_id"`
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type ProductReviewComment struct {
	CommentID            string         `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	ReviewID             string         `gorm:"column:review_id;index" json:"review_id"`
	AccountID            string         `gorm:"column:account_id" json:"account_id"`
	Content              string         `gorm:"column:content" json:"content"`
	ReplyToCommentID     *string        `gorm:"column:reply_to_comment_id;index" json:"reply_to_comment_id"`
	RepliedToAccountID   *string        `gorm:"column:replied_to_account_id" json:"replied_to_account_id"`
	RepliedToAccountName *string        `gorm:"column:replied_to_account_name" json:"replied_to_account_name"`
	LikeCount            int            `gorm:"default:0;column:like_count" json:"like_count"`
	CreatedAt            time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	Account *Account               `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Replies []ProductReviewComment `gorm:"foreignKey:ReplyToCommentID;references:CommentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"replies,omitempty"`
}

type ProductReviewCommentLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CommentID string    `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type ProductReviewCommentReport struct {
	ReportID  string    `gorm:"primaryKey;column:report_id" json:"report_id"`
	CommentID string    `gorm:"column:comment_id" json:"comment_id"`
	AccountID string    `gorm:"column:account_id" json:"account_id"`
	Reason    string    `gorm:"column:reason" json:"reason"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// DTO Input untuk Review Produk
type CreateProductReviewInput struct {
	Rating       float64  `json:"rating" validate:"required,min=1,max=5"`
	Comment      string   `json:"comment" validate:"required"`
	ReviewImages []string `json:"review_images" validate:"max=5"`
	ReviewVideo  *string  `json:"review_video"`
}

type UpdateProductReviewInput struct {
	Rating       float64  `json:"rating" validate:"required,min=1,max=5"`
	Comment      string   `json:"comment" validate:"required"`
	ReviewImages []string `json:"review_images" validate:"max=5"`
	ReviewVideo  *string  `json:"review_video"`
}

type CreateProductReviewReplyInput struct {
	Content              string  `json:"content" validate:"required"`
	ReplyToCommentID     *string `json:"reply_to_comment_id"`
	RepliedToAccountID   *string `json:"replied_to_account_id"`
	RepliedToAccountName *string `json:"replied_to_account_name"`
}

// DTO Output Response
type BusinessReviewResponse struct {
	ReviewID               string                          `json:"review_id"`
	BusinessID             string                          `json:"business_id"`
	AccountID              string                          `json:"account_id"`
	AccountName            string                          `json:"account_name"`
	AccountAvatar          *string                         `json:"account_avatar,omitempty"`
	Rating                 int                             `json:"rating"`
	Comment                string                          `json:"comment"`
	ReviewImages           []string                        `json:"review_images"`
	ReviewVideo            *string                         `json:"review_video,omitempty"`
	LikeCount              int                             `json:"like_count"`
	CommentCount           int                             `json:"comment_count"`
	IsLikedByMe            bool                            `json:"is_liked_by_me"`
	IsOwnerLiked           bool                            `json:"is_owner_liked"`
	IsOwnerReplied         bool                            `json:"is_owner_replied"`
	OwnerInteractionLabel  string                          `json:"owner_interaction_label,omitempty"`
	Replies                []BusinessReviewCommentResponse `json:"replies,omitempty"`
	CreatedAt              time.Time                       `json:"created_at"`
	UpdatedAt              time.Time                       `json:"updated_at"`
}

type BusinessReviewCommentResponse struct {
	CommentID              string                          `json:"comment_id"`
	ReviewID               string                          `json:"review_id"`
	AccountID              string                          `json:"account_id"`
	AccountName            string                          `json:"account_name"`
	AccountAvatar          *string                         `json:"account_avatar,omitempty"`
	Content                string                          `json:"content"`
	LikeCount              int                             `json:"like_count"`
	IsLikedByMe            bool                            `json:"is_liked_by_me"`
	IsByOwner              bool                            `json:"is_by_owner"`
	IsOwnerLiked           bool                            `json:"is_owner_liked"`
	OwnerInteractionLabel  string                          `json:"owner_interaction_label,omitempty"`
	ReplyToCommentID       *string                         `json:"reply_to_comment_id,omitempty"`
	RepliedToAccountID     *string                         `json:"replied_to_account_id,omitempty"`
	RepliedToAccountName   *string                         `json:"replied_to_account_name,omitempty"`
	Replies                []BusinessReviewCommentResponse `json:"replies,omitempty"`
	CreatedAt              time.Time                       `json:"created_at"`
}

type ProductReviewResponse struct {
	ReviewID               string                         `json:"review_id"`
	ProductID              string                         `json:"product_id"`
	BusinessID             string                         `json:"business_id"`
	AccountID              string                         `json:"account_id"`
	AccountName            string                         `json:"account_name"`
	AccountAvatar          *string                        `json:"account_avatar,omitempty"`
	Rating                 float64                        `json:"rating"`
	Comment                string                         `json:"comment"`
	ReviewImages           []string                       `json:"review_images"`
	ReviewVideo            *string                        `json:"review_video,omitempty"`
	LikeCount              int                            `json:"like_count"`
	CommentCount           int                            `json:"comment_count"`
	IsLikedByMe            bool                           `json:"is_liked_by_me"`
	IsOwnerLiked           bool                           `json:"is_owner_liked"`
	IsOwnerReplied         bool                           `json:"is_owner_replied"`
	OwnerInteractionLabel  string                         `json:"owner_interaction_label,omitempty"`
	Replies                []ProductReviewCommentResponse `json:"replies,omitempty"`
	CreatedAt              time.Time                      `json:"created_at"`
	UpdatedAt              time.Time                      `json:"updated_at"`
}

type ProductReviewCommentResponse struct {
	CommentID              string                         `json:"comment_id"`
	ReviewID               string                         `json:"review_id"`
	AccountID              string                         `json:"account_id"`
	AccountName            string                         `json:"account_name"`
	AccountAvatar          *string                        `json:"account_avatar,omitempty"`
	Content                string                         `json:"content"`
	LikeCount              int                            `json:"like_count"`
	IsLikedByMe            bool                           `json:"is_liked_by_me"`
	IsByOwner              bool                           `json:"is_by_owner"`
	IsOwnerLiked           bool                           `json:"is_owner_liked"`
	OwnerInteractionLabel  string                         `json:"owner_interaction_label,omitempty"`
	ReplyToCommentID       *string                        `json:"reply_to_comment_id,omitempty"`
	RepliedToAccountID     *string                        `json:"replied_to_account_id,omitempty"`
	RepliedToAccountName   *string                        `json:"replied_to_account_name,omitempty"`
	Replies                []ProductReviewCommentResponse `json:"replies,omitempty"`
	CreatedAt              time.Time                      `json:"created_at"`
}
