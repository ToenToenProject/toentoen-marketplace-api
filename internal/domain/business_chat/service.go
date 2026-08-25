package business_chat

import (
	"errors"
	"strings"
	"time"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type BusinessChatService interface {
	InitiateChat(customerID string, businessID string, input models.InitiateBusinessChatInput) (*models.DirectBusinessConversation, *models.DirectBusinessMessage, bool, error)
	GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error)
	GetMerchantConversations(businessID string, requesterID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error)
	GetMessages(conversationID string, requesterID string, limit int, cursor string) ([]models.DirectBusinessMessage, error)
	SendMessage(conversationID string, requesterID string, input models.SendBusinessChatMessageInput) (*models.DirectBusinessMessage, error)
}

type businessChatService struct {
	repo BusinessChatRepository
	hub  *realtime.Hub
}

func NewBusinessChatService(repo BusinessChatRepository, hub *realtime.Hub) BusinessChatService {
	return &businessChatService{
		repo: repo,
		hub:  hub,
	}
}

func (s *businessChatService) InitiateChat(customerID string, businessID string, input models.InitiateBusinessChatInput) (*models.DirectBusinessConversation, *models.DirectBusinessMessage, bool, error) {
	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, nil, false, errors.New("business_id wajib diisi")
	}

	business, err := s.repo.GetBusinessByID(businessID)
	if err != nil || business == nil {
		return nil, nil, false, errors.New("Bisnis/Toko tidak ditemukan")
	}

	// Cegah chat dengan toko milik sendiri
	if business.AccountID == customerID {
		return nil, nil, false, errors.New("Anda tidak dapat memulai chat dengan toko milik Anda sendiri")
	}

	// 1. Cek apakah percakapan sudah ada
	existingConv, err := s.repo.FindConversation(businessID, customerID)
	if err == nil && existingConv != nil {
		// Percakapan sudah ada
		var lastMsg *models.DirectBusinessMessage
		// Jika ada pesan awal baru yang dikirim saat initiate ulang, kirimkan pesan tersebut
		if strings.TrimSpace(input.Message) != "" || (input.ProductID != nil && *input.ProductID != "") {
			sendInput := models.SendBusinessChatMessageInput{
				Content:   strings.TrimSpace(input.Message),
				ProductID: input.ProductID,
			}
			if input.ProductID != nil && *input.ProductID != "" {
				sendInput.MessageType = "product_inquiry"
			}
			lastMsg, _ = s.SendMessage(existingConv.ConversationID, customerID, sendInput)
		}
		return existingConv, lastMsg, false, nil
	}

	// 2. Buat percakapan baru
	now := time.Now()
	convID := utils.GenerateUUID("BCONV")

	preview := "Memulai percakapan dengan toko"
	if strings.TrimSpace(input.Message) != "" {
		preview = strings.TrimSpace(input.Message)
	}

	newConv := &models.DirectBusinessConversation{
		ConversationID:     convID,
		BusinessID:         businessID,
		CustomerID:         customerID,
		LastMessagePreview: preview,
		LastMessageAt:      now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Ambil data customer
	customer, _ := s.repo.GetAccountByID(customerID)
	senderName := "Pelanggan"
	if customer != nil {
		if customer.FirstName != "" {
			senderName = customer.FirstName
		} else if customer.Username != "" {
			senderName = customer.Username
		}
	}

	var firstMsg *models.DirectBusinessMessage
	if strings.TrimSpace(input.Message) != "" || (input.ProductID != nil && *input.ProductID != "") {
		msgType := "text"
		content := strings.TrimSpace(input.Message)
		if input.ProductID != nil && *input.ProductID != "" {
			msgType = "product_inquiry"
			if content == "" {
				content = "Halo, saya tertarik dengan produk ini"
			}
		}

		firstMsg = &models.DirectBusinessMessage{
			MessageID:      utils.GenerateUUID("BCM"),
			ConversationID: convID,
			SenderID:       customerID,
			SenderType:     "customer",
			SenderName:     senderName,
			MessageType:    msgType,
			Content:        content,
			ProductID:      input.ProductID,
			IsRead:         false,
			CreatedAt:      now,
		}
		newConv.LastMessagePreview = content

		// Preload product detail
		if input.ProductID != nil && *input.ProductID != "" {
			firstMsg.ProductDetail, _ = s.repo.GetProductByID(*input.ProductID)
		}
	}

	if err := s.repo.CreateConversation(newConv, firstMsg); err != nil {
		return nil, nil, false, errors.New("Gagal membuat percakapan dengan toko")
	}

	newConv.Business = business
	newConv.Customer = customer

	// Broadcast WS event ke seluruh staff/owner toko jika ada hub
	if s.hub != nil && firstMsg != nil {
		event := realtime.WSEvent{
			Type: "BUSINESS_CHAT_NEW_MESSAGE",
			Data: firstMsg,
		}
		staffIDs, _ := s.repo.GetMerchantStaffAccountIDs(businessID)
		for _, staffID := range staffIDs {
			s.hub.SendToAccount(staffID, event)
		}
	}

	return newConv, firstMsg, true, nil
}

func (s *businessChatService) GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetCustomerConversations(customerID, limit, cursor)
}

func (s *businessChatService) GetMerchantConversations(businessID string, requesterID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error) {
	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, errors.New("business_id wajib diisi")
	}

	hasAccess, err := s.repo.VerifyMerchantAccess(businessID, requesterID)
	if err != nil || !hasAccess {
		return nil, errors.New("Anda tidak memiliki hak akses untuk mengelola chat toko ini")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.repo.GetMerchantConversations(businessID, limit, cursor)
}

func (s *businessChatService) GetMessages(conversationID string, requesterID string, limit int, cursor string) ([]models.DirectBusinessMessage, error) {
	conv, err := s.repo.GetConversationByID(conversationID)
	if err != nil || conv == nil {
		return nil, errors.New("Percakapan tidak ditemukan")
	}

	isCustomer := conv.CustomerID == requesterID
	isMerchant := false
	if !isCustomer {
		hasAccess, _ := s.repo.VerifyMerchantAccess(conv.BusinessID, requesterID)
		if hasAccess {
			isMerchant = true
		}
	}

	if !isCustomer && !isMerchant {
		return nil, errors.New("Anda tidak memiliki akses ke percakapan ini")
	}

	readerType := "customer"
	if isMerchant {
		readerType = "merchant"
	}
	_ = s.repo.MarkMessagesAsRead(conversationID, readerType)

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.repo.GetMessagesByConversationID(conversationID, limit, cursor)
}

func (s *businessChatService) SendMessage(conversationID string, requesterID string, input models.SendBusinessChatMessageInput) (*models.DirectBusinessMessage, error) {
	conv, err := s.repo.GetConversationByID(conversationID)
	if err != nil || conv == nil {
		return nil, errors.New("Percakapan tidak ditemukan")
	}

	isCustomer := conv.CustomerID == requesterID
	isMerchant := false
	if !isCustomer {
		hasAccess, _ := s.repo.VerifyMerchantAccess(conv.BusinessID, requesterID)
		if hasAccess {
			isMerchant = true
		}
	}

	if !isCustomer && !isMerchant {
		return nil, errors.New("Anda tidak memiliki akses untuk mengirim pesan di percakapan ini")
	}

	senderType := "customer"
	senderName := "Pelanggan"
	if isMerchant {
		senderType = "merchant"
		if conv.Business != nil && conv.Business.BusinessName != "" {
			senderName = conv.Business.BusinessName
		} else {
			senderName = "Penjual"
		}
	} else {
		customer, _ := s.repo.GetAccountByID(requesterID)
		if customer != nil {
			if customer.FirstName != "" {
				senderName = customer.FirstName
			} else if customer.Username != "" {
				senderName = customer.Username
			}
		}
	}

	msgType := "text"
	if input.MessageType != "" {
		msgType = input.MessageType
	}
	if input.MediaURL != nil && *input.MediaURL != "" && msgType == "text" {
		msgType = "image"
	}
	if input.ProductID != nil && *input.ProductID != "" {
		msgType = "product_inquiry"
	}

	now := time.Now()
	message := &models.DirectBusinessMessage{
		MessageID:      utils.GenerateUUID("BCM"),
		ConversationID: conversationID,
		SenderID:       requesterID,
		SenderType:     senderType,
		SenderName:     senderName,
		MessageType:    msgType,
		Content:        strings.TrimSpace(input.Content),
		MediaURL:       input.MediaURL,
		ProductID:      input.ProductID,
		IsRead:         false,
		CreatedAt:      now,
	}

	if err := s.repo.SaveMessage(message); err != nil {
		return nil, errors.New("Gagal mengirim pesan")
	}

	_ = s.repo.UpdateConversationLastMessage(conversationID, message.Content, now)

	// Load product detail jika ada
	if message.ProductID != nil && *message.ProductID != "" {
		message.ProductDetail, _ = s.repo.GetProductByID(*message.ProductID)
	}

	// Broadcast via WebSocket
	if s.hub != nil {
		event := realtime.WSEvent{
			Type: "BUSINESS_CHAT_NEW_MESSAGE",
			Data: message,
		}

		if isCustomer {
			// Kirim ke seluruh staf dan owner toko
			staffIDs, _ := s.repo.GetMerchantStaffAccountIDs(conv.BusinessID)
			for _, staffID := range staffIDs {
				s.hub.SendToAccount(staffID, event)
			}
		} else {
			// Kirim ke customer
			s.hub.SendToAccount(conv.CustomerID, event)
		}
	}

	return message, nil
}
