package models

import (
	"time"

	"gorm.io/gorm"
)

// ==========================================
// 1. INTERAKSI UNTUK PRODUK
// ==========================================

type ProductComment struct {
	CommentID            string         `gorm:"primaryKey;column:comment_id" json:"comment_id"`
	ProductID            string         `gorm:"column:product_id;index" json:"product_id"` // 👈 Foreign Key ke Product
	BusinessID           string         `gorm:"column:business_id;index" json:"business_id"`
	AccountID            string         `gorm:"column:account_id" json:"account_id"`
	Content              string         `gorm:"column:content" json:"content"`
	
	// Kunci Nesting Level 1
	ReplyToCommentID     *string        `gorm:"column:reply_to_comment_id;index" json:"reply_to_comment_id"`
	
	// 🪄 DITAMBAHKAN: Fitur Mention/Tagging (Untuk membalas child comment)
	RepliedToAccountID   *string        `gorm:"column:replied_to_account_id" json:"replied_to_account_id"`
	RepliedToAccountName *string        `gorm:"column:replied_to_account_name" json:"replied_to_account_name"`

	LikeCount            int            `gorm:"default:0;column:like_count" json:"like_count"`
	CreatedAt            time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	Account *Account         `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Replies []ProductComment `gorm:"foreignKey:ReplyToCommentID;references:CommentID" json:"replies,omitempty"`
}

type ProductLike struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	ProductID string    `gorm:"primaryKey;column:product_id" json:"product_id"`
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

	Account *Account               `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Replies []BusinessPostComment  `gorm:"foreignKey:ReplyToCommentID;references:CommentID" json:"replies,omitempty"`
}

type BusinessPostLike struct {
	PostID    string    `gorm:"primaryKey;column:post_id" json:"post_id"`
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// ==========================================
// 3. DTO (Data Transfer Object)
// ==========================================

// Karena strukturnya mirip, DTO Input bisa dipakai bersamaan untuk Product dan Post
type CreateCommentInput struct {
	Content              string  `json:"content" validate:"required"`
	ReplyToCommentID     *string `json:"reply_to_comment_id"`
	
	// 🪄 DITAMBAHKAN: Agar Frontend bisa mengirim data user yang di-mention
	RepliedToAccountID   *string `json:"replied_to_account_id"`
	RepliedToAccountName *string `json:"replied_to_account_name"`
}

// DTO Output (Response yang dikirim ke Frontend)
type CommentResponse struct {
	CommentID            string            `json:"comment_id"`
	AccountID            string            `json:"account_id"`
	AccountName          string            `json:"account_name"`
	Content              string            `json:"content"`
	LikeCount            int               `json:"like_count"`
	CreatedAt            time.Time         `json:"created_at"`
	
	// 🪄 DITAMBAHKAN omitempty: Agar JSON rapi (tidak muncul "null") jika ini komentar utama
	RepliedToAccountID   *string           `json:"replied_to_account_id,omitempty"`
	RepliedToAccountName *string           `json:"replied_to_account_name,omitempty"`
	
	Replies              []CommentResponse `json:"replies,omitempty"`
}