package ticket

import (
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type SupportTicketRepository interface {
	CreateTicket(ticket *models.SupportTicket, attachments []models.SupportTicketAttachment) error
	GetTicketsByAccountID(accountID string, limit int, cursor string) ([]models.SupportTicket, error)
	GetTicketByID(ticketID string) (*models.SupportTicket, error)
	GetAdminTickets(status string, limit int, cursor string) ([]models.SupportTicket, error)
	ResolveTicket(ticketID string, status string, suggestions string, resolvedBy string) error
	GetAccountByID(accountID string) (*models.Account, error)
}

type supportTicketRepository struct {
	db *gorm.DB
}

func NewSupportTicketRepository(db *gorm.DB) SupportTicketRepository {
	return &supportTicketRepository{db: db}
}

func (r *supportTicketRepository) CreateTicket(ticket *models.SupportTicket, attachments []models.SupportTicketAttachment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			if err := tx.Create(&attachments).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *supportTicketRepository) GetTicketsByAccountID(accountID string, limit int, cursor string) ([]models.SupportTicket, error) {
	var tickets []models.SupportTicket
	query := r.db.
		Preload("Attachments").
		Where("account_id = ?", accountID)

	if cursor != "" {
		var cursorTicket models.SupportTicket
		if err := r.db.Select("created_at").Where("ticket_id = ?", cursor).First(&cursorTicket).Error; err == nil {
			query = query.Where("created_at < ?", cursorTicket.CreatedAt)
		}
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&tickets).Error
	return tickets, err
}

func (r *supportTicketRepository) GetTicketByID(ticketID string) (*models.SupportTicket, error) {
	var ticket models.SupportTicket
	err := r.db.
		Preload("Account").
		Preload("Business").
		Preload("Resolver").
		Preload("Attachments").
		Where("ticket_id = ?", ticketID).
		First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *supportTicketRepository) GetAdminTickets(status string, limit int, cursor string) ([]models.SupportTicket, error) {
	var tickets []models.SupportTicket
	query := r.db.
		Preload("Account").
		Preload("Business").
		Preload("Attachments")

	if status != "" {
		query = query.Where("support_ticket_status = ?", status)
	}

	if cursor != "" {
		var cursorTicket models.SupportTicket
		if err := r.db.Select("created_at").Where("ticket_id = ?", cursor).First(&cursorTicket).Error; err == nil {
			query = query.Where("created_at < ?", cursorTicket.CreatedAt)
		}
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&tickets).Error
	return tickets, err
}

func (r *supportTicketRepository) ResolveTicket(ticketID string, status string, suggestions string, resolvedBy string) error {
	now := time.Now()
	return r.db.Model(&models.SupportTicket{}).
		Where("ticket_id = ?", ticketID).
		Updates(map[string]interface{}{
			"support_ticket_status":         status,
			"customer_service_suggestions": suggestions,
			"resolved_at":                   &now,
			"resolved_by":                   resolvedBy,
			"updated_at":                    now,
		}).Error
}

func (r *supportTicketRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	if err := r.db.Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

