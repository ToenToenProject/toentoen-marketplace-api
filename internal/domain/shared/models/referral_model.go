package models

import "time"

type ReferralUser struct {
	ID           string    `gorm:"primaryKey;column:id" json:"id"`
	WorkerID     string    `gorm:"column:worker_id" json:"worker_id"`
	Name         string    `gorm:"column:name" json:"name"`
	Role         string    `gorm:"column:role" json:"role"`
	ReferralCode *string   `gorm:"column:referral_code" json:"referral_code"`
	IsActive     bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (ReferralUser) TableName() string {
	return "operations.users"
}

// OperationsUser adalah representasi pengguna internal dari skema operations (CTO, Admin, CS, dsb)
type OperationsUser = ReferralUser
