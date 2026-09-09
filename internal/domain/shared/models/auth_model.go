package models

import "time"

type Account struct {
	AccountID     string     `gorm:"primaryKey" json:"accountId"`
	AuthProvider  string     `gorm:"default:'local'" json:"authProvider"`
	Email             *string    `gorm:"unique;" json:"email"`
	PhoneNumber       *string    `gorm:"unique;" json:"phoneNumber"`
	FirstName         string     `json:"firstName"`
	LastName          string     `json:"lastName"`
	Username          string     `gorm:"unique;" json:"username"`
	ProfilePictureUrl *string    `json:"profilePictureUrl"`
	PasswordHash  *string    `json:"-"`
	Role          string     `json:"role"`
	Gender        *string    `json:"gender"`
	AccountStatus     string     `gorm:"default:'inactive'" json:"accountStatus"`
	EmailVerifiedAt   *time.Time `json:"emailVerifiedAt"`
	PhoneVerifiedAt   *time.Time `gorm:"column:phone_verified_at" json:"-"`
	LastLoginAt       *time.Time `json:"lastLoginAt"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`

	Customer      *Customer      `gorm:"foreignKey:AccountID;references:AccountID" json:"customer,omitempty"`
	Owner         *Owner         `gorm:"foreignKey:AccountID;references:AccountID" json:"owner,omitempty"`
	BusinessAdmin *BusinessAdmin `gorm:"foreignKey:AccountID;references:AccountID" json:"business_admin,omitempty"`
}

type Customer struct {
	AccountID         string    `gorm:"primaryKey" json:"accountId"`
	Bio               *string   `json:"bio"`
	BirthDate         *string   `gorm:"type:date" json:"birthDate"`
	LastLatitude      float64   `json:"lastLatitude"`
	LastLongitude     float64   `json:"lastLongitude"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Owner struct {
	AccountID          string     `gorm:"primaryKey;column:account_id" json:"accountId"`
	KtpNumber          *string    `gorm:"type:text;column:ktp_number" json:"ktpNumber"`
	KtpPhotoUrl        *string    `gorm:"type:text;column:ktp_photo_url" json:"ktpPhotoUrl"`
	SelfiePhotoUrl     *string    `gorm:"type:text;column:selfie_photo_url" json:"selfiePhotoUrl"`
	ReferralCode       *string    `gorm:"column:referral_code" json:"referralCode"`
	Country            *string    `gorm:"column:country" json:"country"`
	KtpName            *string    `gorm:"type:text;column:ktp_name" json:"ktpName"`
	KtpBirthPlace      *string    `gorm:"type:text;column:ktp_birth_place" json:"ktpBirthPlace"`
	KtpBirthDate       *string    `gorm:"type:date;column:ktp_birth_date" json:"ktpBirthDate"`
	KtpGender          *string    `gorm:"type:text;column:ktp_gender" json:"ktpGender"`
	KtpAddress         *string    `gorm:"type:text;column:ktp_address" json:"ktpAddress"`
	KtpRtRw            *string    `gorm:"type:text;column:ktp_rt_rw" json:"ktpRtRw"`
	KtpVillage         *string    `gorm:"type:text;column:ktp_village" json:"ktpVillage"`
	KtpDistrict        *string    `gorm:"type:text;column:ktp_district" json:"ktpDistrict"`
	KtpReligion        *string    `gorm:"type:text;column:ktp_religion" json:"ktpReligion"`
	KtpOccupation      *string    `gorm:"type:text;column:ktp_occupation" json:"ktpOccupation"`
	KtpCitizenship     *string    `gorm:"type:text;column:ktp_citizenship" json:"ktpCitizenship"`
	KycStatus          string     `gorm:"default:'unverified';column:kyc_status" json:"kycStatus"`
	KycRejectionReason *string    `gorm:"type:text;column:kyc_rejection_reason" json:"kycRejectionReason"`
	KycSubmittedAt     *time.Time `gorm:"column:kyc_submitted_at" json:"kycSubmittedAt"`
	KycVerifiedAt      *time.Time `gorm:"column:kyc_verified_at" json:"kycVerifiedAt"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"createdAt"`
}

func (Owner) TableName() string {
	return "owners"
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
