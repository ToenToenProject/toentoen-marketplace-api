package models

import "time"

type Account struct {
	AccountID     string      `gorm:"primaryKey" json:"accountId"`
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

	User          *User          `gorm:"foreignKey:AccountID;references:AccountID" json:"user,omitempty"`
	Owner         *Owner         `gorm:"foreignKey:AccountID;references:AccountID" json:"owner,omitempty"`
	BusinessAdmin *BusinessAdmin `gorm:"foreignKey:AccountID;references:AccountID" json:"business_admin,omitempty"`
	PlatformAdmin *PlatformAdmin `gorm:"foreignKey:AccountID;references:AccountID" json:"platform_admin,omitempty"`
}

type User struct {
	AccountID         string     `gorm:"primaryKey" json:"accountId"`
	Bio               *string   `json:"bio"`
	ProfilePictureUrl *string   `json:"profile_pictureUrl"`
	BirthDate         *string   `gorm:"type:date" json:"birthDate"`
	LastLatitude      float64   `json:"lastLatitude"`
	LastLongitude     float64   `json:"lastLongitude"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Owner struct {
	AccountID   string     `gorm:"primaryKey" json:"accountId"`
	KtpNumber   *string    `json:"ktpNumber"`
	KtpPhotoUrl *string    `json:"ktpPhotoUrl"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PlatformAdmin struct {
	AccountID string  `gorm:"primaryKey" json:"accountId"`
	Role      string `json:"role"`
}
