package models

import (
	"time"
)

// SupportTicket merepresentasikan tiket bantuan / keluhan pengguna
type SupportTicket struct {
	TicketID                   string     `gorm:"primaryKey;column:ticket_id;type:varchar(50)" json:"ticket_id"`
	AccountID                  string     `gorm:"column:account_id;type:varchar(50);not null;index" json:"account_id"`
	UserRole                   string     `gorm:"column:user_role;type:varchar(20);not null" json:"user_role"` // customer, owner, business_admin
	BusinessID                 *string    `gorm:"column:business_id;type:varchar(50);index" json:"business_id,omitempty"`
	Category                   string     `gorm:"column:category;type:varchar(100);not null;index" json:"category"`
	Description                string     `gorm:"column:description;type:text;not null" json:"description"`
	SupportTicketStatus        string     `gorm:"column:support_ticket_status;type:varchar(20);not null;default:'pending';index" json:"support_ticket_status"` // pending, in_progress, solved, closed
	CustomerServiceSuggestions *string    `gorm:"column:customer_service_suggestions;type:text" json:"customer_service_suggestions,omitempty"`
	ResolvedAt                 *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy                 *string    `gorm:"column:resolved_by;type:varchar(50);index" json:"resolved_by,omitempty"`
	CreatedAt                  time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt                  time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relasi
	Account     *Account                  `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Business    *Business                 `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Resolver    *Account                  `gorm:"foreignKey:ResolvedBy;references:AccountID" json:"resolver,omitempty"`
	Attachments []SupportTicketAttachment `gorm:"foreignKey:TicketID;references:TicketID" json:"attachments,omitempty"`
}

func (SupportTicket) TableName() string {
	return "support_tickets"
}

// SupportTicketAttachment merepresentasikan lampiran file pada tiket
type SupportTicketAttachment struct {
	AttachmentID string    `gorm:"primaryKey;column:attachment_id;type:varchar(50)" json:"attachment_id"`
	TicketID     string    `gorm:"column:ticket_id;type:varchar(50);not null;index" json:"ticket_id"`
	FileURL      string    `gorm:"column:file_url;type:text;not null" json:"file_url"`
	FileName     string    `gorm:"column:file_name;type:varchar(255);not null" json:"file_name"`
	FileSize     int64     `gorm:"column:file_size;not null" json:"file_size"` // in bytes
	ContentType  string    `gorm:"column:content_type;type:varchar(100);not null" json:"content_type"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SupportTicketAttachment) TableName() string {
	return "support_ticket_attachments"
}

// Input Structs
type CreateTicketAttachmentInput struct {
	FileURL     string `json:"file_url" validate:"required,url"`
	FileName    string `json:"file_name" validate:"required"`
	FileSize    int64  `json:"file_size" validate:"required,min=1"`
	ContentType string `json:"content_type" validate:"required"`
}

type CreateTicketInput struct {
	Category    string                        `json:"category" validate:"required"`
	Description string                        `json:"description" validate:"required,min=10"`
	BusinessID  *string                       `json:"business_id,omitempty"`
	Attachments []CreateTicketAttachmentInput `json:"attachments,omitempty"`
}

type ResolveTicketInput struct {
	Status                     string `json:"status" validate:"required,oneof=in_progress solved closed"`
	CustomerServiceSuggestions string `json:"customer_service_suggestions" validate:"required"`
}

// Response Detail Struct with fallback suggestions
type SupportTicketDetailResponse struct {
	TicketID                   string                    `json:"ticket_id"`
	Category                   string                    `json:"category"`
	Description                string                    `json:"description"`
	SupportTicketStatus        string                    `json:"support_ticket_status"`
	CustomerServiceSuggestions string                    `json:"customer_service_suggestions"`
	BusinessID                 *string                   `json:"business_id,omitempty"`
	CreatedAt                  time.Time                 `json:"created_at"`
	ResolvedAt                 *time.Time                `json:"resolved_at,omitempty"`
	Attachments                []SupportTicketAttachment `json:"attachments"`
}

// FormatDetailResponse mengonversi SupportTicket model ke response dengan fallback saran CS
func (t *SupportTicket) FormatDetailResponse() SupportTicketDetailResponse {
	suggestions := "Tiket masih direview"
	if t.CustomerServiceSuggestions != nil && *t.CustomerServiceSuggestions != "" {
		suggestions = *t.CustomerServiceSuggestions
	}

	attachments := t.Attachments
	if attachments == nil {
		attachments = []SupportTicketAttachment{}
	}

	return SupportTicketDetailResponse{
		TicketID:                   t.TicketID,
		Category:                   t.Category,
		Description:                t.Description,
		SupportTicketStatus:        t.SupportTicketStatus,
		CustomerServiceSuggestions: suggestions,
		BusinessID:                 t.BusinessID,
		CreatedAt:                  t.CreatedAt,
		ResolvedAt:                 t.ResolvedAt,
		Attachments:                attachments,
	}
}
