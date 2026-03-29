package models

import "time"

type BusinessAdmin struct {
	AccountID string     `gorm:"primaryKey" json:"account_id"`
	OwnerID   string     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`

	Account     Account                   `gorm:"foreignKey:AccountID;references:AccountID" json:"account"`
	Assignments []BusinessAdminAssignment `gorm:"foreignKey:BusinessAdminID;references:AccountID" json:"assignments"`
}

type BusinessAdminAssignment struct {
	BusinessAdminID string     `gorm:"primaryKey;column:admin_id" json:"admin_id"`
	BusinessID      string     `gorm:"primaryKey" json:"business_id"`
	Role            string    `gorm:"default:'admin';not null" json:"role"`
	ShiftStart      *string   `json:"shift_start"`
	ShiftEnd        *string   `json:"shift_end"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`

	Business *Business `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Admin    *BusinessAdmin `gorm:"foreignKey:BusinessAdminID;references:AccountID" json:"admin,omitempty"`
}

type AssignBusinessAdminInput struct {
	BusinessAdminID string   `json:"business_admin_id"`
	BusinessID      string   `json:"business_id"`
	Role            string  `json:"role"`
	ShiftStart      *string `json:"shift_start"`
	ShiftEnd        *string `json:"shift_end"`
}

type UpdateAssignmentInput struct {
	Fullname    *string `json:"fullname"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Gender      *string `json:"gender"`
	Password    *string `json:"password"`

	Role       *string `json:"role"`
	ShiftStart *string `json:"shift_start"`
	ShiftEnd   *string `json:"shift_end"`
	IsActive   *bool   `json:"is_active"`
}
