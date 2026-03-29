package models

type UpdateProfileUserInput struct {
	Fullname    *string `json:"fullname"`
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
	Fullname    *string `json:"fullname"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Gender      *string `json:"gender"`

	KtpNumber   *string `json:"ktp_number"`
	KtpPhotoUrl *string `json:"ktp_photo_url"`
}