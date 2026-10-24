package models

import (
	"time"
)

// DirectBusinessConversation merepresentasikan sesi chat antara 1 Customer dan 1 Toko (Business)
type DirectBusinessConversation struct {
	ConversationID     string    `gorm:"primaryKey;column:conversation_id;type:varchar(50)" json:"conversation_id"`
	BusinessID         string    `gorm:"column:business_id;type:varchar(50);not null;index" json:"business_id"`
	CustomerID         string    `gorm:"column:customer_id;type:varchar(50);not null;index" json:"customer_id"`
	SignalID           *string   `gorm:"column:signal_id;type:varchar(50);index" json:"signal_id,omitempty"`
	LastMessagePreview string    `gorm:"column:last_message_preview;type:text" json:"last_message_preview"`
	LastMessageAt      time.Time `gorm:"column:last_message_at;index" json:"last_message_at"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relasi
	Business *Business               `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Customer *Account                `gorm:"foreignKey:CustomerID;references:AccountID" json:"customer,omitempty"`
	Signal   *Signal                 `gorm:"foreignKey:SignalID;references:SignalID" json:"signal,omitempty"`
	Messages []DirectBusinessMessage `gorm:"foreignKey:ConversationID;references:ConversationID" json:"messages,omitempty"`
}

func (DirectBusinessConversation) TableName() string {
	return "direct_business_conversations"
}

// DirectBusinessMessage merepresentasikan pesan dalam percakapan toko
type DirectBusinessMessage struct {
	MessageID      string     `gorm:"primaryKey;column:message_id;type:varchar(50)" json:"message_id"`
	ConversationID string     `gorm:"column:conversation_id;type:varchar(50);not null;index" json:"conversation_id"`
	SenderID       string     `gorm:"column:sender_id;type:varchar(50);not null;index" json:"sender_id"`
	SenderType     string     `gorm:"column:sender_type;type:varchar(20);not null" json:"sender_type"` // 'customer' atau 'merchant'
	SenderName     string     `gorm:"column:sender_name;type:varchar(100);not null" json:"sender_name"`
	MessageType    string     `gorm:"column:message_type;type:varchar(20);not null;default:'text'" json:"message_type"` // 'text', 'image', 'product_inquiry', 'system_event'
	Content        string     `gorm:"column:content;type:text;not null" json:"content"`
	MediaURL       *string    `gorm:"column:media_url;type:text" json:"media_url,omitempty"`
	ProductID      *string    `gorm:"column:product_id;type:varchar(50);index" json:"product_id,omitempty"`
	IsRead         bool       `gorm:"column:is_read;not null;default:false" json:"is_read"`
	ReadAt         *time.Time `gorm:"column:read_at" json:"read_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`

	// Relasi
	ProductDetail *Product `gorm:"foreignKey:ProductID;references:ProductID" json:"product_detail,omitempty"`
}

func (DirectBusinessMessage) TableName() string {
	return "direct_business_messages"
}

// Input Models
type InitiateBusinessChatInput struct {
	Message   string  `json:"message"`
	ProductID *string `json:"product_id,omitempty"`
	SignalID  *string `json:"signal_id,omitempty"`
}

type SendBusinessChatMessageInput struct {
	Content     string  `json:"content" validate:"required"`
	MediaURL    *string `json:"media_url,omitempty"`
	ProductID   *string `json:"product_id,omitempty"`
	MessageType string  `json:"message_type,omitempty"` // default 'text'
}

// Response List Item
type CustomerBusinessConversationListItem struct {
	ConversationID      string               `json:"conversation_id"`
	BusinessID          string               `json:"business_id"`
	BusinessName        string               `json:"business_name"`
	BusinessLogo        *string              `json:"business_logo,omitempty"`
	LastMessagePreview  string               `json:"last_message_preview"`
	LastMessageAt       time.Time            `json:"last_message_at"`
	UnreadCount         int64                `json:"unread_count"`
	SignalID            string               `json:"signal_id,omitempty"`
	SignalTitle         string               `json:"signal_title,omitempty"`
	SignalDescription   string               `json:"signal_description,omitempty"`
	SignalPostReference *SignalPostReference `json:"signal_post_reference,omitempty"`
}

type MerchantBusinessConversationListItem struct {
	ConversationID      string               `json:"conversation_id"`
	BusinessID          string               `json:"business_id"`
	CustomerID          string               `json:"customer_id"`
	CustomerName        string               `json:"customer_name"`
	CustomerAvatar      *string              `json:"customer_avatar,omitempty"`
	LastMessagePreview  string               `json:"last_message_preview"`
	LastMessageAt       time.Time            `json:"last_message_at"`
	UnreadCount         int64                `json:"unread_count"`
	SignalID            string               `json:"signal_id,omitempty"`
	SignalTitle         string               `json:"signal_title,omitempty"`
	SignalDescription   string               `json:"signal_description,omitempty"`
	SignalPostReference *SignalPostReference `json:"signal_post_reference,omitempty"`
}
