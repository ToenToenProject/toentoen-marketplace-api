package models

import (
	"time"
)

// CSChatSession merepresentasikan sebuah sesi live chat antara user dan tim CS ToenToen
type CSChatSession struct {
	SessionID          string     `gorm:"primaryKey;column:session_id;type:varchar(50)" json:"session_id"`
	AccountID          string     `gorm:"column:account_id;type:varchar(50);not null;index" json:"account_id"`
	UserRole           string     `gorm:"column:user_role;type:varchar(20);not null" json:"user_role"` // customer, owner, business_admin
	BusinessID         *string    `gorm:"column:business_id;type:varchar(50);index" json:"business_id,omitempty"`
	CSAdminID          *string    `gorm:"column:cs_admin_id;type:varchar(50);index" json:"cs_admin_id,omitempty"`
	Status             string     `gorm:"column:status;type:varchar(20);not null;default:'open';index" json:"status"` // open, in_progress, ended
	UserHidden         bool       `gorm:"column:user_hidden;not null;default:false;index" json:"user_hidden"`
	EndedAt            *time.Time `gorm:"column:ended_at" json:"ended_at,omitempty"`
	EndedBy            *string    `gorm:"column:ended_by;type:varchar(20)" json:"ended_by,omitempty"` // user, cs, system
	Rating             *int       `gorm:"column:rating" json:"rating,omitempty"`                      // 1-5
	Feedback           *string    `gorm:"column:feedback;type:text" json:"feedback,omitempty"`
	LastMessagePreview string     `gorm:"column:last_message_preview;type:text" json:"last_message_preview"`
	LastMessageAt      time.Time  `gorm:"column:last_message_at;index" json:"last_message_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relasi
	Account  *Account        `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Business *Business       `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	CSAdmin  *Account        `gorm:"foreignKey:CSAdminID;references:AccountID" json:"cs_admin,omitempty"`
	Messages []CSChatMessage `gorm:"foreignKey:SessionID;references:SessionID" json:"messages,omitempty"`
}

func (CSChatSession) TableName() string {
	return "cs_chat_sessions"
}

// CSChatRoom adalah type alias untuk backward compatibility
type CSChatRoom = CSChatSession

// CSChatMessage merepresentasikan satu pesan dalam sesi live chat CS
type CSChatMessage struct {
	MessageID   string     `gorm:"primaryKey;column:message_id;type:varchar(50)" json:"message_id"`
	SessionID   string     `gorm:"column:session_id;type:varchar(50);not null;index" json:"session_id"`
	SenderID    string     `gorm:"column:sender_id;type:varchar(50);not null;index" json:"sender_id"` // account_id atau "SYSTEM"
	SenderType  string     `gorm:"column:sender_type;type:varchar(20);not null" json:"sender_type"`   // user, cs, system
	SenderName  string     `gorm:"column:sender_name;type:varchar(100);not null" json:"sender_name"`
	MessageType string     `gorm:"column:message_type;type:varchar(20);not null;default:'text'" json:"message_type"` // text, image, template, system_event
	Content     string     `gorm:"column:content;type:text;not null" json:"content"`
	MediaURL    *string    `gorm:"column:media_url;type:text" json:"media_url,omitempty"`
	IsRead      bool       `gorm:"column:is_read;not null;default:false" json:"is_read"`
	ReadAt      *time.Time `gorm:"column:read_at" json:"read_at,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
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
