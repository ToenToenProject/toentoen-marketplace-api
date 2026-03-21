package models

import "time"

type Account struct {
	AccountID     int64      `gorm:"primaryKey" json:"account_id"`
	AuthProvider  string     `gorm:"default:'local'" json:"auth_provider"`
	Email         *string    `gorm:"unique;" json:"email"`
	PhoneNumber   *string    `gorm:"unique;" json:"phone_number"`
	Fullname      string     `json:"fullname"`
	Username      string     `gorm:"unique;" json:"username"`
	PasswordHash  *string    `json:"-"`
	Role          string     `json:"role"`
	Gender        *string    `json:"gender"`
	AccountStatus string     `gorm:"default:'inactive'" json:"account_status"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	User          *User          `gorm:"foreignKey:AccountID;references:AccountID" json:"user,omitempty"`
	Owner         *Owner         `gorm:"foreignKey:AccountID;references:AccountID" json:"owner,omitempty"`
	BusinessAdmin *BusinessAdmin `gorm:"foreignKey:AccountID;references:AccountID" json:"business_admin,omitempty"`
	PlatformAdmin *PlatformAdmin `gorm:"foreignKey:AccountID;references:AccountID" json:"platform_admin,omitempty"`
}

type User struct {
	AccountID         int64     `gorm:"primaryKey" json:"account_id"`
	Bio               *string   `json:"bio"`
	ProfilePictureUrl *string   `json:"profile_picture_url"`
	BirthDate         *string   `gorm:"type:date" json:"birth_date"`
	LastLatitude      float64   `json:"last_latitude"`
	LastLongitude     float64   `json:"last_longitude"`
	CreatedAt         time.Time `json:"created_at"`
}

type Owner struct {
	AccountID   int64     `gorm:"primaryKey" json:"account_id"`
	KtpNumber   *string    `json:"ktp_number"`
	KtpPhotoUrl *string    `json:"ktp_photo_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type PlatformAdmin struct {
	AccountID int64  `gorm:"primaryKey" json:"account_id"`
	Role      string `json:"role"`
}
