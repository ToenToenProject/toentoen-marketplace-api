package models

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
	CurrentEmail string `json:"current_email" validate:"required,email"`
	NewEmail     string `json:"new_email" validate:"required,email"`
}

type VerifyEmailChangeInput struct {
	NewEmail string `json:"new_email" validate:"required,email"`
	OTP      string `json:"otp" validate:"required"`
}

type RequestPhoneChangeInput struct {
	CurrentPhone string `json:"current_phone" validate:"required"`
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

