package models

type RegisterUnifiedInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	Role 		 string `json:"role"`
	FromRole *string `json:"from_role"`
}

type LoginUnifiedInput struct {
	Identifier string `json:"identifier"`
	Password	 string `json:"password"`
}

type VerifyLoginOTPInput struct {
	Identifier string `json:"identifier"`
	OTP string `json:"otp"`
}

type GoogleAuthInput struct {
	GoogleID string `json:"google_id"`
	Email string `json:"email"`
	Fullname string `json:"fullname"`
}

type ForgotPasswordInput struct {
	Identifier string `json:"identifier"`
}

type VerifyForgotOTPInput struct {
	Identifier string `json:"identifier"`
	OTP string `json:"otp"`
}

type ResetPasswordInput struct {
	Identifier string `json:"identifier"`
	ResetToken string `json:"reset_token"`
	NewPassword string `json:"new_password" binding:"min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" `
}

type ChangePasswordInput struct {
	Email string `json:"email"`
	CurrentPassword string `json:"current_password"`
	NewPassword string `json:"new_password" binding:"min=8"`
}

type CreateBusinessAdminInput struct {
	Identifier string `json:"identifier"`
	Password string `json:"password"`
}