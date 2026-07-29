package models

import "time"

type Account struct {
	AccountID     string     `gorm:"primaryKey" json:"accountId"`
	AuthProvider  string     `gorm:"default:'local'" json:"authProvider"`
	Email         *string    `gorm:"unique;" json:"email"`
	PhoneNumber   *string    `gorm:"unique;" json:"phoneNumber"`
	Fullname      string     `json:"fullname"`
	Username      string     `gorm:"unique;" json:"username"`
	PasswordHash  *string    `json:"-"`
	Role          string     `json:"role"`
	Gender        *string    `json:"gender"`
	AccountStatus string     `gorm:"default:'inactive'" json:"accountStatus"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`

	Customer      *Customer      `json:"customer,omitempty"`
	Owner         *Owner         `json:"owner,omitempty"`
	BusinessAdmin *BusinessAdmin `json:"business_admin,omitempty"`
	PlatformAdmin *PlatformAdmin `json:"platform_admin,omitempty"`
}

type Customer struct {
	AccountID         string    `gorm:"primaryKey" json:"accountId"`
	Bio               *string   `json:"bio"`
	ProfilePictureUrl *string   `json:"profilePictureUrl"`
	BirthDate         *string   `gorm:"type:date" json:"birthDate"`
	LastLatitude      float64   `json:"lastLatitude"`
	LastLongitude     float64   `json:"lastLongitude"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Owner struct {
	AccountID    string    `gorm:"primaryKey" json:"accountId"`
	KtpNumber    *string   `json:"ktpNumber"`
	KtpPhotoUrl  *string   `json:"ktpPhotoUrl"`
	ReferralCode *string   `json:"referralCode" `
	CreatedAt    time.Time `json:"createdAt"`
}

type PlatformAdmin struct {
	AccountID string `gorm:"primaryKey" json:"accountId"`
	Role      string `json:"role"`
}

type RegisterFCMTokenInput struct {
	Token      string `json:"fcm_token"`
	DeviceType string `json:"device_type"` // opsional: 'android', 'ios', atau 'web'
}

type DeviceToken struct {
	TokenID    int64     `gorm:"primaryKey;autoIncrement" json:"token_id"`
	AccountID  string    `gorm:"not null;column:account_id" json:"account_id"`
	FCMToken   string    `gorm:"not null;unique;column:fcm_token" json:"fcm_token"`
	DeviceType string    `gorm:"column:device_type" json:"device_type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
