package feedback

import (
	"errors"
	"fmt"
	"strings"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type FeedbackService interface {
	SendFeedback(accountID, userRole string, input models.CreateFeedbackInput) error
}

type feedbackService struct {
	repo FeedbackRepository
}

func NewFeedbackService(repo FeedbackRepository) FeedbackService {
	return &feedbackService{repo: repo}
}

func (s *feedbackService) SendFeedback(accountID, userRole string, input models.CreateFeedbackInput) error {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil || account == nil {
		return errors.New("data akun pengguna tidak ditemukan")
	}

	userEmail := ""
	if account.Email != nil {
		userEmail = *account.Email
	}
	if userEmail == "" {
		return errors.New("akun Anda belum memiliki alamat email yang terdaftar")
	}

	userName := strings.TrimSpace(fmt.Sprintf("%s %s", account.FirstName, account.LastName))
	if userName == "" {
		userName = account.Username
	}
	if userName == "" {
		userName = "Pengguna ToenToen"
	}

	subject := strings.TrimSpace(input.Subject)
	if subject == "" {
		subject = "Feedback Pengguna ToenToen"
	}

	message := strings.TrimSpace(input.Message)
	if message == "" {
		return errors.New("pesan feedback tidak boleh kosong")
	}

	return utils.SendFeedbackEmail(userEmail, userName, userRole, subject, message)
}
