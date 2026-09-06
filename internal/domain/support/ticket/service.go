package ticket

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/utils"
)

const (
	MaxFileSizePerAttachment = 10 * 1024 * 1024 // 10MB
	MaxTotalAttachmentsSize  = 20 * 1024 * 1024 // 20MB
)

func decryptTicket(t *models.SupportTicket) {
	if t == nil {
		return
	}
	t.Description, _ = crypto.Decrypt(t.Description)
	t.CustomerServiceSuggestions, _ = crypto.DecryptPointer(t.CustomerServiceSuggestions)
	for i := range t.Attachments {
		t.Attachments[i].FileURL, _ = crypto.Decrypt(t.Attachments[i].FileURL)
	}
}

type SupportTicketService interface {
	CreateTicket(accountID string, userRole string, input models.CreateTicketInput) (*models.SupportTicketDetailResponse, error)
	GetMyTickets(accountID string, limit int, cursor string) ([]models.SupportTicketDetailResponse, error)
	GetTicketDetail(ticketID string, requesterID string, isCS bool) (*models.SupportTicketDetailResponse, error)
}

type supportTicketService struct {
	repo SupportTicketRepository
}

func NewSupportTicketService(repo SupportTicketRepository) SupportTicketService {
	return &supportTicketService{repo: repo}
}

func (s *supportTicketService) CreateTicket(accountID string, userRole string, input models.CreateTicketInput) (*models.SupportTicketDetailResponse, error) {
	category := strings.TrimSpace(input.Category)
	if category == "" {
		return nil, errors.New("Kategori tiket wajib diisi")
	}

	description := strings.TrimSpace(input.Description)
	if len(description) < 10 {
		return nil, errors.New("Deskripsi masalah minimal 10 karakter")
	}

	// Validasi batas ukuran file lampiran
	var totalSize int64 = 0
	for _, att := range input.Attachments {
		if att.FileSize > MaxFileSizePerAttachment {
			return nil, fmt.Errorf("Ukuran file '%s' (%d MB) melebihi batas maksimal 10MB per file", att.FileName, att.FileSize/(1024*1024))
		}
		totalSize += att.FileSize
	}

	if totalSize > MaxTotalAttachmentsSize {
		return nil, errors.New("Total ukuran seluruh file lampiran melebihi batas maksimal 20MB")
	}

	ticketID := utils.GenerateUUID("TCK")
	now := time.Now()

	var businessID *string
	if input.BusinessID != nil && strings.TrimSpace(*input.BusinessID) != "" {
		bID := strings.TrimSpace(*input.BusinessID)
		businessID = &bID
	}

	ticket := &models.SupportTicket{
		TicketID:                   ticketID,
		AccountID:                  accountID,
		UserRole:                   userRole,
		BusinessID:                 businessID,
		Category:                   category,
		Description:                description,
		SupportTicketStatus:        "pending",
		CustomerServiceSuggestions: nil, // Nil agar fallback aktif
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	var attachments []models.SupportTicketAttachment
	var attachmentsToSave []models.SupportTicketAttachment
	for _, att := range input.Attachments {
		attachment := models.SupportTicketAttachment{
			AttachmentID: utils.GenerateUUID("TATT"),
			TicketID:     ticketID,
			FileURL:      att.FileURL,
			FileName:     att.FileName,
			FileSize:     att.FileSize,
			ContentType:  att.ContentType,
			CreatedAt:    now,
		}
		attachments = append(attachments, attachment)

		attSave := attachment
		attSave.FileURL, _ = crypto.Encrypt(attachment.FileURL)
		attachmentsToSave = append(attachmentsToSave, attSave)
	}

	ticketToSave := *ticket
	ticketToSave.Description, _ = crypto.Encrypt(ticket.Description)

	if err := s.repo.CreateTicket(&ticketToSave, attachmentsToSave); err != nil {
		return nil, errors.New("Gagal menyimpan tiket bantuan")
	}

	ticket.Attachments = attachments
	res := ticket.FormatDetailResponse()
	return &res, nil
}

func (s *supportTicketService) GetMyTickets(accountID string, limit int, cursor string) ([]models.SupportTicketDetailResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	tickets, err := s.repo.GetTicketsByAccountID(accountID, limit, cursor)
	if err != nil {
		return nil, errors.New("Gagal mengambil daftar tiket")
	}

	var responses []models.SupportTicketDetailResponse
	for _, t := range tickets {
		decryptTicket(&t)
		responses = append(responses, t.FormatDetailResponse())
	}
	return responses, nil
}

func (s *supportTicketService) GetTicketDetail(ticketID string, requesterID string, isCS bool) (*models.SupportTicketDetailResponse, error) {
	ticket, err := s.repo.GetTicketByID(ticketID)
	if err != nil {
		return nil, errors.New("Tiket bantuan tidak ditemukan")
	}

	if !isCS && ticket.AccountID != requesterID {
		return nil, errors.New("Anda tidak memiliki akses ke tiket bantuan ini")
	}

	decryptTicket(ticket)
	res := ticket.FormatDetailResponse()
	return &res, nil
}
