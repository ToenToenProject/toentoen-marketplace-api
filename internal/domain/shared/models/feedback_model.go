package models

// CreateFeedbackInput represents user feedback submitted via email.
type CreateFeedbackInput struct {
	Subject string `json:"subject" validate:"omitempty,max=150"`
	Message string `json:"message" validate:"required,min=5,max=3000"`
}
