package ticket

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

const (
	MaxFileSizePerAttachment = 10 * 1024 * 1024 // 10MB
	MaxTotalAttachmentsSize  = 20 * 1024 * 1024 // 20MB
)

type SupportTicketService interface {
	CreateTicket(accountID string, userRole string, input models.CreateTicketInput) (*models.SupportTicketDetailResponse, error)
	GetMyTickets(accountID string, limit int, cursor string) ([]models.SupportTicketDetailResponse, error)
	GetTicketDetail(ticketID string, requesterID string, isCS bool) (*models.SupportTicketDetailResponse, error)
	AdminGetTickets(status string, limit int, cursor string) ([]models.SupportTicketDetailResponse, error)
	AdminResolveTicket(ticketID string, input models.ResolveTicketInput, resolvedBy string) (*models.SupportTicketDetailResponse, error)
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
	for _, att := range input.Attachments {
		attachments = append(attachments, models.SupportTicketAttachment{
			AttachmentID: utils.GenerateUUID("TATT"),
			TicketID:     ticketID,
			FileURL:      att.FileURL,
			FileName:     att.FileName,
			FileSize:     att.FileSize,
			ContentType:  att.ContentType,
			CreatedAt:    now,
		})
	}

	if err := s.repo.CreateTicket(ticket, attachments); err != nil {
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

	res := ticket.FormatDetailResponse()
	return &res, nil
}

func (s *supportTicketService) AdminGetTickets(status string, limit int, cursor string) ([]models.SupportTicketDetailResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	tickets, err := s.repo.GetAdminTickets(status, limit, cursor)
	if err != nil {
		return nil, errors.New("Gagal mengambil daftar tiket admin")
	}

	var responses []models.SupportTicketDetailResponse
	for _, t := range tickets {
		responses = append(responses, t.FormatDetailResponse())
	}
	return responses, nil
}

func (s *supportTicketService) AdminResolveTicket(ticketID string, input models.ResolveTicketInput, resolvedBy string) (*models.SupportTicketDetailResponse, error) {
	ticket, err := s.repo.GetTicketByID(ticketID)
	if err != nil {
		return nil, errors.New("Tiket bantuan tidak ditemukan")
	}

	suggestions := strings.TrimSpace(input.CustomerServiceSuggestions)
	if suggestions == "" {
		return nil, errors.New("Saran/tanggapan customer service wajib diisi")
	}

	if err := s.repo.ResolveTicket(ticketID, input.Status, suggestions, resolvedBy); err != nil {
		return nil, errors.New("Gagal memperbarui status tiket")
	}

	updatedTicket, err := s.repo.GetTicketByID(ticketID)
	if err != nil {
		ticket.SupportTicketStatus = input.Status
		ticket.CustomerServiceSuggestions = &suggestions
		res := ticket.FormatDetailResponse()
		return &res, nil
	}

	res := updatedTicket.FormatDetailResponse()
	return &res, nil
}
