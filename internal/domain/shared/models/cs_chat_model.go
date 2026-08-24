package models

import (
	"time"
)

// CSChatRoom merepresentasikan sebuah room percakapan live chat antara user dan tim CS ToenToen
type CSChatRoom struct {
	RoomID             string     `gorm:"primaryKey;type:varchar(50)" json:"room_id"`
	AccountID          string     `gorm:"type:varchar(50);not null;index" json:"account_id"`
	UserRole           string     `gorm:"type:varchar(20);not null" json:"user_role"` // customer, owner, business_admin
	BusinessID         *string    `gorm:"type:varchar(50);index" json:"business_id,omitempty"`
	CSAdminID          *string    `gorm:"type:varchar(50);index" json:"cs_admin_id,omitempty"`
	Status             string     `gorm:"type:varchar(20);not null;default:'open';index" json:"status"` // open, in_progress, ended
	UserHidden         bool       `gorm:"not null;default:false;index" json:"user_hidden"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`
	EndedBy            *string    `gorm:"type:varchar(20)" json:"ended_by,omitempty"` // user, cs, system
	Rating             *int       `json:"rating,omitempty"`                           // 1-5
	Feedback           *string    `gorm:"type:text" json:"feedback,omitempty"`
	LastMessagePreview string     `gorm:"type:text" json:"last_message_preview"`
	LastMessageAt      time.Time  `gorm:"index" json:"last_message_at"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relasi
	Account  *Account        `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Business *Business       `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	CSAdmin  *Account        `gorm:"foreignKey:CSAdminID;references:AccountID" json:"cs_admin,omitempty"`
	Messages []CSChatMessage `gorm:"foreignKey:RoomID;references:RoomID" json:"messages,omitempty"`
}

func (CSChatRoom) TableName() string {
	return "cs_chat_rooms"
}

// CSChatMessage merepresentasikan satu pesan dalam room live chat CS
type CSChatMessage struct {
	MessageID   string     `gorm:"primaryKey;type:varchar(50)" json:"message_id"`
	RoomID      string     `gorm:"type:varchar(50);not null;index" json:"room_id"`
	SenderID    string     `gorm:"type:varchar(50);not null;index" json:"sender_id"` // account_id atau "SYSTEM"
	SenderType  string     `gorm:"type:varchar(20);not null" json:"sender_type"`     // user, cs, system
	SenderName  string     `gorm:"type:varchar(100);not null" json:"sender_name"`
	MessageType string     `gorm:"type:varchar(20);not null;default:'text'" json:"message_type"` // text, image, template, system_event
	Content     string     `gorm:"type:text;not null" json:"content"`
	MediaURL    *string    `gorm:"type:text" json:"media_url,omitempty"`
	IsRead      bool       `gorm:"not null;default:false" json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
}

func (CSChatMessage) TableName() string {
	return "cs_chat_messages"
}

// Input Models
type InitiateCSChatInput struct {
	BusinessID *string `json:"business_id,omitempty"`
}

type SendCSMessageInput struct {
	Content     string  `json:"content" validate:"required"`
	MediaURL    *string `json:"media_url,omitempty"`
	MessageType string  `json:"message_type,omitempty"` // default "text"
}

type EndCSChatInput struct {
	Rating   *int    `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	Feedback *string `json:"feedback,omitempty"`
}

type AssignCSChatInput struct {
	CSAdminID string `json:"cs_admin_id" validate:"required"`
}
