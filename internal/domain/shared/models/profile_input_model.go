package models

type UpdateProfileCustomerInput struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Username    *string `json:"username"`
	PhoneNumber *string `json:"phone_number"`
	Gender      *string `json:"gender"`

	Bio               *string  `json:"bio"`
	ProfilePictureUrl *string  `json:"profile_picture_url"`
	BirthDate         *string  `json:"birth_date"`
	LastLatitude      *float64 `json:"last_latitude"`
	LastLongitude     *float64 `json:"last_longitude"`
}

type UpdateProfileOwnerInput struct {
	FirstName         *string `json:"first_name"`
	LastName          *string `json:"last_name"`
	Username          *string `json:"username"`
	PhoneNumber       *string `json:"phone_number"`
	Title             *string `json:"title"`
	ProfilePictureUrl *string `json:"profile_picture_url"`
	Country           *string `json:"country"`
	ReferralCode      *string `json:"referral_code"`
}

type VerifyDeleteOTPInput struct {
	OTP string `json:"otp" validate:"required"`
}

type ForceDeleteAccountInput struct {
	Identifier string `json:"identifier" validate:"required"`
}