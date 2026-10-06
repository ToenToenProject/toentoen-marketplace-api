package models

import (
	"strings"
	"time"
)

// ==========================================
// ROLE GROUP (Pemisahan identitas akun)
// ==========================================
// Satu email / nomor telepon boleh dipakai maksimal 1x per kelompok role:
//   - RoleGroupCustomer : role "customer"
//   - RoleGroupMerchant : role "owner" dan "business_admin" (tidak boleh berbagi email)
// Setiap akun tetap memiliki account_id sendiri sehingga seluruh relasi data
// (review, signal, bisnis, laporan, dll.) tidak pernah tercampur antar kelompok.
const (
	RoleGroupCustomer = "customer"
	RoleGroupMerchant = "merchant"
)

// RoleGroupOf mengembalikan kelompok role. String kosong jika role tidak dikenal/kosong.
func RoleGroupOf(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "customer":
		return RoleGroupCustomer
	case "owner", "business_admin":
		return RoleGroupMerchant
	default:
		return ""
	}
}

// RolesInGroup mengembalikan daftar role yang termasuk dalam kelompok tertentu.
func RolesInGroup(group string) []string {
	switch group {
	case RoleGroupCustomer:
		return []string{"customer"}
	case RoleGroupMerchant:
		return []string{"owner", "business_admin"}
	default:
		return nil
	}
}

type Account struct {
	AccountID     string     `gorm:"primaryKey" json:"accountId"`
	AuthProvider  string     `gorm:"default:'local'" json:"authProvider"`
	// Unik per kelompok role (lihat scripts/migrations/20261006_accounts_identity_per_role_group.sql)
	Email             *string    `json:"email"`
	PhoneNumber       *string    `json:"phoneNumber"`
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
	AcceptedTermsAt   *time.Time `gorm:"column:accepted_terms_at" json:"acceptedTermsAt,omitempty"`
	TermsVersion      *string    `gorm:"column:terms_version" json:"termsVersion,omitempty"`
	AcceptedPrivacyAt *time.Time `gorm:"column:accepted_privacy_at" json:"acceptedPrivacyAt,omitempty"`
	PrivacyVersion    *string    `gorm:"column:privacy_version" json:"privacyVersion,omitempty"`
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
