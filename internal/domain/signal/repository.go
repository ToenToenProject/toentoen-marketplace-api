package signal

import (
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type SignalRepository interface {
	// Signal CRUD
	CreateSignal(signal *models.Signal) error
	UpdateSignalLocation(signalID string, lat, lng float64) error
	GetSignalByID(signalID string) (*models.Signal, error)
	GetMySignals(customerID string) ([]models.Signal, error)
	GetSignalWithConversations(signalID string) (*models.Signal, error)
	CompleteSignal(signalID string) error

	// Signal Images
	CreateSignalImages(images []models.SignalImage) error

	// Explore (Merchant) — PostGIS ST_DWithin
	ExploreActiveSignals(merchantLat, merchantLng float64, categories []string) ([]models.SignalExploreItem, error)

	// Conversations
	CreateConversation(conv *models.SignalConversation) error
	GetConversationByID(convID string) (*models.SignalConversation, error)
	GetConversationsBySignal(signalID string) ([]models.SignalConversation, error)
	FindExistingConversation(signalID, merchantID string) (*models.SignalConversation, error)
	GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error)
	GetCustomerConversations(customerAccountID string) ([]models.ConversationListItem, error)

	// Messages
	CreateMessage(msg *models.SignalMessage) error
	GetMessagesByConversation(convID string, page, limit int) ([]models.SignalMessage, int64, error)
	MarkMessagesAsRead(convID, readerID string) error

	// Access Control
	VerifyBusinessAccess(accountID, role, businessID string) bool

	// FCM & Notification Support
	GetFCMTokensByAccountID(accountID string) ([]string, error)
	GetMerchantIDsBySignal(signalID string) ([]string, error)
}

type signalRepository struct {
	db *gorm.DB
}

func NewSignalRepository(db *gorm.DB) SignalRepository {
	return &signalRepository{db}
}

// ================================================================
// SIGNAL CRUD
// ================================================================

func (r *signalRepository) CreateSignal(signal *models.Signal) error {
	return r.db.Create(signal).Error
}

// UpdateSignalLocation menggunakan raw SQL untuk mengisi kolom PostGIS geometry
func (r *signalRepository) UpdateSignalLocation(signalID string, lat, lng float64) error {
	return r.db.Exec(
		"UPDATE signals SET location = ST_SetSRID(ST_MakePoint(?, ?), 4326) WHERE signal_id = ?",
		lng, lat, signalID, // ST_MakePoint menerima (longitude, latitude)
	).Error
}

func (r *signalRepository) GetSignalByID(signalID string) (*models.Signal, error) {
	var signal models.Signal
	err := r.db.
		Preload("Account").
		Preload("Images").
		Where("signal_id = ?", signalID).
		First(&signal).Error
	if err != nil {
		return nil, err
	}
	return &signal, nil
}

func (r *signalRepository) GetMySignals(customerID string) ([]models.Signal, error) {
	var signals []models.Signal
	err := r.db.
		Preload("Images").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&signals).Error
	return signals, err
}

func (r *signalRepository) GetSignalWithConversations(signalID string) (*models.Signal, error) {
	var signal models.Signal
	err := r.db.
		Preload("Account").
		Preload("Images").
		Preload("Conversations", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Conversations.Business").
		Preload("Conversations.Merchant").
		Where("signal_id = ?", signalID).
		First(&signal).Error
	if err != nil {
		return nil, err
	}
	return &signal, nil
}

// CompleteSignal mengubah status signal ke 'completed' dan mengunci semua conversation
func (r *signalRepository) CompleteSignal(signalID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update status signal
		if err := tx.Model(&models.Signal{}).
			Where("signal_id = ?", signalID).
			Update("status", "completed").Error; err != nil {
			return err
		}

		// 2. Lock semua conversation terkait
		if err := tx.Model(&models.SignalConversation{}).
			Where("signal_id = ?", signalID).
			Update("status", "locked").Error; err != nil {
			return err
		}

		return nil
	})
}

// ================================================================
// SIGNAL IMAGES
// ================================================================

func (r *signalRepository) CreateSignalImages(images []models.SignalImage) error {
	if len(images) == 0 {
		return nil
	}
	return r.db.Create(&images).Error
}

// ================================================================
// EXPLORE (MERCHANT) — PostGIS ST_DWithin
// ================================================================

func (r *signalRepository) ExploreActiveSignals(merchantLat, merchantLng float64, categories []string) ([]models.SignalExploreItem, error) {
	var results []models.SignalExploreItem

	if len(categories) == 0 {
		return results, nil
	}

	// Buat titik merchant dalam format PostGIS
	merchantPoint := fmt.Sprintf("ST_SetSRID(ST_MakePoint(%f, %f), 4326)", merchantLng, merchantLat)

	query := fmt.Sprintf(`
		SELECT 
			s.signal_id,
			s.customer_id,
			s.description,
			s.category,
			s.is_urgent,
			s.radius_km,
			s.expires_at,
			s.created_at,
			ST_Distance(s.location::geography, %s::geography) / 1000 AS distance_km,
			TRIM(CONCAT(a.first_name, ' ', a.last_name)) AS customer_name,
			a.profile_picture_url AS customer_avatar
		FROM signals s
		JOIN accounts a ON s.customer_id = a.account_id
		WHERE s.status = 'active'
		  AND s.expires_at > NOW()
		  AND s.location IS NOT NULL
		  AND s.category IN ?
		  AND ST_DWithin(s.location::geography, %s::geography, s.radius_km * 1000)
		ORDER BY s.is_urgent DESC, s.created_at DESC
	`, merchantPoint, merchantPoint)

	err := r.db.Raw(query, categories).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Ambil images untuk setiap signal
	for i, item := range results {
		var images []models.SignalImage
		r.db.Where("signal_id = ?", item.SignalID).Find(&images)
		imgURLs := make([]string, len(images))
		for j, img := range images {
			imgURLs[j] = img.ImageURL
		}
		results[i].Images = imgURLs
	}

	return results, nil
}

// ================================================================
// CONVERSATIONS
// ================================================================

func (r *signalRepository) CreateConversation(conv *models.SignalConversation) error {
	return r.db.Create(conv).Error
}

func (r *signalRepository) GetConversationByID(convID string) (*models.SignalConversation, error) {
	var conv models.SignalConversation
	err := r.db.
		Preload("Business").
		Preload("Merchant").
		Where("conversation_id = ?", convID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *signalRepository) GetConversationsBySignal(signalID string) ([]models.SignalConversation, error) {
	var convs []models.SignalConversation
	err := r.db.
		Preload("Business").
		Preload("Merchant").
		Where("signal_id = ?", signalID).
		Order("created_at ASC").
		Find(&convs).Error
	return convs, err
}

func (r *signalRepository) FindExistingConversation(signalID, merchantID string) (*models.SignalConversation, error) {
	var conv models.SignalConversation
	err := r.db.
		Preload("Business").
		Preload("Merchant").
		Where("signal_id = ? AND merchant_id = ?", signalID, merchantID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *signalRepository) GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error) {
	var conversations []models.SignalConversation
	err := r.db.
		Preload("Business").
		Preload("Merchant").
		Where("merchant_id = ?", merchantAccountID).
		Order("created_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, err
	}

	var items []models.ConversationListItem
	for _, conv := range conversations {
		signal, errSig := r.GetSignalByID(conv.SignalID)
		sigDesc := ""
		sigCat := ""
		sigStatus := ""
		isUrgent := false
		peerID := ""
		peerName := "Customer"
		var peerAvatar *string

		if errSig == nil && signal != nil {
			sigDesc = signal.Description
			sigCat = signal.Category
			sigStatus = signal.Status
			isUrgent = signal.IsUrgent
			peerID = signal.CustomerID
			if signal.Account != nil {
				peerName = strings.TrimSpace(signal.Account.FirstName + " " + signal.Account.LastName)
				if peerName == "" {
					peerName = "Customer"
				}
				peerAvatar = signal.Account.ProfilePictureUrl
			}
		}

		bName := ""
		var bLogo *string
		if conv.Business != nil {
			if conv.Business.BusinessName != nil {
				bName = *conv.Business.BusinessName
			}
			bLogo = conv.Business.LogoURL
		}

		// Ambil pesan terakhir
		var lastMsg models.SignalMessage
		var lastMsgText *string
		var lastMsgSender string
		var lastMsgAt time.Time = conv.CreatedAt

		if err := r.db.Where("conversation_id = ?", conv.ConversationID).
			Order("created_at DESC").First(&lastMsg).Error; err == nil {
			lastMsgText = &lastMsg.MessageText
			lastMsgSender = lastMsg.SenderType
			lastMsgAt = lastMsg.CreatedAt
		}

		// Hitung unread messages (pesan dari customer yang belum dibaca merchant)
		var unreadCount int64
		r.db.Model(&models.SignalMessage{}).
			Where("conversation_id = ? AND sender_type = 'customer' AND is_read = false", conv.ConversationID).
			Count(&unreadCount)

		item := models.ConversationListItem{
			ConversationID:    conv.ConversationID,
			SignalID:          conv.SignalID,
			SignalDescription: sigDesc,
			SignalCategory:    sigCat,
			SignalStatus:      sigStatus,
			IsUrgent:          isUrgent,
			PeerID:            peerID,
			PeerName:          peerName,
			PeerAvatar:        peerAvatar,
			BusinessID:        conv.BusinessID,
			BusinessName:      bName,
			BusinessLogo:      bLogo,
			LastMessage:       lastMsgText,
			LastMessageSender: lastMsgSender,
			LastMessageAt:     lastMsgAt,
			UnreadCount:       unreadCount,
			Status:            conv.Status,
			CreatedAt:         conv.CreatedAt,
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *signalRepository) GetCustomerConversations(customerAccountID string) ([]models.ConversationListItem, error) {
	var conversations []models.SignalConversation
	err := r.db.
		Preload("Business").
		Preload("Merchant").
		Joins("JOIN signals ON signals.signal_id = signal_conversations.signal_id").
		Where("signals.customer_id = ?", customerAccountID).
		Order("signal_conversations.created_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, err
	}

	var items []models.ConversationListItem
	for _, conv := range conversations {
		signal, errSig := r.GetSignalByID(conv.SignalID)
		sigDesc := ""
		sigCat := ""
		sigStatus := ""
		isUrgent := false

		if errSig == nil && signal != nil {
			sigDesc = signal.Description
			sigCat = signal.Category
			sigStatus = signal.Status
			isUrgent = signal.IsUrgent
		}

		peerID := conv.MerchantID
		peerName := "Merchant"
		var peerAvatar *string

		if conv.Merchant != nil {
			peerName = strings.TrimSpace(conv.Merchant.FirstName + " " + conv.Merchant.LastName)
			if peerName == "" {
				peerName = "Merchant"
			}
			peerAvatar = conv.Merchant.ProfilePictureUrl
		}

		bName := ""
		var bLogo *string
		if conv.Business != nil {
			if conv.Business.BusinessName != nil {
				bName = *conv.Business.BusinessName
			}
			bLogo = conv.Business.LogoURL
			if bLogo != nil && *bLogo != "" {
				peerAvatar = bLogo
			}
		}

		// Ambil pesan terakhir
		var lastMsg models.SignalMessage
		var lastMsgText *string
		var lastMsgSender string
		var lastMsgAt time.Time = conv.CreatedAt

		if err := r.db.Where("conversation_id = ?", conv.ConversationID).
			Order("created_at DESC").First(&lastMsg).Error; err == nil {
			lastMsgText = &lastMsg.MessageText
			lastMsgSender = lastMsg.SenderType
			lastMsgAt = lastMsg.CreatedAt
		}

		// Hitung unread messages (pesan dari merchant yang belum dibaca customer)
		var unreadCount int64
		r.db.Model(&models.SignalMessage{}).
			Where("conversation_id = ? AND sender_type = 'merchant' AND is_read = false", conv.ConversationID).
			Count(&unreadCount)

		item := models.ConversationListItem{
			ConversationID:    conv.ConversationID,
			SignalID:          conv.SignalID,
			SignalDescription: sigDesc,
			SignalCategory:    sigCat,
			SignalStatus:      sigStatus,
			IsUrgent:          isUrgent,
			PeerID:            peerID,
			PeerName:          peerName,
			PeerAvatar:        peerAvatar,
			BusinessID:        conv.BusinessID,
			BusinessName:      bName,
			BusinessLogo:      bLogo,
			LastMessage:       lastMsgText,
			LastMessageSender: lastMsgSender,
			LastMessageAt:     lastMsgAt,
			UnreadCount:       unreadCount,
			Status:            conv.Status,
			CreatedAt:         conv.CreatedAt,
		}
		items = append(items, item)
	}

	return items, nil
}

// ================================================================
// MESSAGES
// ================================================================

func (r *signalRepository) CreateMessage(msg *models.SignalMessage) error {
	return r.db.Create(msg).Error
}

func (r *signalRepository) GetMessagesByConversation(convID string, page, limit int) ([]models.SignalMessage, int64, error) {
	var messages []models.SignalMessage
	var total int64

	r.db.Model(&models.SignalMessage{}).Where("conversation_id = ?", convID).Count(&total)

	offset := (page - 1) * limit
	err := r.db.
		Where("conversation_id = ?", convID).
		Order("created_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error

	return messages, total, err
}

func (r *signalRepository) MarkMessagesAsRead(convID, readerID string) error {
	return r.db.Model(&models.SignalMessage{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = ?", convID, readerID, false).
		Update("is_read", true).Error
}

// ================================================================
// ACCESS CONTROL
// ================================================================

func (r *signalRepository) VerifyBusinessAccess(accountID, role, businessID string) bool {
	var count int64
	if role == "owner" {
		r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count)
	} else if role == "business_admin" {
		r.db.Model(&models.BusinessAdminAssignment{}).Where("business_id = ? AND admin_id = ? AND is_active = ?", businessID, accountID, true).Count(&count)
	}
	return count > 0
}

// ExpireOldSignals digunakan oleh cron job untuk mengexpirasi signal yang sudah lewat waktunya
func (r *signalRepository) ExpireOldSignals() error {
	return r.db.Model(&models.Signal{}).
		Where("status = ? AND expires_at < ?", "active", time.Now()).
		Update("status", "expired").Error
}

// ================================================================
// FCM & NOTIFICATION SUPPORT
// ================================================================

// GetFCMTokensByAccountID mengambil semua token FCM milik satu user
func (r *signalRepository) GetFCMTokensByAccountID(accountID string) ([]string, error) {
	var tokens []string
	err := r.db.Table("device_tokens").
		Where("account_id = ?", accountID).
		Pluck("fcm_token", &tokens).Error
	return tokens, err
}

// GetMerchantIDsBySignal mengambil semua merchant_id yang punya conversation aktif pada signal
func (r *signalRepository) GetMerchantIDsBySignal(signalID string) ([]string, error) {
	var merchantIDs []string
	err := r.db.Model(&models.SignalConversation{}).
		Where("signal_id = ?", signalID).
		Pluck("merchant_id", &merchantIDs).Error
	return merchantIDs, err
}
