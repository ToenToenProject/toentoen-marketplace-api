package business_chat

import (
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type BusinessChatRepository interface {
	FindConversation(businessID, customerID string) (*models.DirectBusinessConversation, error)
	GetConversationByID(conversationID string) (*models.DirectBusinessConversation, error)
	CreateConversation(conv *models.DirectBusinessConversation, firstMsg *models.DirectBusinessMessage) error
	GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error)
	GetMerchantConversations(businessID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error)
	SaveMessage(message *models.DirectBusinessMessage) error
	UpdateConversationLastMessage(conversationID, preview string, lastMessageAt time.Time) error
	GetMessagesByConversationID(conversationID string, limit int, cursor string) ([]models.DirectBusinessMessage, error)
	MarkMessagesAsRead(conversationID string, readerType string) error
	VerifyMerchantAccess(businessID string, accountID string) (bool, error)
	GetMerchantStaffAccountIDs(businessID string) ([]string, error)
	GetAccountByID(accountID string) (*models.Account, error)
	GetBusinessByID(businessID string) (*models.Business, error)
	GetProductByID(productID string) (*models.Product, error)
}

type businessChatRepository struct {
	db *gorm.DB
}

func NewBusinessChatRepository(db *gorm.DB) BusinessChatRepository {
	return &businessChatRepository{db: db}
}

func (r *businessChatRepository) FindConversation(businessID, customerID string) (*models.DirectBusinessConversation, error) {
	var conv models.DirectBusinessConversation
	err := r.db.
		Preload("Business").
		Preload("Customer").
		Where("business_id = ? AND customer_id = ?", businessID, customerID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *businessChatRepository) GetConversationByID(conversationID string) (*models.DirectBusinessConversation, error) {
	var conv models.DirectBusinessConversation
	err := r.db.
		Preload("Business").
		Preload("Customer").
		Where("conversation_id = ?", conversationID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *businessChatRepository) CreateConversation(conv *models.DirectBusinessConversation, firstMsg *models.DirectBusinessMessage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conv).Error; err != nil {
			return err
		}
		if firstMsg != nil {
			if err := tx.Create(firstMsg).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *businessChatRepository) GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error) {
	var convs []models.DirectBusinessConversation
	query := r.db.
		Preload("Business").
		Where("customer_id = ?", customerID)

	if cursor != "" {
		var cursorConv models.DirectBusinessConversation
		if err := r.db.Select("last_message_at").Where("conversation_id = ?", cursor).First(&cursorConv).Error; err == nil {
			query = query.Where("last_message_at < ?", cursorConv.LastMessageAt)
		}
	}

	err := query.Order("last_message_at DESC").Limit(limit).Find(&convs).Error
	if err != nil {
		return nil, err
	}

	var results []models.CustomerBusinessConversationListItem
	for _, c := range convs {
		var unread int64
		r.db.Model(&models.DirectBusinessMessage{}).
			Where("conversation_id = ? AND sender_type = 'merchant' AND is_read = false", c.ConversationID).
			Count(&unread)

		var bizName string
		var bizLogo *string
		if c.Business != nil {
			if c.Business.BusinessName != nil {
				bizName = *c.Business.BusinessName
			}
			bizLogo = c.Business.LogoURL
		}

		results = append(results, models.CustomerBusinessConversationListItem{
			ConversationID:     c.ConversationID,
			BusinessID:         c.BusinessID,
			BusinessName:       bizName,
			BusinessLogo:       bizLogo,
			LastMessagePreview: c.LastMessagePreview,
			LastMessageAt:      c.LastMessageAt,
			UnreadCount:        unread,
		})
	}

	return results, nil
}

func (r *businessChatRepository) GetMerchantConversations(businessID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error) {
	var convs []models.DirectBusinessConversation
	query := r.db.
		Preload("Customer").
		Where("business_id = ?", businessID)

	if cursor != "" {
		var cursorConv models.DirectBusinessConversation
		if err := r.db.Select("last_message_at").Where("conversation_id = ?", cursor).First(&cursorConv).Error; err == nil {
			query = query.Where("last_message_at < ?", cursorConv.LastMessageAt)
		}
	}

	err := query.Order("last_message_at DESC").Limit(limit).Find(&convs).Error
	if err != nil {
		return nil, err
	}

	var results []models.MerchantBusinessConversationListItem
	for _, c := range convs {
		var unread int64
		r.db.Model(&models.DirectBusinessMessage{}).
			Where("conversation_id = ? AND sender_type = 'customer' AND is_read = false", c.ConversationID).
			Count(&unread)

		var custName string
		var custAvatar *string
		if c.Customer != nil {
			if c.Customer.FirstName != "" {
				custName = c.Customer.FirstName
				if c.Customer.LastName != "" {
					custName += " " + c.Customer.LastName
				}
			} else {
				custName = c.Customer.Username
			}
			custAvatar = c.Customer.ProfilePictureUrl
		}

		results = append(results, models.MerchantBusinessConversationListItem{
			ConversationID:     c.ConversationID,
			BusinessID:         c.BusinessID,
			CustomerID:         c.CustomerID,
			CustomerName:       custName,
			CustomerAvatar:     custAvatar,
			LastMessagePreview: c.LastMessagePreview,
			LastMessageAt:      c.LastMessageAt,
			UnreadCount:        unread,
		})
	}

	return results, nil
}

func (r *businessChatRepository) SaveMessage(message *models.DirectBusinessMessage) error {
	return r.db.Create(message).Error
}

func (r *businessChatRepository) UpdateConversationLastMessage(conversationID, preview string, lastMessageAt time.Time) error {
	return r.db.Model(&models.DirectBusinessConversation{}).
		Where("conversation_id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_preview": preview,
			"last_message_at":      lastMessageAt,
			"updated_at":           time.Now(),
		}).Error
}

func (r *businessChatRepository) GetMessagesByConversationID(conversationID string, limit int, cursor string) ([]models.DirectBusinessMessage, error) {
	var messages []models.DirectBusinessMessage

	if cursor != "" {
		var cursorMsg models.DirectBusinessMessage
		if err := r.db.Select("created_at").Where("message_id = ?", cursor).First(&cursorMsg).Error; err == nil {
			// Mengambil N pesan tepat sebelum kursor (scroll ke atas)
			var reverseMessages []models.DirectBusinessMessage
			err := r.db.Preload("ProductDetail").
				Where("conversation_id = ? AND created_at < ?", conversationID, cursorMsg.CreatedAt).
				Order("created_at DESC").
				Limit(limit).
				Find(&reverseMessages).Error
			if err != nil {
				return nil, err
			}
			for i := len(reverseMessages) - 1; i >= 0; i-- {
				messages = append(messages, reverseMessages[i])
			}
			return messages, nil
		}
	}

	// KETIKA CURSOR KOSONG (Initial Load saat buka chat room):
	// Ambil N pesan TERAKHIR (terbaru) dengan ORDER BY created_at DESC LIMIT limit, lalu dibalik ke ASC!
	var reverseMessages []models.DirectBusinessMessage
	err := r.db.Preload("ProductDetail").
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Find(&reverseMessages).Error
	if err != nil {
		return nil, err
	}
	for i := len(reverseMessages) - 1; i >= 0; i-- {
		messages = append(messages, reverseMessages[i])
	}
	return messages, nil
}

func (r *businessChatRepository) MarkMessagesAsRead(conversationID string, readerType string) error {
	targetSenderType := "merchant"
	if readerType == "merchant" {
		targetSenderType = "customer"
	}

	now := time.Now()
	return r.db.Model(&models.DirectBusinessMessage{}).
		Where("conversation_id = ? AND sender_type = ? AND is_read = false", conversationID, targetSenderType).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		}).Error
}

func (r *businessChatRepository) VerifyMerchantAccess(businessID string, accountID string) (bool, error) {
	var count int64
	// 1. Cek apakah owner
	if err := r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// 2. Cek apakah assigned business admin yang aktif
	if err := r.db.Model(&models.BusinessAdminAssignment{}).
		Where("business_id = ? AND admin_id = ? AND status IN ('accepted', 'active')", businessID, accountID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *businessChatRepository) GetMerchantStaffAccountIDs(businessID string) ([]string, error) {
	var ids []string

	// 1. Owner account ID
	var business models.Business
	if err := r.db.Select("account_id").Where("business_id = ?", businessID).First(&business).Error; err == nil && business.AccountID != "" {
		ids = append(ids, business.AccountID)
	}

	// 2. Active assigned business admins
	var assignments []models.BusinessAdminAssignment
	if err := r.db.Select("admin_id").
		Where("business_id = ? AND status IN ('accepted', 'active')", businessID).
		Find(&assignments).Error; err == nil {
		for _, a := range assignments {
			ids = append(ids, a.AdminID)
		}
	}

	return ids, nil
}

func (r *businessChatRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *businessChatRepository) GetBusinessByID(businessID string) (*models.Business, error) {
	var business models.Business
	err := r.db.Where("business_id = ?", businessID).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *businessChatRepository) GetProductByID(productID string) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("product_id = ?", productID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}
