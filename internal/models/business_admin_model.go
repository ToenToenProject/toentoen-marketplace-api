package models

import "time"

type BusinessAdmin struct {
	AccountID int64     `gorm:"primaryKey" json:"account_id"`
	OwnerID   int64     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`

	Account     Account                   `gorm:"foreignKey:AccountID;references:AccountID" json:"account"`
	Assignments []BusinessAdminAssignment `gorm:"foreignKey:BusinessAdminID;references:AccountID" json:"assignments"`
}

type BusinessAdminAssignment struct {
	BusinessAdminID int64     `gorm:"primaryKey;column:admin_id" json:"admin_id"`
	BusinessID      int64     `gorm:"primaryKey" json:"business_id"`
	Role            string    `gorm:"default:'admin';not null" json:"role"`
	ShiftStart      *string   `json:"shift_start"`
	ShiftEnd        *string   `json:"shift_end"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Admin    *BusinessAdmin `gorm:"foreignKey:BusinessAdminID;references:AccountID" json:"admin,omitempty"`
}

type AssignBusinessAdminInput struct {
	BusinessAdminID int64   `json:"business_admin_id"`
	BusinessID      int64   `json:"business_id"`
	Role            string  `json:"role"`
	ShiftStart      *string `json:"shift_start"`
	ShiftEnd        *string `json:"shift_end"`
}

type UpdateAssignmentInput struct {
	Role       *string `json:"role"`
	ShiftStart *string `json:"shift_start"`
	ShiftEnd   *string `json:"shift_end"`
	IsActive   *bool   `json:"is_active"`
}
