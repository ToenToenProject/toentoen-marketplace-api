package models

type RegisterUnifiedInput struct {
	Identifier string  `json:"identifier"`
	Password   string  `json:"password"`
	Role       string  `json:"role"`
	FromRole   *string `json:"from_role"`
}

type VerifyRegisterOTPInput struct {
	Identifier string `json:"identifier" validate:"required,email"`
	OTP        string `json:"otp" validate:"required,len=6"`
}

type LoginUnifiedInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type VerifyLoginOTPInput struct {
	Identifier string `json:"identifier"`
	OTP        string `json:"otp"`
}

type GoogleAuthInput struct {
	GoogleID string `json:"google_id"`
	Email    string `json:"email"`
	Fullname string `json:"fullname"`
}

type ForgotPasswordInput struct {
	Identifier string `json:"identifier"`
}

type VerifyForgotOTPInput struct {
	Identifier string `json:"identifier"`
	OTP        string `json:"otp"`
}

type ResetPasswordInput struct {
	Identifier         string `json:"identifier"`
	ResetToken         string `json:"reset_token"`
	NewPassword        string `json:"new_password" binding:"min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" `
}

type RequestEmailChangeInput struct {
	CurrentEmail string `json:"current_email"`
	NewEmail     string `json:"new_email"`
}

type VerifyEmailChangeInput struct {
	NewEmail string `json:"new_email"`
	OTP      string `json:"otp"`
}

type ChangePasswordInput struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password"`
	ConfirmNewPassword string `json:"confirm_new_password"`
}

type CreateBusinessAdminInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}
