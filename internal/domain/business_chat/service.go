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
	GetMessages(conversationID string, requesterID string, limit int, cursor string) ([]models.DirectBusinessMessage, *models.SignalPostReference, error)
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

	// Prevent customers from chatting with their own or managed stores.
	if business.AccountID == customerID {
		return nil, nil, false, errors.New("Anda tidak dapat memulai chat dengan toko milik Anda sendiri")
	}
	if isStaff, _ := s.repo.VerifyMerchantAccess(businessID, customerID); isStaff {
		return nil, nil, false, errors.New("Anda tidak dapat memulai chat dengan toko yang Anda kelola")
	}

	// 1. Check if conversation already exists.
	existingConv, err := s.repo.FindConversation(businessID, customerID)
	if err == nil && existingConv != nil {
		// Existing conversation found.
		var lastMsg *models.DirectBusinessMessage
		// Dispatch initial message if provided on re-initiation.
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
		if input.SignalID != nil && *input.SignalID != "" {
			if database.DB != nil {
				var sigCustomerID string
				if errSig := database.DB.Table("signals").Select("customer_id").Where("signal_id = ?", *input.SignalID).Scan(&sigCustomerID).Error; errSig != nil || sigCustomerID != customerID {
					return nil, nil, false, errors.New("signal_id tidak valid atau bukan milik Anda")
				}
			}
			_ = s.repo.UpdateConversationSignalID(existingConv.ConversationID, *input.SignalID)
			existingConv.SignalID = input.SignalID
		}
		existingConv.LastMessagePreview, _ = crypto.Decrypt(existingConv.LastMessagePreview)
		return existingConv, lastMsg, false, nil
	}

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
		SignalID:           input.SignalID,
		LastMessagePreview: preview,
		LastMessageAt:      now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

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

	// Encrypt message body and preview before database persistence.
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

		cleanContent := strings.TrimSpace(firstMsg.Content)
		isTemplate := strings.EqualFold(cleanContent, "halo kak, saya ingin bertanya mengenai toko anda.") ||
			strings.Contains(strings.ToLower(cleanContent), "halo kak, saya ingin bertanya mengenai toko anda")

		if !isTemplate && s.notifService != nil && len(staffIDs) > 0 {
			go func() {
				title := "Pesan Baru dari Pelanggan"
				body := formatChatPushBody(firstMsg.MessageType, firstMsg.Content)
				data := map[string]interface{}{
					"type":            "chat",
					"target_id":       convID,
					"conversation_id": convID,
					"business_id":     businessID,
					"customer_id":     customerID,
					"customer_name":   senderName,
					"sender_name":     senderName,
					"message_id":      firstMsg.MessageID,
				}
				s.notifService.SendNotification(
					staffIDs,
					customerID,
					"chat",
					convID,
					title,
					body,
					data,
					nil,
				)
			}()
		} else if isTemplate && s.notifService != nil && len(offlineStaffIDs) > 0 {
			go func() {
				title := senderName
				body := formatChatPushBody(firstMsg.MessageType, firstMsg.Content)
				pushData := map[string]string{
					"conversation_id": convID,
					"type":            "chat",
					"action":          "open_business_chat",
					"business_id":     businessID,
					"sender_type":     "customer",
					"sender_name":     senderName,
					"target_type":     "chat",
					"target_id":       convID,
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

func (s *businessChatService) GetMessages(conversationID string, requesterID string, limit int, cursor string) ([]models.DirectBusinessMessage, *models.SignalPostReference, error) {
	conv, err := s.repo.GetConversationByID(conversationID)
	if err != nil || conv == nil {
		return nil, nil, errors.New("Percakapan tidak ditemukan")
	}

	isCustomer := conv.CustomerID == requesterID
	isMerchant := false
	if !isCustomer {
		hasAccess, _ := s.repo.VerifyMerchantAccess(conv.BusinessID, requesterID)
		if hasAccess {
			if !permissions.HasPermission(database.DB, requesterID, conv.BusinessID, models.PermManageBusinessChat) {
				return nil, nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengelola chat pelanggan")
			}
			isMerchant = true
		}
	}

	if !isCustomer && !isMerchant {
		return nil, nil, errors.New("Anda tidak memiliki akses ke percakapan ini")
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
		return nil, nil, err
	}

	for i := range messages {
		messages[i].Content, _ = crypto.Decrypt(messages[i].Content)
		messages[i].MediaURL, _ = crypto.DecryptPointer(messages[i].MediaURL)
	}

	var signalRef *models.SignalPostReference
	if conv.SignalID != nil && *conv.SignalID != "" {
		signalRef = s.repo.GetSignalPostReferenceByID(*conv.SignalID)
	}

	return messages, signalRef, nil
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

	// Load product details if referenced.
	if message.ProductID != nil && *message.ProductID != "" {
		message.ProductDetail, _ = s.repo.GetProductByID(*message.ProductID)
	}

	event := realtime.WSEvent{
		Type: "BUSINESS_CHAT_NEW_MESSAGE",
		Data: message,
	}

	var offlineAccountIDs []string
	if isCustomer {
		// Dispatch to store staff and owner.
		staffIDs, _ := s.repo.GetMerchantStaffAccountIDs(conv.BusinessID)
		for _, staffID := range staffIDs {
			if s.hub != nil && s.hub.IsOnline(staffID) {
				s.hub.SendToAccount(staffID, event)
			} else {
				offlineAccountIDs = append(offlineAccountIDs, staffID)
			}
		}
	} else {
		// Dispatch to customer.
		if s.hub != nil && s.hub.IsOnline(conv.CustomerID) {
			s.hub.SendToAccount(conv.CustomerID, event)
		} else {
			offlineAccountIDs = append(offlineAccountIDs, conv.CustomerID)
		}
	}

	if isCustomer {
		cleanContent := strings.TrimSpace(message.Content)
		isTemplate := strings.EqualFold(cleanContent, "halo kak, saya ingin bertanya mengenai toko anda.") ||
			strings.Contains(strings.ToLower(cleanContent), "halo kak, saya ingin bertanya mengenai toko anda")

		// Create in-app notification for merchant staff for non-template messages.
		staffIDs, _ := s.repo.GetMerchantStaffAccountIDs(conv.BusinessID)
		if !isTemplate && s.notifService != nil && len(staffIDs) > 0 {
			go func(recipients []string, sName string) {
				title := "Pesan Baru dari Pelanggan"
				body := formatChatPushBody(message.MessageType, message.Content)
				data := map[string]interface{}{
					"type":            "chat",
					"target_id":       conversationID,
					"conversation_id": conversationID,
					"business_id":     conv.BusinessID,
					"customer_id":     conv.CustomerID,
					"customer_name":   sName,
					"sender_name":     sName,
					"message_id":      message.MessageID,
				}
				s.notifService.SendNotification(
					recipients,
					requesterID,
					"chat",
					conversationID,
					title,
					body,
					data,
					nil,
				)
			}(staffIDs, senderName)
		} else if isTemplate && s.notifService != nil && len(offlineAccountIDs) > 0 {
			go func() {
				title := senderName
				body := formatChatPushBody(message.MessageType, message.Content)
				pushData := map[string]string{
					"conversation_id": conversationID,
					"type":            "chat",
					"action":          "open_business_chat",
					"business_id":     conv.BusinessID,
					"sender_type":     senderType,
					"sender_name":     senderName,
					"target_type":     "chat",
					"target_id":       conversationID,
				}
				_ = s.notifService.SendPushToAccounts(offlineAccountIDs, title, body, pushData)
			}()
		}
	} else {
		// Merchant-to-customer message flow.
		if s.notifService != nil && len(offlineAccountIDs) > 0 {
			go func() {
				title := senderName
				body := formatChatPushBody(message.MessageType, message.Content)
				pushData := map[string]string{
					"conversation_id": conversationID,
					"type":            "chat",
					"action":          "open_business_chat",
					"business_id":     conv.BusinessID,
					"sender_type":     senderType,
					"sender_name":     senderName,
					"target_type":     "chat",
					"target_id":       conversationID,
				}
				_ = s.notifService.SendPushToAccounts(offlineAccountIDs, title, body, pushData)
			}()
		}
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

