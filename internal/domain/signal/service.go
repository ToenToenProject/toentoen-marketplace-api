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
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"
)

type SignalService interface {
	// Customer
	CreateSignal(customerID string, input models.CreateSignalInput) (*models.Signal, error)
	UpdateSignal(customerID, signalID string, input models.UpdateSignalInput) (*models.Signal, error)
	DeleteSignal(customerID, signalID string) error
	CloseSignal(customerID, signalID string) error
	GetMySignals(customerID string) ([]models.Signal, error)
	GetSignalDetail(signalID, accountID, role string) (*models.Signal, error)
	CompleteSignal(customerID, signalID string) error
	GetCustomerConversations(customerID string) ([]models.ConversationListItem, error)

	// Merchant
	ExploreSignals(accountID, businessID string, lat, lng float64) ([]models.SignalExploreItem, error)
	GetMerchantSignalDetail(signalID, merchantAccountID string) (*models.MerchantSignalDetailResponse, error)
	ReplyToSignal(accountID, role, signalID string, input models.InitiateConversationInput) (*models.SignalConversation, error)
	GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error)
	GetProductSuggestions(businessID, signalID string) ([]models.Product, error)

	// Shared (Customer & Merchant)
	GetMessages(accountID, role, conversationID string, page, limit int) ([]models.SignalMessage, int64, *models.SignalPostReference, error)
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

	// 1. Kumpulkan semua category ID yang diminta
	var requestedCatIDs []int64
	if len(input.CategoryIDs) > 0 {
		seen := make(map[int64]bool)
		for _, id := range input.CategoryIDs {
			if id > 0 && !seen[id] {
				seen[id] = true
				requestedCatIDs = append(requestedCatIDs, id)
			}
		}
	} else if input.CategoryID != nil && *input.CategoryID > 0 {
		requestedCatIDs = append(requestedCatIDs, *input.CategoryID)
	}

	catInput := strings.TrimSpace(input.Category)
	if len(requestedCatIDs) == 0 && catInput == "" {
		return nil, errors.New("kategori wajib dipilih dari daftar kategori yang tersedia")
	}

	if len(requestedCatIDs) > 5 {
		return nil, errors.New("maksimal 5 kategori dapat dipilih")
	}

	// 2. Validasi ke database
	var resolvedCategories []models.Category
	if len(requestedCatIDs) > 0 {
		cats, err := s.signalRepo.GetCategoriesByIDs(requestedCatIDs)
		if err != nil || len(cats) == 0 {
			return nil, errors.New("kategori tidak valid. Silakan pilih kategori resmi yang tersedia di sistem")
		}
		catMap := make(map[int64]models.Category)
		for _, c := range cats {
			catMap[c.CategoryID] = c
		}
		for _, id := range requestedCatIDs {
			c, ok := catMap[id]
			if !ok {
				return nil, fmt.Errorf("kategori dengan ID %d tidak ditemukan di sistem", id)
			}
			resolvedCategories = append(resolvedCategories, c)
		}
	} else {
		validCategory, errCat := s.signalRepo.GetCategoryByNameOrID(catInput, 0)
		if errCat != nil || validCategory == nil {
			return nil, errors.New("kategori tidak valid. Silakan pilih kategori resmi yang tersedia di sistem")
		}
		resolvedCategories = append(resolvedCategories, *validCategory)
	}

	var resolvedCatIDs []int64
	var resolvedCatNames []string
	for _, c := range resolvedCategories {
		resolvedCatIDs = append(resolvedCatIDs, c.CategoryID)
		resolvedCatNames = append(resolvedCatNames, c.Name)
	}

	primaryCatID := resolvedCatIDs[0]
	primaryCatName := resolvedCatNames[0]

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

	maxOffers := 5
	if input.MaxOffers != nil && *input.MaxOffers > 0 {
		maxOffers = *input.MaxOffers
	}

	signal := models.Signal{
		SignalID:    utils.GenerateUUID("SIG"),
		CustomerID:  customerID,
		Description: desc,
		CategoryID:  &primaryCatID,
		Category:    primaryCatName,
		CategoryIDs: resolvedCatIDs,
		Categories:  resolvedCatNames,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		RadiusKm:    radiusKm,
		IsUrgent:    input.IsUrgent,
		Status:      "active",
		MinBudget:   input.MinBudget,
		MaxBudget:   input.MaxBudget,
		MaxOffers:   &maxOffers,
		OfferCount:  0,
		ExpiresAt:   time.Now().Add(time.Duration(expiresInHours) * time.Hour),
	}

	// 1. Simpan signal ke database
	if err := s.signalRepo.CreateSignal(&signal); err != nil {
		return nil, fmt.Errorf("gagal membuat signal: %w", err)
	}

	// 2. Simpan relasi multi-kategori ke database
	var signalCats []models.SignalCategory
	for _, c := range resolvedCategories {
		signalCats = append(signalCats, models.SignalCategory{
			SignalID:   signal.SignalID,
			CategoryID: c.CategoryID,
		})
	}
	if err := s.signalRepo.CreateSignalCategories(signalCats); err != nil {
		fmt.Printf("Warning: gagal menyimpan signal categories: %v\n", err)
	}
	signal.SignalCategories = signalCats

	// 3. Update kolom PostGIS location via raw SQL
	if err := s.signalRepo.UpdateSignalLocation(signal.SignalID, input.Latitude, input.Longitude); err != nil {
		fmt.Printf("Warning: gagal update lokasi PostGIS untuk signal %s: %v\n", signal.SignalID, err)
	}

	// 4. Simpan gambar lampiran (jika ada, dukung image_urls, media_urls, atau images)
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

	// 5. Simpan tags yang dipilih customer (jika ada) — digunakan sebagai booster relevansi
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

	// 6. 🔔 REAL-TIME BROADCAST: Kirim notifikasi instan ke toko terdekat yang relevan
	go s.notifyNearbyMerchantsNewSignal(signal.SignalID, resolvedCatIDs, resolvedCatNames, input.Latitude, input.Longitude, float64(radiusKm), signal.IsUrgent)

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
	if len(input.CategoryIDs) > 0 || (input.CategoryID != nil && *input.CategoryID > 0) || strings.TrimSpace(input.Category) != "" {
		var requestedCatIDs []int64
		if len(input.CategoryIDs) > 0 {
			seen := make(map[int64]bool)
			for _, id := range input.CategoryIDs {
				if id > 0 && !seen[id] {
					seen[id] = true
					requestedCatIDs = append(requestedCatIDs, id)
				}
			}
		} else if input.CategoryID != nil && *input.CategoryID > 0 {
			requestedCatIDs = append(requestedCatIDs, *input.CategoryID)
		}

		if len(requestedCatIDs) > 5 {
			return nil, errors.New("maksimal 5 kategori dapat dipilih")
		}

		var resolvedCategories []models.Category
		if len(requestedCatIDs) > 0 {
			cats, err := s.signalRepo.GetCategoriesByIDs(requestedCatIDs)
			if err != nil || len(cats) == 0 {
				return nil, errors.New("kategori tidak valid. Silakan pilih kategori resmi yang tersedia di sistem")
			}
			catMap := make(map[int64]models.Category)
			for _, c := range cats {
				catMap[c.CategoryID] = c
			}
			for _, id := range requestedCatIDs {
				c, ok := catMap[id]
				if !ok {
					return nil, fmt.Errorf("kategori dengan ID %d tidak ditemukan di sistem", id)
				}
				resolvedCategories = append(resolvedCategories, c)
			}
		} else {
			catInput := strings.TrimSpace(input.Category)
			validCategory, errCat := s.signalRepo.GetCategoryByNameOrID(catInput, 0)
			if errCat != nil || validCategory == nil {
				return nil, errors.New("kategori tidak valid. Silakan pilih kategori resmi yang tersedia di sistem")
			}
			resolvedCategories = append(resolvedCategories, *validCategory)
		}

		primaryCatID := resolvedCategories[0].CategoryID
		primaryCatName := resolvedCategories[0].Name
		signal.CategoryID = &primaryCatID
		signal.Category = primaryCatName

		var signalCats []models.SignalCategory
		for _, c := range resolvedCategories {
			signalCats = append(signalCats, models.SignalCategory{
				SignalID:   signal.SignalID,
				CategoryID: c.CategoryID,
			})
		}
		if err := s.signalRepo.ReplaceSignalCategories(signal.SignalID, signalCats); err != nil {
			fmt.Printf("Warning: gagal update signal categories: %v\n", err)
		}
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

	// 7. Budget & MaxOffers jika diubah
	if input.MinBudget != nil {
		signal.MinBudget = input.MinBudget
	}
	if input.MaxBudget != nil {
		signal.MaxBudget = input.MaxBudget
	}
	if input.MaxOffers != nil && *input.MaxOffers > 0 {
		signal.MaxOffers = input.MaxOffers
	}

	// 8. Simpan perubahan signal
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
	return s.CloseSignal(customerID, signalID)
}

func (s *signalService) CloseSignal(customerID, signalID string) error {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return errors.New("signal tidak ditemukan")
	}

	if signal.CustomerID != customerID {
		return errors.New("anda bukan pemilik signal ini")
	}

	if signal.Status == "completed" {
		return errors.New("signal telah selesai dan tidak dapat ditutup/dibatalkan")
	}

	if signal.Status == "closed" {
		return nil
	}

	if err := s.signalRepo.CloseSignal(signalID); err != nil {
		return fmt.Errorf("gagal menutup signal: %w", err)
	}

	return nil
}

func (s *signalService) GetMySignals(customerID string) ([]models.Signal, error) {
	signals, err := s.signalRepo.GetMySignals(customerID)
	if err != nil {
		return nil, err
	}

	var expiredIDs []string
	for i := range signals {
		wasActive := (signals[i].Status == "active")
		signals[i].EvaluateStatus()
		if wasActive && signals[i].Status == "expired" {
			expiredIDs = append(expiredIDs, signals[i].SignalID)
		}
	}

	if len(expiredIDs) > 0 {
		go func(ids []string) {
			_ = s.signalRepo.BatchExpireSignals(ids)
		}(expiredIDs)
	}

	return signals, nil
}

func (s *signalService) GetCustomerConversations(customerID string) ([]models.ConversationListItem, error) {
	items, err := s.signalRepo.GetCustomerConversations(customerID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].LastMessage, _ = crypto.DecryptPointer(items[i].LastMessage)
	}
	return items, nil
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

	wasActive := (signal.Status == "active")
	signal.EvaluateStatus()
	if wasActive && signal.Status == "expired" {
		go func(id string) {
			_ = s.signalRepo.ExpireSignal(id)
		}(signal.SignalID)
	}

	// Dekripsi pesan pada relasi percakapan
	for i := range signal.Conversations {
		for j := range signal.Conversations[i].Messages {
			signal.Conversations[i].Messages[j].MessageText, _ = crypto.Decrypt(signal.Conversations[i].Messages[j].MessageText)
			signal.Conversations[i].Messages[j].AttachmentURL, _ = crypto.DecryptPointer(signal.Conversations[i].Messages[j].AttachmentURL)
		}
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

// notifyNearbyMerchantsNewSignal menyebarkan event dan push notifikasi ke seluruh merchant terdekat yang relevan (semua kategori yang cocok)
func (s *signalService) notifyNearbyMerchantsNewSignal(signalID string, categoryIDs []int64, categoryNames []string, lat, lng, radiusKm float64, isUrgent bool) {
	accountIDs, err := s.signalRepo.FindNearbyMatchingMerchantAccountIDs(categoryIDs, lat, lng, radiusKm)
	if err != nil || len(accountIDs) == 0 {
		return
	}

	catLabel := strings.Join(categoryNames, ", ")
	if catLabel == "" {
		catLabel = "kebutuhan"
	}

	primaryCatID := int64(0)
	primaryCatName := ""
	if len(categoryIDs) > 0 {
		primaryCatID = categoryIDs[0]
	}
	if len(categoryNames) > 0 {
		primaryCatName = categoryNames[0]
	}

	title := "📢 Peluang Baru: Permintaan Kebutuhan Sekitarmu!"
	if isUrgent {
		title = "🚨 Mendesak: Calon Pembeli Butuh Cepat di Dekatmu!"
	}
	body := fmt.Sprintf("Ada calon pembeli mencari kebutuhan kategori %s dalam radius %.0f km dari tokomu.", catLabel, radiusKm)

	event := realtime.WSEvent{
		Type: "NEW_DEMAND_SIGNAL",
		Data: map[string]interface{}{
			"type":         "signal",
			"signal_id":    signalID,
			"post_id":      signalID,
			"category_id":  primaryCatID,
			"category":     primaryCatName,
			"category_ids": categoryIDs,
			"categories":   categoryNames,
			"is_urgent":    isUrgent,
			"radius_km":    radiusKm,
		},
	}

	// Broadcast ke WebSocket merchant yang sedang online
	s.hub.BroadcastToUsers(accountIDs, event)

	// Kirim In-App Notification & Push Notification FCM
	if s.notifService != nil {
		s.notifService.SendNotification(
			accountIDs,
			"SYSTEM",
			"signal",
			signalID,
			title,
			body,
			map[string]interface{}{
				"type":         "signal",
				"signal_id":    signalID,
				"post_id":      signalID,
				"category":     primaryCatName,
				"category_ids": categoryIDs,
				"categories":   categoryNames,
				"is_urgent":    isUrgent,
			},
			nil,
		)
	}
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
			"signal",
			signalID,
			"✅ Sinyal Kebutuhan Selesai",
			"Customer telah menyelesaikan sinyal kebutuhan. Percakapan telah ditutup.",
			map[string]interface{}{
				"type":      "signal",
				"signal_id": signalID,
				"post_id":   signalID,
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
		return nil, errors.New("signal tidak ditemukan")
	}

	wasActive := (signal.Status == "active")
	signal.EvaluateStatus()
	if wasActive && signal.Status == "expired" {
		go func(id string) {
			_ = s.signalRepo.ExpireSignal(id)
		}(signal.SignalID)
	}

	existing, _ := s.signalRepo.FindExistingConversation(signalID, merchantAccountID)
	hasReplied := existing != nil
	var myConvID *string
	var myConv *models.SignalConversation

	if hasReplied {
		myConvID = &existing.ConversationID
		myConv, err = s.signalRepo.GetConversationByID(existing.ConversationID)
		if err == nil && myConv != nil {
			for i := range myConv.Messages {
				myConv.Messages[i].MessageText, _ = crypto.Decrypt(myConv.Messages[i].MessageText)
				myConv.Messages[i].AttachmentURL, _ = crypto.DecryptPointer(myConv.Messages[i].AttachmentURL)
			}
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
	items, err := s.signalRepo.GetMerchantConversations(merchantAccountID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].LastMessage, _ = crypto.DecryptPointer(items[i].LastMessage)
	}
	return items, nil
}

func (s *signalService) ReplyToSignal(accountID, role, signalID string, input models.InitiateConversationInput) (*models.SignalConversation, error) {
	if !s.signalRepo.VerifyBusinessAccess(accountID, role, input.BusinessID) {
		return nil, errors.New("anda tidak memiliki akses ke bisnis ini")
	}

	if role == "business_admin" && !permissions.HasPermission(database.DB, accountID, input.BusinessID, models.PermManageDemandSignals) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk merespon demand signal")
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

	// Validasi Kuota Penawaran (Offer Cap)
	if signal.MaxOffers != nil && *signal.MaxOffers > 0 && signal.OfferCount >= *signal.MaxOffers {
		return nil, errors.New("batas maksimal kuota penawaran untuk sinyal ini telah tercapai")
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

	// Increment offer count
	_ = s.signalRepo.IncrementOfferCount(signalID)

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

		msgToSave := msg
		msgToSave.MessageText, _ = crypto.Encrypt(msg.MessageText)
		msgToSave.AttachmentURL, _ = crypto.EncryptPointer(msg.AttachmentURL)

		if err := s.signalRepo.CreateMessage(&msgToSave); err != nil {
			fmt.Printf("Warning: gagal menyimpan pesan pertama: %v\n", err)
		}
		firstMsg = &msg
	}

	// 🔔 REAL-TIME: Beritahu customer bahwa merchant telah membalas
	go s.notifyNewConversation(signal.CustomerID, accountID, &conv, firstMsg)

	// 🔔 Notifikasi ke Owner jika yang merespons sinyal adalah staf admin
	if role == "business_admin" {
		notification.NotifyOwnerStaffActivity(
			database.DB,
			s.notifService,
			input.BusinessID,
			accountID,
			"respond_demand_signal",
			"🎯 Staf Merespons Kebutuhan Pasar",
			fmt.Sprintf("Staf toko Anda telah menanggapi permintaan kebutuhan pembeli: %s", utils.TruncateString(signal.Description, 50)),
			map[string]interface{}{
				"conversation_id": conv.ConversationID,
				"signal_id":       signalID,
				"action":          "monitor_negotiation",
			},
		)
	}

	return &conv, nil
}

// GetProductSuggestions mencari produk dari katalog toko yang paling cocok dengan sinyal kebutuhan
func (s *signalService) GetProductSuggestions(businessID, signalID string) ([]models.Product, error) {
	signal, err := s.signalRepo.GetSignalByID(signalID)
	if err != nil {
		return nil, errors.New("signal tidak ditemukan")
	}

	var tagIDs []int64
	for _, t := range signal.Tags {
		tagIDs = append(tagIDs, t.TagID)
	}

	products, err := s.signalRepo.FindMatchingProductsForSignal(businessID, signal.Description, tagIDs)
	if err != nil {
		return nil, err
	}

	return products, nil
}

// notifyNewConversation mengirimkan event new_conversation ke customer pemilik signal
func (s *signalService) notifyNewConversation(customerID, merchantAccountID string, conv *models.SignalConversation, firstMsg *models.SignalMessage) {
	event := realtime.WSEvent{
		Type: "new_conversation",
		Data: map[string]interface{}{
			"type":            "signal_offer",
			"conversation_id": conv.ConversationID,
			"offer_id":        conv.ConversationID,
			"signal_id":       conv.SignalID,
			"post_id":         conv.SignalID,
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
			"signal_offer",
			conv.ConversationID,
			"💬 Ada Tanggapan Baru!",
			body,
			map[string]interface{}{
				"type":            "signal_offer",
				"conversation_id": conv.ConversationID,
				"offer_id":        conv.ConversationID,
				"signal_id":       conv.SignalID,
				"post_id":         conv.SignalID,
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

func (s *signalService) GetMessages(accountID, role, conversationID string, page, limit int) ([]models.SignalMessage, int64, *models.SignalPostReference, error) {
	conv, err := s.signalRepo.GetConversationByID(conversationID)
	if err != nil {
		return nil, 0, nil, errors.New("percakapan tidak ditemukan")
	}

	signal, err := s.signalRepo.GetSignalByID(conv.SignalID)
	if err != nil {
		return nil, 0, nil, errors.New("signal tidak ditemukan")
	}

	isCustomer := signal.CustomerID == accountID
	isMerchant := conv.MerchantID == accountID || s.signalRepo.VerifyBusinessAccess(accountID, role, conv.BusinessID)

	if !isCustomer && !isMerchant {
		return nil, 0, nil, errors.New("anda tidak memiliki akses ke percakapan ini")
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

	messages, total, err := s.signalRepo.GetMessagesByConversation(conversationID, page, limit)
	if err != nil {
		return nil, 0, nil, err
	}

	for i := range messages {
		messages[i].MessageText, _ = crypto.Decrypt(messages[i].MessageText)
		messages[i].AttachmentURL, _ = crypto.DecryptPointer(messages[i].AttachmentURL)
	}

	signalRef := signal.ToPostReference()

	return messages, total, signalRef, nil
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
		if role == "business_admin" && !permissions.HasPermission(database.DB, accountID, conv.BusinessID, models.PermManageDemandSignals) {
			return nil, errors.New("forbidden: Anda tidak memiliki izin untuk merespon demand signal")
		}
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

	msgToSave := msg
	msgToSave.MessageText, _ = crypto.Encrypt(msg.MessageText)
	msgToSave.AttachmentURL, _ = crypto.EncryptPointer(msg.AttachmentURL)

	if err := s.signalRepo.CreateMessage(&msgToSave); err != nil {
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
				"offer_id":        conv.ConversationID,
				"signal_id":       conv.SignalID,
				"post_id":         conv.SignalID,
				"message_id":      msg.MessageID,
				"sender_id":       senderID,
			},
			nil,
		)
	}
}
