package models

type UpdateProfileCustomerInput struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Gender      *string `json:"gender"`

	Bio               *string  `json:"bio"`
	ProfilePictureUrl *string  `json:"profile_picture_url"`
	BirthDate         *string  `json:"birth_date"`
	LastLatitude      *float64 `json:"last_latitude"`
	LastLongitude     *float64 `json:"last_longitude"`
}

type UpdateProfileOwnerInput struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Gender      *string `json:"gender"`
	
	ProfilePictureUrl *string `json:"profile_picture_url"`
	Country           *string `json:"country"`

	KtpNumber      *string `json:"ktp_number"`
	KtpPhotoUrl    *string `json:"ktp_photo_url"`
	KtpName        *string `json:"ktp_name"`
	KtpBirthPlace  *string `json:"ktp_birth_place"`
	KtpBirthDate   *string `json:"ktp_birth_date"`
	KtpGender      *string `json:"ktp_gender"`
	KtpAddress     *string `json:"ktp_address"`
	KtpRtRw        *string `json:"ktp_rt_rw"`
	KtpVillage     *string `json:"ktp_village"`
	KtpDistrict    *string `json:"ktp_district"`
	KtpReligion    *string `json:"ktp_religion"`
	KtpOccupation  *string `json:"ktp_occupation"`
	KtpCitizenship *string `json:"ktp_citizenship"`

	ReferralCode *string `json:"referral_code"`
}

type VerifyDeleteOTPInput struct {
	OTP string `json:"otp" validate:"required"`
}

type ForceDeleteAccountInput struct {
	Identifier string `json:"identifier" validate:"required"`
}