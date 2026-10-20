package business_chat

import (
	"errors"
	"strings"
	"time"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/database"
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
	repo         BusinessChatRepository
	hub          *realtime.Hub
	notifService notification.NotificationService
}

func NewBusinessChatService(repo BusinessChatRepository, hub *realtime.Hub, notifService notification.NotificationService) BusinessChatService {
	return &businessChatService{
		repo:         repo,
		hub:          hub,
		notifService: notifService,
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

	// Cegah chat dengan toko milik sendiri atau toko yang dikelola
	if business.AccountID == customerID {
		return nil, nil, false, errors.New("Anda tidak dapat memulai chat dengan toko milik Anda sendiri")
	}
	if isStaff, _ := s.repo.VerifyMerchantAccess(businessID, customerID); isStaff {
		return nil, nil, false, errors.New("Anda tidak dapat memulai chat dengan toko yang Anda kelola")
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
		existingConv.LastMessagePreview, _ = crypto.Decrypt(existingConv.LastMessagePreview)
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

	// Enkripsi pesan & preview sebelum simpan ke database
	newConvToSave := *newConv
	newConvToSave.LastMessagePreview, _ = crypto.Encrypt(newConv.LastMessagePreview)

	var firstMsgToSave *models.DirectBusinessMessage
	if firstMsg != nil {
		msgCopy := *firstMsg
		msgCopy.Content, _ = crypto.Encrypt(firstMsg.Content)
		msgCopy.MediaURL, _ = crypto.EncryptPointer(firstMsg.MediaURL)
		firstMsgToSave = &msgCopy
	}

	if err := s.repo.CreateConversation(&newConvToSave, firstMsgToSave); err != nil {
		return nil, nil, false, errors.New("Gagal membuat percakapan dengan toko")
	}

	newConv.Business = business
	newConv.Customer = customer

	// Broadcast WS event ke seluruh staff/owner toko jika ada hub & kirim Push Notification jika offline
	if firstMsg != nil {
		event := realtime.WSEvent{
			Type: "BUSINESS_CHAT_NEW_MESSAGE",
			Data: firstMsg,
		}
		staffIDs, _ := s.repo.GetMerchantStaffAccountIDs(businessID)
		var offlineStaffIDs []string
		for _, staffID := range staffIDs {
			if s.hub != nil && s.hub.IsOnline(staffID) {
				s.hub.SendToAccount(staffID, event)
			} else {
				offlineStaffIDs = append(offlineStaffIDs, staffID)
			}
		}

		if s.notifService != nil && len(offlineStaffIDs) > 0 {
			go func() {
				title := senderName
				body := formatChatPushBody(firstMsg.MessageType, firstMsg.Content)
				pushData := map[string]string{
					"conversation_id": convID,
					"type":            "business_chat",
					"action":          "open_business_chat",
					"business_id":     businessID,
					"sender_type":     "customer",
					"sender_name":     senderName,
				}
				_ = s.notifService.SendPushToAccounts(offlineStaffIDs, title, body, pushData)
			}()
		}
	}

	return newConv, firstMsg, true, nil
}

func (s *businessChatService) GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	items, err := s.repo.GetCustomerConversations(customerID, limit, cursor)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].LastMessagePreview, _ = crypto.Decrypt(items[i].LastMessagePreview)
	}
	return items, nil
}

func (s *businessChatService) GetMerchantConversations(businessID string, requesterID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error) {
	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, errors.New("business_id wajib diisi")
	}

	hasAccess, err := s.repo.VerifyMerchantAccess(businessID, requesterID)
	if err != nil || !hasAccess {
		return nil, errors.New("forbidden: Anda tidak memiliki hak akses untuk mengelola chat toko ini")
	}
	if !permissions.HasPermission(database.DB, requesterID, businessID, models.PermManageBusinessChat) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengelola chat pelanggan")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	items, err := s.repo.GetMerchantConversations(businessID, limit, cursor)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].LastMessagePreview, _ = crypto.Decrypt(items[i].LastMessagePreview)
	}
	return items, nil
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
			if !permissions.HasPermission(database.DB, requesterID, conv.BusinessID, models.PermManageBusinessChat) {
				return nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengelola chat pelanggan")
			}
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

	messages, err := s.repo.GetMessagesByConversationID(conversationID, limit, cursor)
	if err != nil {
		return nil, err
	}

	for i := range messages {
		messages[i].Content, _ = crypto.Decrypt(messages[i].Content)
		messages[i].MediaURL, _ = crypto.DecryptPointer(messages[i].MediaURL)
	}

	return messages, nil
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
			if !permissions.HasPermission(database.DB, requesterID, conv.BusinessID, models.PermManageBusinessChat) {
				return nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengelola chat pelanggan")
			}
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
		if conv.Business != nil && conv.Business.BusinessName != nil && *conv.Business.BusinessName != "" {
			senderName = *conv.Business.BusinessName
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

	// Enkripsi sebelum disimpan ke basis data
	msgToSave := *message
	msgToSave.Content, _ = crypto.Encrypt(message.Content)
	msgToSave.MediaURL, _ = crypto.EncryptPointer(message.MediaURL)

	if err := s.repo.SaveMessage(&msgToSave); err != nil {
		return nil, errors.New("Gagal mengirim pesan")
	}

	encPreview, _ := crypto.Encrypt(message.Content)
	_ = s.repo.UpdateConversationLastMessage(conversationID, encPreview, now)

	// Load product detail jika ada
	if message.ProductID != nil && *message.ProductID != "" {
		message.ProductDetail, _ = s.repo.GetProductByID(*message.ProductID)
	}

	// Broadcast via WebSocket (kirim data plaintext ke websocket) & Push Notification jika offline
	event := realtime.WSEvent{
		Type: "BUSINESS_CHAT_NEW_MESSAGE",
		Data: message,
	}

	var offlineAccountIDs []string
	if isCustomer {
		// Kirim ke seluruh staf dan owner toko
		staffIDs, _ := s.repo.GetMerchantStaffAccountIDs(conv.BusinessID)
		for _, staffID := range staffIDs {
			if s.hub != nil && s.hub.IsOnline(staffID) {
				s.hub.SendToAccount(staffID, event)
			} else {
				offlineAccountIDs = append(offlineAccountIDs, staffID)
			}
		}
	} else {
		// Kirim ke customer
		if s.hub != nil && s.hub.IsOnline(conv.CustomerID) {
			s.hub.SendToAccount(conv.CustomerID, event)
		} else {
			offlineAccountIDs = append(offlineAccountIDs, conv.CustomerID)
		}
	}

	if s.notifService != nil && len(offlineAccountIDs) > 0 {
		go func() {
			title := senderName
			body := formatChatPushBody(message.MessageType, message.Content)
			pushData := map[string]string{
				"conversation_id": conversationID,
				"type":            "business_chat",
				"action":          "open_business_chat",
				"business_id":     conv.BusinessID,
				"sender_type":     senderType,
				"sender_name":     senderName,
			}
			_ = s.notifService.SendPushToAccounts(offlineAccountIDs, title, body, pushData)
		}()
	}

	return message, nil
}

func formatChatPushBody(msgType, content string) string {
	switch msgType {
	case "image":
		if strings.TrimSpace(content) != "" {
			return utils.TruncateString(content, 100)
		}
		return "Mengirim sebuah gambar"
	case "product_inquiry":
		if strings.TrimSpace(content) != "" {
			return utils.TruncateString(content, 100)
		}
		return "Menanyakan produk Anda"
	default:
		if strings.TrimSpace(content) != "" {
			return utils.TruncateString(content, 100)
		}
		return "Mengirim sebuah pesan"
	}
}

