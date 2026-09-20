package models

import "time"

type RegisterUnifiedInput struct {
	Identifier   string  `json:"identifier" validate:"required"`
	Password     string  `json:"password" validate:"required,min=6"`
	Role         string  `json:"role" validate:"required"`
	FromRole     *string `json:"from_role"`
	ReferralCode *string `json:"referral_code"`
}

type VerifyRegisterOTPInput struct {
	Identifier string `json:"identifier" validate:"required"`
	OTP        string `json:"otp" validate:"required,len=6"`
}

type PendingRegistration struct {
	Identifier   string  `json:"identifier"`
	PasswordHash string  `json:"password_hash"`
	Role         string  `json:"role"`
	ReferralCode *string `json:"referral_code,omitempty"`
	OTP          string  `json:"otp"`
}

type ConsentInput struct {
	AcceptedTerms        *bool   `json:"accepted_terms"`
	AcceptedTermsCamel   *bool   `json:"acceptedTerms"`
	TermsVersion         *string `json:"terms_version"`
	TermsVersionCamel    *string `json:"termsVersion"`
	AcceptedPrivacy      *bool   `json:"accepted_privacy"`
	AcceptedPrivacyCamel *bool   `json:"acceptedPrivacy"`
	PrivacyVersion       *string `json:"privacy_version"`
	PrivacyVersionCamel  *string `json:"privacyVersion"`
}

func (c *ConsentInput) Normalize() {
	if c.AcceptedTerms == nil && c.AcceptedTermsCamel != nil {
		c.AcceptedTerms = c.AcceptedTermsCamel
	}
	if c.TermsVersion == nil && c.TermsVersionCamel != nil {
		c.TermsVersion = c.TermsVersionCamel
	}
	if c.AcceptedPrivacy == nil && c.AcceptedPrivacyCamel != nil {
		c.AcceptedPrivacy = c.AcceptedPrivacyCamel
	}
	if c.PrivacyVersion == nil && c.PrivacyVersionCamel != nil {
		c.PrivacyVersion = c.PrivacyVersionCamel
	}
}

type LoginUnifiedInput struct {
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required"`
}

type VerifyLoginOTPInput struct {
	Identifier string `json:"identifier" validate:"required"`
	OTP        string `json:"otp" validate:"required"`
}

type ForgotPasswordInput struct {
	Identifier string `json:"identifier" validate:"required"`
}

type VerifyForgotOTPInput struct {
	Identifier string `json:"identifier" validate:"required"`
	OTP        string `json:"otp" validate:"required"`
}

type ResetPasswordInput struct {
	Identifier         string `json:"identifier" validate:"required"`
	ResetToken         string `json:"reset_token" validate:"required"`
	NewPassword        string `json:"new_password" validate:"required,min=6"`
	ConfirmNewPassword string `json:"confirm_new_password" validate:"required,eqfield=NewPassword"`
}

type RequestEmailChangeInput struct {
	CurrentEmail string `json:"current_email" validate:"omitempty,email"`
	NewEmail     string `json:"new_email" validate:"required,email"`
}

type VerifyEmailChangeInput struct {
	NewEmail string `json:"new_email" validate:"required,email"`
	OTP      string `json:"otp" validate:"required"`
}

type RequestPhoneChangeInput struct {
	CurrentPhone string `json:"current_phone" validate:"omitempty"`
	NewPhone     string `json:"new_phone" validate:"required"`
}

type VerifyPhoneChangeInput struct {
	NewPhone string `json:"new_phone" validate:"required"`
	OTP      string `json:"otp" validate:"required"`
}

type ChangePasswordInput struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password" validate:"required,min=6"`
	ConfirmNewPassword string `json:"confirm_new_password" validate:"required,eqfield=NewPassword"`
}

type CreateBusinessAdminInput struct {
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required,min=6"`
}

type CheckReferralInput struct {
	Code string `json:"code" query:"code" validate:"required"`
}

type CheckReferralResponse struct {
	Code    string `json:"code"`
	IsValid bool   `json:"is_valid"`
	Message string `json:"message"`
}

type SendEmailOTPInput struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyEmailOTPInput struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type SendPhoneOTPInput struct {
	Phone string `json:"phone" validate:"required"`
}

type VerifyPhoneOTPInput struct {
	Phone string `json:"phone" validate:"required"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type KYCStatusResponse struct {
	Role               string     `json:"role"`
	KycStatus          string     `json:"kycStatus"`
	RejectionReason    *string    `json:"rejectionReason,omitempty"`
	SubmittedAt        *time.Time `json:"submittedAt,omitempty"`
	VerifiedAt         *time.Time `json:"verifiedAt,omitempty"`
	IsVerified         bool       `json:"isVerified"`
	AcceptedTermsAt    *time.Time `json:"acceptedTermsAt,omitempty"`
	TermsVersion       *string    `json:"termsVersion,omitempty"`
	AcceptedPrivacyAt  *time.Time `json:"acceptedPrivacyAt,omitempty"`
	PrivacyVersion     *string    `json:"privacyVersion,omitempty"`
	HasAcceptedTerms   bool       `json:"hasAcceptedTerms"`
	HasAcceptedPrivacy bool       `json:"hasAcceptedPrivacy"`
}

type SubmitKYCInput struct {
	KtpNumber            string  `json:"ktp_number"`
	KtpNumberCamel       string  `json:"ktpNumber"`
	KtpPhotoUrl          string  `json:"ktp_photo_url"`
	KtpPhotoUrlCamel     string  `json:"ktpPhotoUrl"`
	SelfiePhotoUrl       string  `json:"selfie_photo_url"`
	SelfiePhotoUrlCamel  string  `json:"selfiePhotoUrl"`
	KtpName              *string `json:"ktp_name"`
	KtpNameCamel         *string `json:"ktpName"`
	KtpBirthPlace        *string `json:"ktp_birth_place"`
	KtpBirthPlaceCamel   *string `json:"ktpBirthPlace"`
	KtpBirthDate         *string `json:"ktp_birth_date"`
	KtpBirthDateCamel    *string `json:"ktpBirthDate"`
	KtpGender            *string `json:"ktp_gender"`
	KtpGenderCamel       *string `json:"ktpGender"`
	KtpAddress           *string `json:"ktp_address"`
	KtpAddressCamel      *string `json:"ktpAddress"`
	KtpRtRw              *string `json:"ktp_rt_rw"`
	KtpRtRwCamel         *string `json:"ktpRtRw"`
	KtpVillage           *string `json:"ktp_village"`
	KtpVillageCamel      *string `json:"ktpVillage"`
	KtpDistrict          *string `json:"ktp_district"`
	KtpDistrictCamel     *string `json:"ktpDistrict"`
	KtpReligion          *string `json:"ktp_religion"`
	KtpReligionCamel     *string `json:"ktpReligion"`
	KtpOccupation        *string `json:"ktp_occupation"`
	KtpOccupationCamel   *string `json:"ktpOccupation"`
	KtpCitizenship       *string `json:"ktp_citizenship"`
	KtpCitizenshipCamel  *string `json:"ktpCitizenship"`
	AcceptedTerms        *bool   `json:"accepted_terms"`
	AcceptedTermsCamel   *bool   `json:"acceptedTerms"`
	TermsVersion         *string `json:"terms_version"`
	TermsVersionCamel    *string `json:"termsVersion"`
	AcceptedPrivacy      *bool   `json:"accepted_privacy"`
	AcceptedPrivacyCamel *bool   `json:"acceptedPrivacy"`
	PrivacyVersion       *string `json:"privacy_version"`
	PrivacyVersionCamel  *string `json:"privacyVersion"`
}

func (input *SubmitKYCInput) Normalize() {
	if input.KtpNumber == "" {
		input.KtpNumber = input.KtpNumberCamel
	}
	if input.KtpPhotoUrl == "" {
		input.KtpPhotoUrl = input.KtpPhotoUrlCamel
	}
	if input.SelfiePhotoUrl == "" {
		input.SelfiePhotoUrl = input.SelfiePhotoUrlCamel
	}
	if input.KtpName == nil {
		input.KtpName = input.KtpNameCamel
	}
	if input.KtpBirthPlace == nil {
		input.KtpBirthPlace = input.KtpBirthPlaceCamel
	}
	if input.KtpBirthDate == nil {
		input.KtpBirthDate = input.KtpBirthDateCamel
	}
	if input.KtpGender == nil {
		input.KtpGender = input.KtpGenderCamel
	}
	if input.KtpAddress == nil {
		input.KtpAddress = input.KtpAddressCamel
	}
	if input.KtpRtRw == nil {
		input.KtpRtRw = input.KtpRtRwCamel
	}
	if input.KtpVillage == nil {
		input.KtpVillage = input.KtpVillageCamel
	}
	if input.KtpDistrict == nil {
		input.KtpDistrict = input.KtpDistrictCamel
	}
	if input.KtpReligion == nil {
		input.KtpReligion = input.KtpReligionCamel
	}
	if input.KtpOccupation == nil {
		input.KtpOccupation = input.KtpOccupationCamel
	}
	if input.KtpCitizenship == nil {
		input.KtpCitizenship = input.KtpCitizenshipCamel
	}
	if input.AcceptedTerms == nil && input.AcceptedTermsCamel != nil {
		input.AcceptedTerms = input.AcceptedTermsCamel
	}
	if input.TermsVersion == nil && input.TermsVersionCamel != nil {
		input.TermsVersion = input.TermsVersionCamel
	}
	if input.AcceptedPrivacy == nil && input.AcceptedPrivacyCamel != nil {
		input.AcceptedPrivacy = input.AcceptedPrivacyCamel
	}
	if input.PrivacyVersion == nil && input.PrivacyVersionCamel != nil {
		input.PrivacyVersion = input.PrivacyVersionCamel
	}
}

