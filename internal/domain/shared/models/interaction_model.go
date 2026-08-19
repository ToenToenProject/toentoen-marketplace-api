package models

import (
	"time"

	"gorm.io/gorm"
)

// ==========================================
// 1. INTERAKSI UNTUK PRODUK (DISKUSI / Q&A)
// ==========================================

type ProductComment struct {
	CommentID            string         `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	ProductID            string         `gorm:"column:product_id;index" json:"product_id"`
	BusinessID           string         `gorm:"column:business_id;index" json:"business_id"`
	AccountID            string         `gorm:"column:account_id" json:"account_id"`
	Content              string         `gorm:"column:content" json:"content"`

	// Kunci Nesting Level 1
	ReplyToCommentID     *string        `gorm:"column:reply_to_comment_id;index" json:"reply_to_comment_id"`

	// Fitur Mention/Tagging (Untuk membalas child comment)
	RepliedToAccountID   *string        `gorm:"column:replied_to_account_id" json:"replied_to_account_id"`
	RepliedToAccountName *string        `gorm:"column:replied_to_account_name" json:"replied_to_account_name"`

	LikeCount            int            `gorm:"default:0;column:like_count" json:"like_count"`
	RepliesCount         int            `gorm:"default:0;column:replies_count" json:"replies_count"`
	CreatedAt            time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	Account  *Account         `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Product  *Product         `gorm:"foreignKey:ProductID;references:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"product,omitempty"`
	Business *Business        `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Replies  []ProductComment `gorm:"foreignKey:ReplyToCommentID;references:CommentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"replies,omitempty"`
}

type ProductLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	ProductID string    `gorm:"primaryKey;column:product_id" json:"product_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type ProductCommentLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CommentID string    `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// ==========================================
// 2. INTERAKSI UNTUK POST
// ==========================================

type BusinessPostComment struct {
	CommentID            string         `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	PostID               string         `gorm:"column:post_id;index" json:"post_id"`
	BusinessID           string         `gorm:"column:business_id;index" json:"business_id"`
	AccountID            string         `gorm:"column:account_id" json:"account_id"`
	Content              string         `gorm:"column:content" json:"content"`

	// Kunci Nesting Level 1
	ReplyToCommentID     *string        `gorm:"column:reply_to_comment_id;index" json:"reply_to_comment_id"`

	// Fitur Mention/Tagging (Untuk membalas child comment)
	RepliedToAccountID   *string        `gorm:"column:replied_to_account_id" json:"replied_to_account_id"`
	RepliedToAccountName *string        `gorm:"column:replied_to_account_name" json:"replied_to_account_name"`

	LikeCount            int            `gorm:"default:0;column:like_count" json:"like_count"`
	CreatedAt            time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	Account  *Account              `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Post     *BusinessPost         `gorm:"foreignKey:PostID;references:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"post,omitempty"`
	Business *Business             `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Replies  []BusinessPostComment `gorm:"foreignKey:ReplyToCommentID;references:CommentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"replies,omitempty"`
}

type BusinessPostLike struct {
	PostID    string    `gorm:"primaryKey;column:post_id" json:"post_id"`
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type BusinessPostCommentLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CommentID string    `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// ==========================================
// 3. DTO (Data Transfer Object)
// ==========================================

type CreateProductCommentInput struct {
	Content              string  `json:"content" validate:"required"`
	ReplyToCommentID     *string `json:"reply_to_comment_id"`
	RepliedToAccountID   *string `json:"replied_to_account_id"`
	RepliedToAccountName *string `json:"replied_to_account_name"`
}

type UpdateProductCommentInput struct {
	Content string `json:"content" validate:"required"`
}

type CreateCommentInput struct {
	ProductID            *string  `json:"product_id,omitempty"`
	Content              string   `json:"content" validate:"required"`
	ReplyToCommentID     *string  `json:"reply_to_comment_id"`
	RepliedToAccountID   *string  `json:"replied_to_account_id"`
	RepliedToAccountName *string  `json:"replied_to_account_name"`
	Rating               *float64 `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	MediaURLs            []string `json:"media_urls,omitempty"`
}

type UpdateCommentInput struct {
	Content   string   `json:"content" validate:"required"`
	Rating    *float64 `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	MediaURLs []string `json:"media_urls,omitempty"`
}

// DTO Output (Response yang dikirim ke Frontend untuk Komentar Produk & Post)
type CommentResponse struct {
	CommentID              string            `json:"comment_id"`
	AccountID              string            `json:"account_id"`
	AccountName            string            `json:"account_name"`
	AccountAvatar          *string           `json:"account_avatar,omitempty"`
	Content                string            `json:"content"`
	LikeCount              int               `json:"like_count"`
	CommentCount           int               `json:"comment_count"`
	RepliesCount           int               `json:"replies_count"`
	IsLikedByMe            bool              `json:"is_liked_by_me"`
	IsByOwner              bool              `json:"is_by_owner"`
	IsOwnerLiked           bool              `json:"is_owner_liked"`
	IsOwnerReplied         bool              `json:"is_owner_replied"`
	OwnerInteractionLabel  string            `json:"owner_interaction_label,omitempty"`
	Rating                 *float64          `json:"rating,omitempty"`
	MediaURLs              []string          `json:"media_urls,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	RepliedToAccountID     *string           `json:"replied_to_account_id,omitempty"`
	RepliedToAccountName   *string           `json:"replied_to_account_name,omitempty"`
	Replies                []CommentResponse `json:"replies,omitempty"`
}
