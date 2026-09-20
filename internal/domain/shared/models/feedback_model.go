package models

// CreateFeedbackInput payload untuk mengirimkan pesan feedback pengguna via email
type CreateFeedbackInput struct {
	Subject string `json:"subject" validate:"omitempty,max=150"`
	Message string `json:"message" validate:"required,min=5,max=3000"`
}
