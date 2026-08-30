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
	UpdateSignal(signal *models.Signal) error
	DeleteSignal(signalID string) error
	UpdateSignalLocation(signalID string, lat, lng float64) error
	GetSignalByID(signalID string) (*models.Signal, error)
	GetMySignals(customerID string) ([]models.Signal, error)
	GetSignalWithConversations(signalID string) (*models.Signal, error)
	CompleteSignal(signalID string) error

	// Signal Images
	CreateSignalImages(images []models.SignalImage) error
	ReplaceSignalImages(signalID string, images []models.SignalImage) error

	// Signal Tags (booster relevansi)
	CreateSignalTags(tags []models.SignalTag) error
	ReplaceSignalTags(signalID string, tags []models.SignalTag) error

	// Explore (Merchant) — Category-strict + Tag relevance ranking
	ExploreActiveSignals(merchantBusinessID string, merchantLat, merchantLng float64) ([]models.SignalExploreItem, error)

	// Conversations
	CreateConversation(conv *models.SignalConversation) error
	GetConversationByID(convID string) (*models.SignalConversation, error)
	GetConversationsBySignal(signalID string) ([]models.SignalConversation, error)
	FindExistingConversation(signalID, merchantID string) (*models.SignalConversation, error)
	FindExistingConversationByBusiness(signalID, businessID string) (*models.SignalConversation, error)
	GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error)
	GetCustomerConversations(customerAccountID string) ([]models.ConversationListItem, error)

	// Messages
	CreateMessage(msg *models.SignalMessage) error
	GetMessagesByConversation(convID string, page, limit int) ([]models.SignalMessage, int64, error)
	MarkMessagesAsRead(convID, readerID string) error

	// Category Validation
	GetCategoryByNameOrID(categoryName string, categoryID int64) (*models.Category, error)

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

func (r *signalRepository) UpdateSignal(signal *models.Signal) error {
	return r.db.Save(signal).Error
}

func (r *signalRepository) DeleteSignal(signalID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Hapus signal_tags & signal_images
		tx.Where("signal_id = ?", signalID).Delete(&models.SignalTag{})
		tx.Where("signal_id = ?", signalID).Delete(&models.SignalImage{})

		// Hapus signal (conversations & messages akan otomatis ter-cascade jika constraint aktif, atau terhapus)
		return tx.Where("signal_id = ?", signalID).Delete(&models.Signal{}).Error
	})
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
		Preload("CategoryObj").
		Preload("Tags.Tag").
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
		Preload("CategoryObj").
		Preload("Tags.Tag").
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
		Preload("CategoryObj").
		Preload("Tags.Tag").
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

func (r *signalRepository) ReplaceSignalImages(signalID string, images []models.SignalImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("signal_id = ?", signalID).Delete(&models.SignalImage{}).Error; err != nil {
			return err
		}
		if len(images) > 0 {
			return tx.Create(&images).Error
		}
		return nil
	})
}

// ================================================================
// SIGNAL TAGS
// ================================================================

func (r *signalRepository) CreateSignalTags(tags []models.SignalTag) error {
	if len(tags) == 0 {
		return nil
	}
	return r.db.Create(&tags).Error
}

func (r *signalRepository) ReplaceSignalTags(signalID string, tags []models.SignalTag) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("signal_id = ?", signalID).Delete(&models.SignalTag{}).Error; err != nil {
			return err
		}
		if len(tags) > 0 {
			return tx.Create(&tags).Error
		}
		return nil
	})
}

// ================================================================
// EXPLORE (MERCHANT) — Category-Strict + Tag Relevance Ranking
// ================================================================

// ExploreActiveSignals mengambil sinyal aktif yang tepat sasaran untuk merchant.
//
// Logika multi-layer:
//  1. WAJIB: Signal harus memiliki category_id yang sama dengan salah satu kategori bisnis merchant
//     → Mencegah signal restoran masuk ke merchant bengkel meski punya tag sama
//  2. LOKASI: Signal harus dalam radius yang ditentukan customer (PostGIS ST_DWithin)
//  3. RANKING: Signal diurutkan berdasarkan:
//     a. is_urgent DESC  — sinyal mendesak paling atas
//     b. tag_overlap DESC — sinyal dengan tag bisnis paling banyak cocok naik peringkat
//     c. created_at DESC — terbaru lebih atas
func (r *signalRepository) ExploreActiveSignals(merchantBusinessID string, merchantLat, merchantLng float64) ([]models.SignalExploreItem, error) {
	var results []models.SignalExploreItem

	merchantPoint := fmt.Sprintf("ST_SetSRID(ST_MakePoint(%f, %f), 4326)", merchantLng, merchantLat)

	query := fmt.Sprintf(`
		SELECT
			s.signal_id,
			s.customer_id,
			s.description,
			s.category_id,
			s.category,
			s.is_urgent,
			s.radius_km,
			s.expires_at,
			s.created_at,
			ST_Distance(s.location::geography, %s::geography) / 1000 AS distance_km,
			TRIM(CONCAT(a.first_name, ' ', a.last_name)) AS customer_name,
			a.profile_picture_url AS customer_avatar,
			-- Layer 3: Hitung berapa tag signal yang cocok dengan tag bisnis merchant
			COALESCE((
				SELECT COUNT(*)
				FROM signal_tags st
				JOIN business_has_tags bht ON st.tag_id = bht.tag_id
				WHERE st.signal_id = s.signal_id
				  AND bht.business_id = ?
			), 0) AS tag_overlap_score
		FROM signals s
		JOIN accounts a ON s.customer_id = a.account_id
		WHERE s.status = 'active'
		  AND s.expires_at > NOW()
		  AND s.location IS NOT NULL
		  -- Layer 1 (WAJIB): Kategori signal HARUS cocok dengan kategori bisnis merchant
		  AND s.category_id IN (
		      SELECT category_id FROM business_has_categories
		      WHERE business_id = ?
		  )
		  -- Layer 2 (WAJIB): Signal dalam radius jangkauan yang ditentukan customer
		  AND ST_DWithin(s.location::geography, %s::geography, s.radius_km * 1000)
		ORDER BY
			s.is_urgent DESC,
			tag_overlap_score DESC,
			s.created_at DESC
	`, merchantPoint, merchantPoint)

	err := r.db.Raw(query, merchantBusinessID, merchantBusinessID).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	// Kumpulkan semua signalID untuk batch query (menghilangkan N+1 query)
	signalIDs := make([]string, len(results))
	for i, item := range results {
		signalIDs[i] = item.SignalID
		results[i].Images = []string{}
		results[i].Tags = []string{}
	}

	// 1. Batch query semua gambar signal dalam 1 query
	var allImages []models.SignalImage
	if err := r.db.Where("signal_id IN ?", signalIDs).Find(&allImages).Error; err == nil {
		imgMap := make(map[string][]string)
		for _, img := range allImages {
			imgMap[img.SignalID] = append(imgMap[img.SignalID], img.ImageURL)
		}
		for i, item := range results {
			if urls, ok := imgMap[item.SignalID]; ok {
				results[i].Images = urls
			}
		}
	}

	// 2. Batch query semua tag signal dalam 1 query
	type tagRow struct {
		SignalID string `gorm:"column:signal_id"`
		TagName  string `gorm:"column:name"`
	}
	var allTags []tagRow
	if err := r.db.Raw(`
		SELECT st.signal_id, bt.name
		FROM signal_tags st
		JOIN business_tags bt ON st.tag_id = bt.tag_id
		WHERE st.signal_id IN ?
	`, signalIDs).Scan(&allTags).Error; err == nil {
		tagMap := make(map[string][]string)
		for _, t := range allTags {
			tagMap[t.SignalID] = append(tagMap[t.SignalID], t.TagName)
		}
		for i, item := range results {
			if names, ok := tagMap[item.SignalID]; ok {
				results[i].Tags = names
			}
		}
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

func (r *signalRepository) FindExistingConversationByBusiness(signalID, businessID string) (*models.SignalConversation, error) {
	var conv models.SignalConversation
	err := r.db.
		Preload("Business").
		Preload("Merchant").
		Where("signal_id = ? AND business_id = ?", signalID, businessID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetMerchantConversations mengambil inbox merchant/toko dengan 1 query optimal (LATERAL JOIN)
// Terurut berdasarkan waktu pesan terakhir (last_message_at DESC) dan mendukung multi-role
func (r *signalRepository) GetMerchantConversations(merchantAccountID string) ([]models.ConversationListItem, error) {
	var items []models.ConversationListItem

	query := `
		SELECT 
			sc.conversation_id,
			sc.signal_id,
			sc.merchant_id,
			sc.business_id,
			sc.status,
			sc.created_at,
			s.description AS signal_description,
			s.category_id,
			s.category AS signal_category,
			s.status AS signal_status,
			s.is_urgent,
			s.customer_id AS peer_id,
			TRIM(CONCAT(ca.first_name, ' ', ca.last_name)) AS peer_name,
			ca.profile_picture_url AS peer_avatar,
			COALESCE(b.business_name, '') AS business_name,
			b.logo_url AS business_logo,
			lm.message_text AS last_message,
			lm.sender_type AS last_message_sender,
			COALESCE(lm.created_at, sc.created_at) AS last_message_at,
			COALESCE(unread.cnt, 0) AS unread_count
		FROM signal_conversations sc
		JOIN signals s ON sc.signal_id = s.signal_id
		LEFT JOIN accounts ca ON s.customer_id = ca.account_id
		LEFT JOIN businesses b ON sc.business_id = b.business_id
		LEFT JOIN LATERAL (
			SELECT message_text, sender_type, created_at
			FROM signal_messages
			WHERE conversation_id = sc.conversation_id
			ORDER BY created_at DESC
			LIMIT 1
		) lm ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt
			FROM signal_messages
			WHERE conversation_id = sc.conversation_id
			  AND sender_type = 'customer'
			  AND is_read = false
		) unread ON true
		WHERE sc.merchant_id = ?
		   OR sc.business_id IN (SELECT business_id FROM businesses WHERE account_id = ?)
		   OR sc.business_id IN (SELECT business_id FROM business_admin_assignments WHERE admin_id = ? AND is_active = true)
		ORDER BY last_message_at DESC
	`

	err := r.db.Raw(query, merchantAccountID, merchantAccountID, merchantAccountID).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetCustomerConversations mengambil inbox customer dengan 1 query optimal (LATERAL JOIN)
// Terurut berdasarkan waktu pesan terakhir (last_message_at DESC)
func (r *signalRepository) GetCustomerConversations(customerAccountID string) ([]models.ConversationListItem, error) {
	var items []models.ConversationListItem

	query := `
		SELECT 
			sc.conversation_id,
			sc.signal_id,
			sc.merchant_id,
			sc.business_id,
			sc.status,
			sc.created_at,
			s.description AS signal_description,
			s.category_id,
			s.category AS signal_category,
			s.status AS signal_status,
			s.is_urgent,
			sc.merchant_id AS peer_id,
			COALESCE(b.business_name, TRIM(CONCAT(ma.first_name, ' ', ma.last_name)), 'Merchant') AS peer_name,
			COALESCE(b.logo_url, ma.profile_picture_url) AS peer_avatar,
			COALESCE(b.business_name, '') AS business_name,
			b.logo_url AS business_logo,
			lm.message_text AS last_message,
			lm.sender_type AS last_message_sender,
			COALESCE(lm.created_at, sc.created_at) AS last_message_at,
			COALESCE(unread.cnt, 0) AS unread_count
		FROM signal_conversations sc
		JOIN signals s ON sc.signal_id = s.signal_id
		LEFT JOIN accounts ma ON sc.merchant_id = ma.account_id
		LEFT JOIN businesses b ON sc.business_id = b.business_id
		LEFT JOIN LATERAL (
			SELECT message_text, sender_type, created_at
			FROM signal_messages
			WHERE conversation_id = sc.conversation_id
			ORDER BY created_at DESC
			LIMIT 1
		) lm ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt
			FROM signal_messages
			WHERE conversation_id = sc.conversation_id
			  AND sender_type = 'merchant'
			  AND is_read = false
		) unread ON true
		WHERE s.customer_id = ?
		ORDER BY last_message_at DESC
	`

	err := r.db.Raw(query, customerAccountID).Scan(&items).Error
	if err != nil {
		return nil, err
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
	switch role {
	case "owner":
		r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count)
	case "business_admin":
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

// ================================================================
// CATEGORY VALIDATION
// ================================================================

// GetCategoryByNameOrID mencari kategori resmi di tabel business_categories berdasarkan nama atau category_id
func (r *signalRepository) GetCategoryByNameOrID(categoryName string, categoryID int64) (*models.Category, error) {
	var category models.Category

	// 1. Coba cari berdasarkan category_id numerik jika disediakan
	if categoryID > 0 {
		if err := r.db.Where("category_id = ?", categoryID).First(&category).Error; err == nil {
			return &category, nil
		}
	}

	// 2. Coba cari berdasarkan nama kategori (case-insensitive & trim whitespace) atau ID string
	cleanName := strings.TrimSpace(categoryName)
	if cleanName != "" {
		if err := r.db.Where("LOWER(TRIM(name)) = LOWER(TRIM(?)) OR CAST(category_id AS TEXT) = ?", cleanName, cleanName).
			First(&category).Error; err == nil {
			return &category, nil
		}
	}

	return nil, fmt.Errorf("kategori '%s' tidak ditemukan di sistem", categoryName)
}
