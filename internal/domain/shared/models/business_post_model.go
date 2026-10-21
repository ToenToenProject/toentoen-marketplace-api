package models

import "time"

// Model untuk Postingan
type BusinessPost struct {
	PostID         string     `gorm:"primaryKey;column:post_id" json:"post_id"`
	BusinessID     string     `gorm:"column:business_id" json:"business_id"`
	Content        string     `gorm:"column:content" json:"content"`
	Images         []string   `gorm:"type:jsonb;serializer:json;column:images" json:"images"`
	VideoURL       *string    `gorm:"column:video_url" json:"video_url"` // Pointer agar bisa null
	LikeCount      int        `gorm:"default:0;column:like_count" json:"like_count"`
	CommentCount   int        `gorm:"default:0;column:comment_count" json:"comment_count"`
	IsPromotion    bool       `gorm:"column:is_promotion;default:false" json:"is_promotion"`
	PromotionID    *string    `gorm:"column:promotion_id" json:"promotion_id,omitempty"`
	PromotionEndAt *time.Time `gorm:"column:promotion_end_at" json:"promotion_end_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
}

// Input Struct
type CreatePostInput struct {
	Content        string     `json:"content"`
	Images         []string   `json:"images"`
	VideoURL       string     `json:"video_url"`
	IsPromotion    *bool      `json:"is_promotion,omitempty"`
	DurationDays   *int       `json:"duration_days,omitempty"`
	PromotionEndAt *time.Time `json:"promotion_end_at,omitempty"`
}

type UpdatePostInput struct {
	Content        *string    `json:"content"`
	Images         *[]string  `json:"images" validate:"omitempty,max=5"`
	VideoURL       *string    `json:"video_url"`
	IsPromotion    *bool      `json:"is_promotion,omitempty"`
	DurationDays   *int       `json:"duration_days,omitempty"`
	PromotionEndAt *time.Time `json:"promotion_end_at,omitempty"`
}

type ExtendPostPromotionInput struct {
	AddDays *int       `json:"add_days"`
	EndAt   *time.Time `json:"end_at"`
}

type PostResponse struct {
	PostID         string     `json:"post_id"`
	BusinessID     string     `json:"business_id"`
	Content        string     `json:"content"`
	Images         []string   `json:"images"`
	VideoURL       *string    `json:"video_url,omitempty"`
	LikeCount      int        `json:"like_count"`
	CommentCount   int        `json:"comment_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	IsLikedByMe    bool       `json:"is_liked_by_me"`
	Business       *Business  `json:"business,omitempty"`
	IsPromotion    bool       `json:"is_promotion"`
	PromotionID    *string    `json:"promotion_id,omitempty"`
	PromotionEndAt *time.Time `json:"promotion_end_at,omitempty"`
	RemainingDays  *int       `json:"remaining_days,omitempty"`
	BadgeText      *string    `json:"badge_text,omitempty"`    // "Promosi" jika is_promotion = true
	TargetAction   *string    `json:"target_action,omitempty"` // "open_promotion_group" jika is_promotion = true
}
