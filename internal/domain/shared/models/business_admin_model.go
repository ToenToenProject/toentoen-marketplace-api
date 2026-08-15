package models

import "time"

type BusinessAdmin struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	Account     Account                   `json:"account"`
	Assignments []BusinessAdminAssignment `gorm:"foreignKey:AdminID;references:AccountID" json:"assignments"`
}

type BusinessAdminAssignment struct {
	AssignmentID string `gorm:"primaryKey;column:assignment_id" json:"assignment_id"`
	
	AdminID    string `gorm:"column:admin_id;uniqueIndex:idx_admin_business" json:"admin_id"`
	BusinessID string `gorm:"column:business_id;uniqueIndex:idx_admin_business" json:"business_id"`
	InvitedBy  string `gorm:"column:invited_by" json:"invited_by"`

	Role       string  `gorm:"default:'admin';not null;column:role" json:"role"`
	
	Status     string  `gorm:"default:'pending';column:status" json:"status"`
	
	ShiftStart *string `gorm:"column:shift_start" json:"shift_start"`
	ShiftEnd   *string `gorm:"column:shift_end" json:"shift_end"`
	
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`

	// Relasi
	Business *Business      `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Admin    *BusinessAdmin `gorm:"foreignKey:AdminID;references:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"admin,omitempty"`
}

type InviteAdminInput struct {
	Email      string  `json:"email" validate:"required,email"`
	BusinessID string  `json:"business_id" validate:"required"`
	Role       string  `json:"role"`
	ShiftStart *string `json:"shift_start"`
	ShiftEnd   *string `json:"shift_end"`
}

type RespondInvitationInput struct {
	Status string `json:"status" validate:"required"`
}

type UpdateAssignmentInput struct {
	Role       *string `json:"role"`
	ShiftStart *string `json:"shift_start"`
	ShiftEnd   *string `json:"shift_end"`
	Status     *string `json:"status"`
}
