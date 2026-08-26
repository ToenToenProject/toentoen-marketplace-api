package models

import (
	"time"
)

// BusinessTransferRequest merepresentasikan entitas permohonan transfer kepemilikan bisnis
type BusinessTransferRequest struct {
	TransferID       string     `gorm:"primaryKey;column:transfer_id;type:varchar(50)" json:"transfer_id"`
	BusinessID       string     `gorm:"column:business_id;type:varchar(50);not null;index" json:"business_id"`
	CurrentOwnerID   string     `gorm:"column:current_owner_id;type:varchar(50);not null;index" json:"current_owner_id"`
	TargetAccountID  string     `gorm:"column:target_account_id;type:varchar(50);not null;index" json:"target_account_id"`
	TargetIdentifier string     `gorm:"column:target_identifier;type:varchar(150);not null" json:"target_identifier"`
	Status           string     `gorm:"column:status;type:varchar(30);not null;default:'pending_acceptance';index" json:"status"` // pending_acceptance, accepted, rejected, cancelled, expired
	OTPVerifiedAt    *time.Time `gorm:"column:otp_verified_at" json:"otp_verified_at,omitempty"`
	AcceptedAt       *time.Time `gorm:"column:accepted_at" json:"accepted_at,omitempty"`
	RejectedAt       *time.Time `gorm:"column:rejected_at" json:"rejected_at,omitempty"`
	CancelledAt      *time.Time `gorm:"column:cancelled_at" json:"cancelled_at,omitempty"`
	ExpiresAt        time.Time  `gorm:"column:expires_at;not null" json:"expires_at"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relasi
	Business     *Business `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	CurrentOwner *Account  `gorm:"foreignKey:CurrentOwnerID;references:AccountID" json:"current_owner,omitempty"`
	TargetAccount *Account `gorm:"foreignKey:TargetAccountID;references:AccountID" json:"target_account,omitempty"`
}

func (BusinessTransferRequest) TableName() string {
	return "business_transfer_requests"
}

// DTO Input & Response

type CheckTransferTargetInput struct {
	TargetIdentifier string `json:"target_identifier" validate:"required"`
}

type TransferTargetPreviewResponse struct {
	TargetAccountID string  `json:"target_account_id"`
	TargetName      string  `json:"target_name"`
	TargetEmail     *string `json:"target_email,omitempty"`
	TargetPhone     *string `json:"target_phone,omitempty"`
	TargetAvatar    *string `json:"target_avatar,omitempty"`
	TargetRole      string  `json:"target_role"`
}

type InitiateTransferInput struct {
	TargetIdentifier string `json:"target_identifier" validate:"required"`
	Channel          string `json:"channel,omitempty"` // "email" atau "whatsapp"
}

type VerifyTransferOTPInput struct {
	TargetIdentifier string `json:"target_identifier" validate:"required"`
	OTP              string `json:"otp" validate:"required,len=6"`
}

type RespondTransferInput struct {
	Action string `json:"action" validate:"required,oneof=accept reject"` // "accept" atau "reject"
}

type IncomingTransferListItem struct {
	TransferID       string    `json:"transfer_id"`
	BusinessID       string    `json:"business_id"`
	BusinessName     string    `json:"business_name"`
	BusinessLogo     *string   `json:"business_logo,omitempty"`
	CurrentOwnerID   string    `json:"current_owner_id"`
	CurrentOwnerName string    `json:"current_owner_name"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}
