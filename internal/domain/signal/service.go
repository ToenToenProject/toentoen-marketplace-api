package signal

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type SignalService interface {
	// Customer
	CreateSignal(customerID string, input models.CreateSignalInput) (*models.Signal, error)
	UpdateSignal(customerID, signalID string, input models.UpdateSignalInput) (*models.Signal, error)
	DeleteSignal(customerID, signalID string) error
	GetMySignals(customerID string) ([]models.Signal, error)
	GetSignalDetail(signalID, accountID, role string) (*models.Signal, error)
	CompleteSignal(customerID, signalID string) error
	GetCustomerConversations(customerID string) ([]models.ConversationListItem, error)

	// Merchant
	ExploreSignals(accountID, businessID string, lat, lng float64) ([]models.SignalExploreItem, error)
	GetMerchantSignalDetail(signalID, merchantAccountID string) (*models.MerchantSignalDetailResponse, error)
	ReplyToSignal(accountID, role, signalID string, input models.InitiateConversationInput) (*models.SignalConversation, error)
	GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error)

	// Shared (Customer & Merchant)
	GetMessages(accountID, role, conversationID string, page, limit int) ([]models.SignalMessage, int64, error)
	SendMessage(accountID, role, conversationID string, input models.SendMessageInput) (*models.SignalMessage, error)
}

type signalService struct {
	signalRepo   SignalRepository
	businessRepo business.BusinessRepository
	notifService notification.NotificationService
	hub          *realtime.Hub
}

func NewSignalService(signalRepo SignalRepository, businessRepo business.BusinessRepository, notifService notification.NotificationService, hub *realtime.Hub) SignalService {
	return &signalService{
		signalRepo:   signalRepo,
		businessRepo: businessRepo,
		notifService: notifService,
		hub:          hub,
	}
}

// ================================================================
// CUSTOMER
// ================================================================

func (s *signalService) CreateSignal(customerID string, input models.CreateSignalInput) (*models.Signal, error) {
	desc := strings.TrimSpace(input.Description)
	if desc == "" {
		return nil, errors.New("deskripsi kebutuhan wajib diisi")
	}

	// Validasi Kategori: Tangkap dari database (bukan string bebas)
	var catID int64
	if input.CategoryID != nil && *input.CategoryID > 0 {
		catID = *input.CategoryID
	}

	catInput := strings.TrimSpace(input.Category)
	if catInput == "" && catID <= 0 {
		return nil, errors.New("kategori wajib dipilih dari daftar kategori yang tersedia")
	}

	validCategory, errCat := s.signalRepo.GetCategoryByNameOrID(catInput, catID)
	if errCat != nil || validCategory == nil {
		return nil, errors.New("kategori tidak valid. Silakan pilih kategori resmi yang tersedia di sistem")
	}

	cat := validCategory.Name
	resolvedCatID := validCategory.CategoryID

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
		Description: desc,
		CategoryID:  &resolvedCatID,
		Category:    cat,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		RadiusKm:    radiusKm,
		IsUrgent:    input.IsUrgent,
		Status:      "active",
		ExpiresAt:   time.Now().Add(time.Duration(expiresInHours) * time.Hour),
	}

	// 1. Simpan signal ke database
	if err := s.signalRepo.CreateSignal(&signal); err != nil {
		return nil, fmt.Errorf("gagal membuat signal: %w", err)
	}

	// 2. Update kolom PostGIS location via raw SQL
	if err := s.signalRepo.UpdateSignalLocation(signal.SignalID, input.Latitude, input.Longitude); err != nil {
		fmt.Printf("Warning: gagal update lokasi PostGIS untuk signal %s: %v\n", signal.SignalID, err)
	}

	// 3. Simpan gambar lampiran (jika ada, dukung image_urls, media_urls, atau images)
	imgList := input.ImageURLs
	if len(imgList) == 0 && len(input.MediaUrls) > 0 {
		imgList = input.MediaUrls
	}
	if len(imgList) == 0 && len(input.Images) > 0 {
		imgList = input.Images
	}

	if len(imgList) > 0 {
		var images []models.SignalImage
		for _, url := range imgList {
			cleanURL := strings.TrimSpace(url)
			if cleanURL != "" {
				images = append(images, models.SignalImage{
					ImageID:  utils.GenerateUUID("IMG"),
					SignalID: signal.SignalID,
					ImageURL: cleanURL,
				})
			}
		}
		if len(images) > 0 {
			if err := s.signalRepo.CreateSignalImages(images); err != nil {
				fmt.Printf("Warning: gagal menyimpan gambar signal: %v\n", err)
			}
			signal.Images = images
		}
	}

	// 4. Simpan tags yang dipilih customer (jika ada) — digunakan sebagai booster relevansi
	if len(input.TagIDs) > 0 {
		var signalTags []models.SignalTag
		for _, tagID := range input.TagIDs {
			if tagID > 0 {
				signalTags = append(signalTags, models.SignalTag{
					SignalID: signal.SignalID,
					TagID:    tagID,
				})
			}
		}
		if len(signalTags) > 0 {
			if err := s.signalRepo.CreateSignalTags(signalTags); err != nil {
				fmt.Printf("Warning: gagal menyimpan signal tags: %v\n", err)
			}
		}
	}

	return &signal, nil
}

func (s *signalService) UpdateSignal(customerID, signalID string, input models.UpdateSignalInput) (*models.Signal, error) {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return nil, errors.New("signal tidak ditemukan")
	}

	if signal.CustomerID != customerID {
		return nil, errors.New("anda bukan pemilik signal ini")
	}

	if signal.Status != "active" {
		return nil, fmt.Errorf("signal tidak dapat diubah karena status saat ini: %s", signal.Status)
	}

	// 1. Deskripsi jika diisi
	if desc := strings.TrimSpace(input.Description); desc != "" {
		signal.Description = desc
	}

	// 2. Kategori jika diubah
	var catID int64
	if input.CategoryID != nil && *input.CategoryID > 0 {
		catID = *input.CategoryID
	}
	catInput := strings.TrimSpace(input.Category)
	if catInput != "" || catID > 0 {
		validCategory, errCat := s.signalRepo.GetCategoryByNameOrID(catInput, catID)
		if errCat != nil || validCategory == nil {
			return nil, errors.New("kategori tidak valid. Silakan pilih kategori resmi yang tersedia di sistem")
		}
		signal.Category = validCategory.Name
		signal.CategoryID = &validCategory.CategoryID
	}

	// 3. Lokasi jika diubah
	if input.Latitude != nil && input.Longitude != nil && *input.Latitude != 0 && *input.Longitude != 0 {
		signal.Latitude = *input.Latitude
		signal.Longitude = *input.Longitude
		if err := s.signalRepo.UpdateSignalLocation(signal.SignalID, *input.Latitude, *input.Longitude); err != nil {
			fmt.Printf("Warning: gagal update lokasi PostGIS untuk signal %s: %v\n", signal.SignalID, err)
		}
	}

	// 4. Radius jika diubah
	if input.RadiusKm != nil && *input.RadiusKm > 0 {
		signal.RadiusKm = *input.RadiusKm
	}

	// 5. IsUrgent jika diubah
	if input.IsUrgent != nil {
		signal.IsUrgent = *input.IsUrgent
	}

	// 6. ExpiresInHours jika diubah
	if input.ExpiresInHours != nil && *input.ExpiresInHours > 0 {
		hours := *input.ExpiresInHours
		if hours > 72 {
			hours = 72
		}
		signal.ExpiresAt = time.Now().Add(time.Duration(hours) * time.Hour)
	}

	// 7. Simpan perubahan signal
	if err := s.signalRepo.UpdateSignal(signal); err != nil {
		return nil, fmt.Errorf("gagal mengupdate signal: %w", err)
	}

	// 8. Update Gambar jika dikirim
	imgList := input.ImageURLs
	if len(imgList) == 0 && len(input.MediaUrls) > 0 {
		imgList = input.MediaUrls
	}
	if len(imgList) == 0 && len(input.Images) > 0 {
		imgList = input.Images
	}
	if input.ImageURLs != nil || input.MediaUrls != nil || input.Images != nil {
		var images []models.SignalImage
		for _, url := range imgList {
			cleanURL := strings.TrimSpace(url)
			if cleanURL != "" {
				images = append(images, models.SignalImage{
					ImageID:  utils.GenerateUUID("IMG"),
					SignalID: signal.SignalID,
					ImageURL: cleanURL,
				})
			}
		}
		if err := s.signalRepo.ReplaceSignalImages(signal.SignalID, images); err != nil {
			fmt.Printf("Warning: gagal update gambar signal: %v\n", err)
		}
	}

	// 9. Update Tags jika dikirim
	if input.TagIDs != nil {
		var signalTags []models.SignalTag
		for _, tagID := range input.TagIDs {
			if tagID > 0 {
				signalTags = append(signalTags, models.SignalTag{
					SignalID: signal.SignalID,
					TagID:    tagID,
				})
			}
		}
		if err := s.signalRepo.ReplaceSignalTags(signal.SignalID, signalTags); err != nil {
			fmt.Printf("Warning: gagal update signal tags: %v\n", err)
		}
	}

	return s.signalRepo.GetSignalByID(signal.SignalID)
}

func (s *signalService) DeleteSignal(customerID, signalID string) error {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return errors.New("signal tidak ditemukan")
	}

	if signal.CustomerID != customerID {
		return errors.New("anda bukan pemilik signal ini")
	}

	// Beritahu merchant terkait jika ada percakapan
	go s.notifySignalCancelled(signalID, customerID)

	return s.signalRepo.DeleteSignal(signalID)
}

// notifySignalCancelled mengirimkan event signal_cancelled ke semua merchant terkait
func (s *signalService) notifySignalCancelled(signalID, customerID string) {
	merchantIDs, err := s.signalRepo.GetMerchantIDsBySignal(signalID)
	if err != nil || len(merchantIDs) == 0 {
		return
	}

	event := realtime.WSEvent{
		Type: "signal_cancelled",
		Data: map[string]string{
			"signal_id": signalID,
			"message":   "Signal telah dibatalkan/dihapus oleh customer.",
		},
	}

	// Kirim via WebSocket (live roomchat event)
	s.hub.BroadcastToUsers(merchantIDs, event)

	// Simpan ke in-app notification & push FCM
	if s.notifService != nil {
		s.notifService.SendNotification(
			merchantIDs,
			customerID,
			"signal_cancelled",
			signalID,
			"❌ Sinyal Kebutuhan Dibatalkan",
			"Customer telah membatalkan sinyal kebutuhan.",
			map[string]interface{}{
				"type":      "signal_cancelled",
				"signal_id": signalID,
			},
			nil,
		)
	}
}

func (s *signalService) GetMySignals(customerID string) ([]models.Signal, error) {
	return s.signalRepo.GetMySignals(customerID)
}

func (s *signalService) GetCustomerConversations(customerID string) ([]models.ConversationListItem, error) {
	return s.signalRepo.GetCustomerConversations(customerID)
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
	go s.notifySignalCompleted(signalID, customerID)

	return nil
}

// notifySignalCompleted mengirimkan event signal_completed ke semua merchant terkait
func (s *signalService) notifySignalCompleted(signalID, customerID string) {
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

	// Kirim via WebSocket (live roomchat event)
	s.hub.BroadcastToUsers(merchantIDs, event)

	// Simpan ke in-app notification & push FCM
	if s.notifService != nil {
		s.notifService.SendNotification(
			merchantIDs,
			customerID,
			"signal_completed",
			signalID,
			"✅ Sinyal Kebutuhan Selesai",
			"Customer telah menyelesaikan sinyal kebutuhan. Percakapan telah ditutup.",
			map[string]interface{}{
				"type":      "signal_completed",
				"signal_id": signalID,
			},
			nil,
		)
	}
}

// ================================================================
// MERCHANT
// ================================================================

func (s *signalService) ExploreSignals(accountID, businessID string, lat, lng float64) ([]models.SignalExploreItem, error) {
	var activeBusiness *models.Business
	var err error

	if businessID != "" {
		activeBusiness, err = s.businessRepo.FindBusinessByID(businessID)
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

	if lat == 0 && lng == 0 {
		if activeBusiness.Latitude != nil && activeBusiness.Longitude != nil {
			lat = *activeBusiness.Latitude
			lng = *activeBusiness.Longitude
		}
	}

	if lat == 0 && lng == 0 {
		return nil, errors.New("lokasi tidak tersedia, kirim parameter lat dan lng")
	}

	return s.signalRepo.ExploreActiveSignals(activeBusiness.BusinessID, lat, lng)
}

func (s *signalService) GetMerchantSignalDetail(signalID, merchantAccountID string) (*models.MerchantSignalDetailResponse, error) {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return nil, fmt.Errorf("signal dengan ID %s tidak ditemukan", signalID)
	}

	var myConv *models.SignalConversation
	var myConvID *string
	hasReplied := false

	if merchantAccountID != "" {
		existing, errConv := s.signalRepo.FindExistingConversation(signalID, merchantAccountID)
		if errConv == nil && existing != nil {
			hasReplied = true
			myConv = existing
			myConvID = &existing.ConversationID
		}
	}

	return &models.MerchantSignalDetailResponse{
		Signal:           signal,
		HasReplied:       hasReplied,
		MyConversationID: myConvID,
		MyConversation:   myConv,
	}, nil
}

func (s *signalService) GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error) {
	return s.signalRepo.GetMerchantConversations(merchantAccountID)
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

	existing, _ := s.signalRepo.FindExistingConversationByBusiness(signalID, input.BusinessID)
	if existing != nil {
		return nil, errors.New("toko Anda sudah pernah membalas signal ini")
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
	go s.notifyNewConversation(signal.CustomerID, accountID, &conv, firstMsg)

	return &conv, nil
}

// notifyNewConversation mengirimkan event new_conversation ke customer pemilik signal
func (s *signalService) notifyNewConversation(customerID, merchantAccountID string, conv *models.SignalConversation, firstMsg *models.SignalMessage) {
	event := realtime.WSEvent{
		Type: "new_conversation",
		Data: map[string]interface{}{
			"conversation_id": conv.ConversationID,
			"signal_id":       conv.SignalID,
			"merchant_id":     conv.MerchantID,
			"business_id":     conv.BusinessID,
		},
	}

	// Kirim via WebSocket (live roomchat event)
	s.hub.SendToUser(customerID, event)

	// Simpan ke in-app notification & push FCM
	if s.notifService != nil {
		bizName := "Toko"
		if biz, err := s.businessRepo.FindBusinessByID(conv.BusinessID); err == nil && biz != nil && biz.BusinessName != nil && *biz.BusinessName != "" {
			bizName = *biz.BusinessName
		}

		body := fmt.Sprintf("%s telah menanggapi sinyal kebutuhan Anda!", bizName)
		if firstMsg != nil && firstMsg.MessageText != "" {
			body = fmt.Sprintf("%s: %s", bizName, utils.TruncateString(firstMsg.MessageText, 80))
		}

		s.notifService.SendNotification(
			[]string{customerID},
			merchantAccountID,
			"signal_reply",
			conv.ConversationID,
			"💬 Ada Tanggapan Baru!",
			body,
			map[string]interface{}{
				"type":            "signal_reply",
				"conversation_id": conv.ConversationID,
				"signal_id":       conv.SignalID,
				"business_id":     conv.BusinessID,
				"business_name":   bizName,
			},
			nil,
		)
	}
}

// ================================================================
// SHARED (CUSTOMER & MERCHANT)
// ================================================================

func (s *signalService) GetMessages(accountID, role, conversationID string, page, limit int) ([]models.SignalMessage, int64, error) {
	conv, err := s.signalRepo.GetConversationByID(conversationID)
	if err != nil {
		return nil, 0, errors.New("percakapan tidak ditemukan")
	}

	signal, err := s.signalRepo.GetSignalByID(conv.SignalID)
	if err != nil {
		return nil, 0, errors.New("signal tidak ditemukan")
	}

	isCustomer := signal.CustomerID == accountID
	isMerchant := conv.MerchantID == accountID || s.signalRepo.VerifyBusinessAccess(accountID, role, conv.BusinessID)

	if !isCustomer && !isMerchant {
		return nil, 0, errors.New("anda tidak memiliki akses ke percakapan ini")
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	// Hanya tandai read saat membaca halaman pertama (mencegah false-positive saat scroll history)
	if page == 1 {
		_ = s.signalRepo.MarkMessagesAsRead(conversationID, accountID)
	}

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
	} else if conv.MerchantID == accountID || s.signalRepo.VerifyBusinessAccess(accountID, role, conv.BusinessID) {
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

	// 🔔 REAL-TIME: Kirim event new_message ke lawan bicara + in-app notification + multi-device sync
	go s.notifyNewMessage(accountID, conv, signal, &msg)

	return &msg, nil
}

// notifyNewMessage mengirimkan event new_message ke lawan bicara dan multi-device staff/owner
func (s *signalService) notifyNewMessage(senderID string, conv *models.SignalConversation, signal *models.Signal, msg *models.SignalMessage) {
	var recipientIDs []string
	var senderTitle string

	event := realtime.WSEvent{
		Type: "new_message",
		Data: msg,
	}

	if senderID == signal.CustomerID {
		// Customer mengirim pesan -> kirim ke merchant creator + owner toko
		recipientIDs = []string{conv.MerchantID}
		if biz, err := s.businessRepo.FindBusinessByID(conv.BusinessID); err == nil && biz != nil && biz.AccountID != "" && biz.AccountID != conv.MerchantID {
			recipientIDs = append(recipientIDs, biz.AccountID)
		}
		senderTitle = "💬 Pesan Baru dari Pelanggan"

		// Live WebSocket broadcast ke semua staff/owner toko
		s.hub.BroadcastToUsers(recipientIDs, event)
	} else {
		// Merchant (Owner/Staff) mengirim pesan -> kirim ke Customer
		recipientIDs = []string{signal.CustomerID}
		bizName := "Toko"
		if biz, err := s.businessRepo.FindBusinessByID(conv.BusinessID); err == nil && biz != nil && biz.BusinessName != nil && *biz.BusinessName != "" {
			bizName = *biz.BusinessName
		}
		senderTitle = fmt.Sprintf("💬 Pesan dari %s", bizName)

		// Live WebSocket ke customer
		s.hub.SendToUser(signal.CustomerID, event)

		// Real-time sync ke merchant lain (owner/staff) di bisnis yang sama agar screen mereka terupdate
		if conv.MerchantID != senderID {
			s.hub.SendToUser(conv.MerchantID, event)
		}
		if biz, err := s.businessRepo.FindBusinessByID(conv.BusinessID); err == nil && biz != nil && biz.AccountID != senderID && biz.AccountID != conv.MerchantID {
			s.hub.SendToUser(biz.AccountID, event)
		}
	}

	// Simpan ke in-app notification & push FCM
	if s.notifService != nil && len(recipientIDs) > 0 {
		body := msg.MessageText
		if body == "" {
			body = "Mengirim lampiran gambar"
		} else {
			body = utils.TruncateString(body, 80)
		}

		s.notifService.SendNotification(
			recipientIDs,
			senderID,
			"signal_message",
			conv.ConversationID,
			senderTitle,
			body,
			map[string]interface{}{
				"type":            "signal_message",
				"conversation_id": conv.ConversationID,
				"signal_id":       conv.SignalID,
				"message_id":      msg.MessageID,
				"sender_id":       senderID,
			},
			nil,
		)
	}
}
