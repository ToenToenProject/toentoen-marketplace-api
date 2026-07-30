package signal

import (
	"errors"
	"fmt"
	"log"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type SignalService interface {
	// Customer
	CreateSignal(customerID string, input models.CreateSignalInput) (*models.Signal, error)
	GetMySignals(customerID string) ([]models.Signal, error)
	GetSignalDetail(signalID, accountID, role string) (*models.Signal, error)
	CompleteSignal(customerID, signalID string) error

	// Merchant
	ExploreSignals(accountID, businessID string, lat, lng float64) ([]models.SignalExploreItem, error)
	GetMerchantSignalDetail(signalID string) (*models.Signal, error)
	ReplyToSignal(accountID, role, signalID string, input models.InitiateConversationInput) (*models.SignalConversation, error)

	// Shared (Customer & Merchant)
	GetMessages(accountID, conversationID string, page, limit int) ([]models.SignalMessage, int64, error)
	SendMessage(accountID, role, conversationID string, input models.SendMessageInput) (*models.SignalMessage, error)
}

type signalService struct {
	signalRepo   SignalRepository
	businessRepo business.BusinessRepository
	hub          *realtime.Hub
}

func NewSignalService(signalRepo SignalRepository, businessRepo business.BusinessRepository, hub *realtime.Hub) SignalService {
	return &signalService{
		signalRepo:   signalRepo,
		businessRepo: businessRepo,
		hub:          hub,
	}
}

// ================================================================
// CUSTOMER
// ================================================================

func (s *signalService) CreateSignal(customerID string, input models.CreateSignalInput) (*models.Signal, error) {
	if input.Description == "" {
		return nil, errors.New("deskripsi kebutuhan wajib diisi")
	}
	if input.Category == "" {
		return nil, errors.New("kategori wajib diisi")
	}
	if input.Latitude == 0 || input.Longitude == 0 {
		return nil, errors.New("lokasi (latitude/longitude) wajib diisi")
	}

	radiusKm := input.RadiusKm
	if radiusKm <= 0 {
		radiusKm = 5 // default 5 km
	}

	expiresInHours := input.ExpiresInHours
	if expiresInHours <= 0 {
		expiresInHours = 24 // default 24 hours
	} else if expiresInHours > 72 {
		expiresInHours = 72 // max 3 days
	}

	signal := models.Signal{
		SignalID:    utils.GenerateUUID("SIG"),
		CustomerID:  customerID,
		Description: input.Description,
		Category:    input.Category,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		RadiusKm:    radiusKm,
		IsUrgent:    input.IsUrgent,
		Status:      "active",
		ExpiresAt:   time.Now().Add(time.Duration(expiresInHours) * time.Hour),
	}

	// 1. Simpan signal ke database
	if err := s.signalRepo.CreateSignal(&signal); err != nil {
		return nil, errors.New("gagal membuat signal")
	}

	// 2. Update kolom PostGIS location via raw SQL
	if err := s.signalRepo.UpdateSignalLocation(signal.SignalID, input.Latitude, input.Longitude); err != nil {
		fmt.Printf("Warning: gagal update lokasi PostGIS untuk signal %s: %v\n", signal.SignalID, err)
	}

	// 3. Simpan gambar lampiran (jika ada)
	if len(input.ImageURLs) > 0 {
		var images []models.SignalImage
		for _, url := range input.ImageURLs {
			images = append(images, models.SignalImage{
				ImageID:  utils.GenerateUUID("IMG"),
				SignalID: signal.SignalID,
				ImageURL: url,
			})
		}
		if err := s.signalRepo.CreateSignalImages(images); err != nil {
			fmt.Printf("Warning: gagal menyimpan gambar signal: %v\n", err)
		}
		signal.Images = images
	}

	return &signal, nil
}

func (s *signalService) GetMySignals(customerID string) ([]models.Signal, error) {
	return s.signalRepo.GetMySignals(customerID)
}

func (s *signalService) GetSignalDetail(signalID, accountID, role string) (*models.Signal, error) {
	signal, err := s.signalRepo.GetSignalWithConversations(signalID)
	if err != nil {
		return nil, fmt.Errorf("signal dengan ID %s tidak ditemukan", signalID)
	}

	// Customer hanya boleh melihat signal miliknya sendiri
	if role == "customer" && signal.CustomerID != accountID {
		return nil, errors.New("anda tidak memiliki akses ke signal ini")
	}

	return signal, nil
}

func (s *signalService) CompleteSignal(customerID, signalID string) error {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return errors.New("signal tidak ditemukan")
	}

	if signal.CustomerID != customerID {
		return errors.New("anda bukan pemilik signal ini")
	}

	if signal.Status != "active" {
		return fmt.Errorf("signal tidak dalam status aktif (status saat ini: %s)", signal.Status)
	}

	if err := s.signalRepo.CompleteSignal(signalID); err != nil {
		return err
	}

	// 🔔 REAL-TIME: Beritahu semua merchant yang punya conversation
	go s.notifySignalCompleted(signalID)

	return nil
}

// notifySignalCompleted mengirimkan event signal_completed ke semua merchant terkait
func (s *signalService) notifySignalCompleted(signalID string) {
	merchantIDs, err := s.signalRepo.GetMerchantIDsBySignal(signalID)
	if err != nil || len(merchantIDs) == 0 {
		return
	}

	event := realtime.WSEvent{
		Type: "signal_completed",
		Data: map[string]string{
			"signal_id": signalID,
			"message":   "Signal telah diselesaikan oleh customer. Semua percakapan terkunci.",
		},
	}

	// Kirim via WebSocket
	s.hub.BroadcastToUsers(merchantIDs, event)

	// FCM fallback untuk merchant yang offline
	for _, merchantID := range merchantIDs {
		if !s.hub.IsOnline(merchantID) {
			tokens, _ := s.signalRepo.GetFCMTokensByAccountID(merchantID)
			if len(tokens) > 0 {
				config.SendPushNotification(tokens,
					"Signal Diselesaikan",
					"Customer telah menyelesaikan signal. Percakapan ditutup.",
					map[string]string{
						"type":      "signal_completed",
						"signal_id": signalID,
					},
				)
			}
		}
	}
}

// ================================================================
// MERCHANT
// ================================================================

func (s *signalService) ExploreSignals(accountID, businessID string, lat, lng float64) ([]models.SignalExploreItem, error) {
	var activeBusiness *models.Business
	var err error

	if businessID != "" {
		activeBusiness, err = s.businessRepo.GetSpecificBusiness(accountID, businessID)
	} else {
		businesses, errRepo := s.businessRepo.FindBusinessesByOwnerID(accountID)
		if errRepo == nil && len(businesses) > 0 {
			activeBusiness = &businesses[0]
		} else {
			err = fmt.Errorf("tidak ada bisnis yang ditemukan")
		}
	}

	if err != nil || activeBusiness == nil {
		return nil, fmt.Errorf("bisnis tidak ditemukan atau akses ditolak")
	}

	var categories []string
	for _, cat := range activeBusiness.Categories {
		categories = append(categories, cat.Name)
	}

	if len(categories) == 0 {
		return []models.SignalExploreItem{}, nil
	}

	if lat == 0 && lng == 0 {
		if activeBusiness.Latitude != nil && activeBusiness.Longitude != nil {
			lat = *activeBusiness.Latitude
			lng = *activeBusiness.Longitude
		}
	}

	if lat == 0 && lng == 0 {
		return nil, errors.New("lokasi tidak tersedia, kirim parameter lat dan lng")
	}

	return s.signalRepo.ExploreActiveSignals(lat, lng, categories)
}

func (s *signalService) GetMerchantSignalDetail(signalID string) (*models.Signal, error) {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return nil, fmt.Errorf("signal dengan ID %s tidak ditemukan", signalID)
	}
	return signal, nil
}

func (s *signalService) ReplyToSignal(accountID, role, signalID string, input models.InitiateConversationInput) (*models.SignalConversation, error) {
	if !s.signalRepo.VerifyBusinessAccess(accountID, role, input.BusinessID) {
		return nil, errors.New("anda tidak memiliki akses ke bisnis ini")
	}

	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return nil, errors.New("signal tidak ditemukan")
	}
	if signal.Status != "active" {
		return nil, errors.New("signal ini sudah tidak aktif")
	}

	existing, _ := s.signalRepo.FindExistingConversation(signalID, accountID)
	if existing != nil {
		return nil, errors.New("anda sudah pernah membalas signal ini")
	}

	conv := models.SignalConversation{
		ConversationID: utils.GenerateUUID("CONV"),
		SignalID:       signalID,
		MerchantID:     accountID,
		BusinessID:     input.BusinessID,
		Status:         "open",
	}

	if err := s.signalRepo.CreateConversation(&conv); err != nil {
		return nil, errors.New("gagal membuat ruang percakapan")
	}

	// Kirim pesan pertama dari merchant
	var firstMsg *models.SignalMessage
	if input.Message != "" {
		msg := models.SignalMessage{
			MessageID:      utils.GenerateUUID("MSG"),
			ConversationID: conv.ConversationID,
			SenderType:     "merchant",
			SenderID:       accountID,
			MessageText:    input.Message,
		}
		if input.ImageURL != "" {
			msg.AttachmentURL = &input.ImageURL
		}
		if err := s.signalRepo.CreateMessage(&msg); err != nil {
			fmt.Printf("Warning: gagal menyimpan pesan pertama: %v\n", err)
		}
		firstMsg = &msg
	}

	// 🔔 REAL-TIME: Beritahu customer bahwa merchant telah membalas
	go s.notifyNewConversation(signal.CustomerID, &conv, firstMsg)

	return &conv, nil
}

// notifyNewConversation mengirimkan event new_conversation ke customer pemilik signal
func (s *signalService) notifyNewConversation(customerID string, conv *models.SignalConversation, firstMsg *models.SignalMessage) {
	event := realtime.WSEvent{
		Type: "new_conversation",
		Data: map[string]interface{}{
			"conversation_id": conv.ConversationID,
			"signal_id":       conv.SignalID,
			"merchant_id":     conv.MerchantID,
			"business_id":     conv.BusinessID,
		},
	}

	// Kirim via WebSocket
	s.hub.SendToUser(customerID, event)

	// FCM fallback jika customer offline
	if !s.hub.IsOnline(customerID) {
		tokens, _ := s.signalRepo.GetFCMTokensByAccountID(customerID)
		if len(tokens) > 0 {
			body := "Sebuah toko telah menanggapi permintaan Anda!"
			if firstMsg != nil && firstMsg.MessageText != "" {
				body = firstMsg.MessageText
			}
			config.SendPushNotification(tokens,
				"🔔 Ada Tanggapan Baru!",
				body,
				map[string]string{
					"type":            "new_conversation",
					"conversation_id": conv.ConversationID,
					"signal_id":       conv.SignalID,
				},
			)
		}
	}
}

// ================================================================
// SHARED (CUSTOMER & MERCHANT)
// ================================================================

func (s *signalService) GetMessages(accountID, conversationID string, page, limit int) ([]models.SignalMessage, int64, error) {
	conv, err := s.signalRepo.GetConversationByID(conversationID)
	if err != nil {
		return nil, 0, errors.New("percakapan tidak ditemukan")
	}

	signal, err := s.signalRepo.GetSignalByID(conv.SignalID)
	if err != nil {
		return nil, 0, errors.New("signal tidak ditemukan")
	}

	if signal.CustomerID != accountID && conv.MerchantID != accountID {
		return nil, 0, errors.New("anda tidak memiliki akses ke percakapan ini")
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	_ = s.signalRepo.MarkMessagesAsRead(conversationID, accountID)

	return s.signalRepo.GetMessagesByConversation(conversationID, page, limit)
}

func (s *signalService) SendMessage(accountID, role, conversationID string, input models.SendMessageInput) (*models.SignalMessage, error) {
	if input.MessageText == "" && input.AttachmentURL == "" {
		return nil, errors.New("pesan atau lampiran wajib diisi")
	}

	conv, err := s.signalRepo.GetConversationByID(conversationID)
	if err != nil {
		return nil, errors.New("percakapan tidak ditemukan")
	}

	if conv.Status != "open" {
		return nil, errors.New("percakapan ini sudah terkunci karena signal telah diselesaikan")
	}

	signal, err := s.signalRepo.GetSignalByID(conv.SignalID)
	if err != nil {
		return nil, errors.New("signal tidak ditemukan")
	}

	var senderType string
	if signal.CustomerID == accountID {
		senderType = "customer"
	} else if conv.MerchantID == accountID {
		senderType = "merchant"
	} else {
		return nil, errors.New("anda tidak memiliki akses ke percakapan ini")
	}

	msg := models.SignalMessage{
		MessageID:      utils.GenerateUUID("MSG"),
		ConversationID: conversationID,
		SenderType:     senderType,
		SenderID:       accountID,
		MessageText:    input.MessageText,
	}

	if input.AttachmentURL != "" {
		msg.AttachmentURL = &input.AttachmentURL
	}

	if err := s.signalRepo.CreateMessage(&msg); err != nil {
		return nil, errors.New("gagal mengirim pesan")
	}

	// 🔔 REAL-TIME: Kirim event new_message ke lawan bicara
	go s.notifyNewMessage(accountID, conv, signal, &msg)

	return &msg, nil
}

// notifyNewMessage mengirimkan event new_message ke lawan bicara
func (s *signalService) notifyNewMessage(senderID string, conv *models.SignalConversation, signal *models.Signal, msg *models.SignalMessage) {
	// Tentukan siapa penerima pesan
	var recipientID string
	if senderID == signal.CustomerID {
		recipientID = conv.MerchantID
	} else {
		recipientID = signal.CustomerID
	}

	event := realtime.WSEvent{
		Type: "new_message",
		Data: msg,
	}

	// Kirim via WebSocket
	s.hub.SendToUser(recipientID, event)

	// FCM fallback jika penerima offline
	if !s.hub.IsOnline(recipientID) {
		tokens, err := s.signalRepo.GetFCMTokensByAccountID(recipientID)
		if err != nil || len(tokens) == 0 {
			return
		}

		title := "💬 Pesan Baru"
		body := msg.MessageText
		if body == "" {
			body = "Mengirim lampiran"
		}

		err = config.SendPushNotification(tokens, title, body, map[string]string{
			"type":            "new_message",
			"conversation_id": conv.ConversationID,
			"signal_id":       conv.SignalID,
			"sender_id":       senderID,
		})
		if err != nil {
			log.Printf("⚠️ Gagal kirim FCM ke %s: %v", recipientID, err)
		}
	}
}
