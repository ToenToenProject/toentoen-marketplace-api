package signal

import (
	"fmt"
	"strings"
	"sync"
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
	CloseSignal(signalID string) error
	ExpireSignal(signalID string) error
	BatchExpireSignals(signalIDs []string) error
	ExpireOverdueSignals() (int64, error)

	// Signal Images
	CreateSignalImages(images []models.SignalImage) error
	ReplaceSignalImages(signalID string, images []models.SignalImage) error

	// Signal Categories (multi-category)
	CreateSignalCategories(categories []models.SignalCategory) error
	ReplaceSignalCategories(signalID string, categories []models.SignalCategory) error

	// Signal Tags (booster relevansi)
	CreateSignalTags(tags []models.SignalTag) error
	ReplaceSignalTags(signalID string, tags []models.SignalTag) error

	// Explore (Merchant) — Category-strict + Tag relevance ranking
	ExploreActiveSignals(merchantBusinessID string, merchantLat, merchantLng float64) ([]models.SignalExploreItem, error)
	FindNearbyMatchingMerchantAccountIDs(categoryIDs []int64, lat, lng float64, radiusKm float64) ([]string, error)
	FindMatchingProductsForSignal(businessID string, queryText string, tagIDs []int64) ([]models.Product, error)
	IncrementOfferCount(signalID string) error

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
	GetCategoriesByIDs(categoryIDs []int64) ([]models.Category, error)

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

var (
	hasSignalCategoriesMu        sync.RWMutex
	hasSignalCategoriesTable     bool
	hasSignalCategoriesLastCheck time.Time
)

func hasSignalCategories(db *gorm.DB) bool {
	hasSignalCategoriesMu.RLock()
	if time.Since(hasSignalCategoriesLastCheck) < 30*time.Second {
		val := hasSignalCategoriesTable
		hasSignalCategoriesMu.RUnlock()
		return val
	}
	hasSignalCategoriesMu.RUnlock()

	hasSignalCategoriesMu.Lock()
	defer hasSignalCategoriesMu.Unlock()
	if time.Since(hasSignalCategoriesLastCheck) < 30*time.Second {
		return hasSignalCategoriesTable
	}

	var exists bool
	err := db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'signal_categories')").Scan(&exists).Error
	hasSignalCategoriesTable = (err == nil && exists)
	hasSignalCategoriesLastCheck = time.Now()
	return hasSignalCategoriesTable
}

func (r *signalRepository) GetSignalByID(signalID string) (*models.Signal, error) {
	var signal models.Signal
	q := r.db.
		Preload("Account").
		Preload("CategoryObj").
		Preload("Tags.Tag").
		Preload("Images")
	if hasSignalCategories(r.db) {
		q = q.Preload("SignalCategories.Category")
	}
	err := q.Where("signal_id = ?", signalID).First(&signal).Error
	if err != nil {
		return nil, err
	}
	signal.PopulateCategories()
	return &signal, nil
}

func (r *signalRepository) GetMySignals(customerID string) ([]models.Signal, error) {
	var signals []models.Signal
	q := r.db.
		Preload("CategoryObj").
		Preload("Tags.Tag").
		Preload("Images")
	if hasSignalCategories(r.db) {
		q = q.Preload("SignalCategories.Category")
	}
	err := q.Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&signals).Error
	if err != nil {
		return nil, err
	}
	for i := range signals {
		signals[i].PopulateCategories()
	}
	return signals, nil
}

func (r *signalRepository) GetSignalWithConversations(signalID string) (*models.Signal, error) {
	var signal models.Signal
	q := r.db.
		Preload("Account").
		Preload("CategoryObj").
		Preload("Tags.Tag").
		Preload("Images").
		Preload("Conversations", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Conversations.Business").
		Preload("Conversations.Merchant")
	if hasSignalCategories(r.db) {
		q = q.Preload("SignalCategories.Category")
	}
	err := q.Where("signal_id = ?", signalID).First(&signal).Error
	if err != nil {
		return nil, err
	}
	signal.PopulateCategories()
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

// CloseSignal mengubah status signal ke 'closed' dan mengunci semua conversation terkait
func (r *signalRepository) CloseSignal(signalID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Signal{}).
			Where("signal_id = ?", signalID).
			Update("status", "closed").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.SignalConversation{}).
			Where("signal_id = ?", signalID).
			Update("status", "locked").Error; err != nil {
			return err
		}

		return nil
	})
}

// ExpireSignal mengubah status signal ke 'expired' jika masih 'active' dan mengunci semua conversation terkait
func (r *signalRepository) ExpireSignal(signalID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Signal{}).
			Where("signal_id = ? AND status = ?", signalID, "active").
			Update("status", "expired").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.SignalConversation{}).
			Where("signal_id = ?", signalID).
			Update("status", "locked").Error; err != nil {
			return err
		}

		return nil
	})
}

// BatchExpireSignals mengubah status sekumpulan signal ke 'expired' jika masih 'active'
func (r *signalRepository) BatchExpireSignals(signalIDs []string) error {
	if len(signalIDs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Signal{}).
			Where("signal_id IN ? AND status = ?", signalIDs, "active").
			Update("status", "expired").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.SignalConversation{}).
			Where("signal_id IN ?", signalIDs).
			Update("status", "locked").Error; err != nil {
			return err
		}

		return nil
	})
}

// ExpireOverdueSignals mencari semua signal dengan status 'active' dan expires_at <= NOW(),
// lalu mengupdate statusnya menjadi 'expired' dan mengunci (locked) conversation terkait.
func (r *signalRepository) ExpireOverdueSignals() (int64, error) {
	var expiredCount int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var overdueIDs []string
		if err := tx.Model(&models.Signal{}).
			Where("status = ? AND expires_at <= NOW()", "active").
			Pluck("signal_id", &overdueIDs).Error; err != nil {
			return err
		}

		if len(overdueIDs) == 0 {
			return nil
		}

		res := tx.Model(&models.Signal{}).
			Where("signal_id IN ?", overdueIDs).
			Update("status", "expired")
		if res.Error != nil {
			return res.Error
		}
		expiredCount = res.RowsAffected

		if err := tx.Model(&models.SignalConversation{}).
			Where("signal_id IN ?", overdueIDs).
			Update("status", "locked").Error; err != nil {
			return err
		}

		return nil
	})
	return expiredCount, err
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
// SIGNAL CATEGORIES (MULTI-CATEGORY)
// ================================================================

func (r *signalRepository) CreateSignalCategories(categories []models.SignalCategory) error {
	if len(categories) == 0 || !hasSignalCategories(r.db) {
		return nil
	}
	return r.db.Create(&categories).Error
}

func (r *signalRepository) ReplaceSignalCategories(signalID string, categories []models.SignalCategory) error {
	if !hasSignalCategories(r.db) {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("signal_id = ?", signalID).Delete(&models.SignalCategory{}).Error; err != nil {
			return err
		}
		if len(categories) > 0 {
			return tx.Create(&categories).Error
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
	categoryClause := "AND s.category_id IN (SELECT category_id FROM merchant_cats)"
	if hasSignalCategories(r.db) {
		categoryClause = `AND (
		      s.category_id IN (SELECT category_id FROM merchant_cats)
		      OR EXISTS (
		          SELECT 1 FROM signal_categories sc
		          WHERE sc.signal_id = s.signal_id
		            AND sc.category_id IN (SELECT category_id FROM merchant_cats)
		      )
		  )`
	}

	query := fmt.Sprintf(`
		WITH merchant_cats AS (
			-- Kategori langsung toko
			SELECT category_id FROM business_has_categories WHERE business_id = ?
			UNION
			-- Child categories: Jika toko memilih Induk "Makanan", maka sinyal dengan sub-kategori "Makanan Ringan" juga cocok
			SELECT category_id FROM business_categories WHERE parent_id IN (
				SELECT category_id FROM business_has_categories WHERE business_id = ?
			)
			UNION
			-- Parent category: Jika toko memilih sub-kategori "Makanan Ringan", maka sinyal dengan kategori Induk "Makanan" juga cocok
			SELECT parent_id AS category_id FROM business_categories WHERE category_id IN (
				SELECT category_id FROM business_has_categories WHERE business_id = ?
			) AND parent_id IS NOT NULL
		)
		SELECT 
			s.signal_id,
			s.customer_id,
			s.description,
			s.category_id,
			s.category,
			s.is_urgent,
			s.radius_km,
			s.min_budget,
			s.max_budget,
			s.max_offers,
			s.offer_count,
			s.expires_at,
			s.created_at,
			ST_Distance(s.location::geography, %s::geography) / 1000 AS distance_km,
			TRIM(CONCAT(a.first_name, ' ', a.last_name)) AS customer_name,
			a.profile_picture_url AS customer_avatar,
			-- Cek apakah toko ini sudah pernah membalas
			EXISTS(
				SELECT 1 FROM signal_conversations sc 
				WHERE sc.signal_id = s.signal_id AND sc.business_id = ?
			) AS has_replied,
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
		  -- Layer 1 (WAJIB): Kategori signal HARUS cocok dengan salah satu kategori bisnis merchant (mendukung hierarki parent & child category serta multi-kategori)
		  %s
		  -- Layer 2 (WAJIB): Signal dalam radius jangkauan yang ditentukan customer
		  AND ST_DWithin(s.location::geography, %s::geography, s.radius_km * 1000)
		  -- Saring signal yang offer_cap sudah tercapai (kecuali toko ini sudah termasuk yang membalas)
		  AND (
		      s.max_offers IS NULL 
		      OR s.max_offers <= 0 
		      OR s.offer_count < s.max_offers 
		      OR EXISTS(SELECT 1 FROM signal_conversations sc WHERE sc.signal_id = s.signal_id AND sc.business_id = ?)
		  )
		ORDER BY
			s.is_urgent DESC,
			tag_overlap_score DESC,
			s.created_at DESC
	`, merchantPoint, categoryClause, merchantPoint)

	err := r.db.Raw(query,
		merchantBusinessID, // merchant_cats: exact
		merchantBusinessID, // merchant_cats: child
		merchantBusinessID, // merchant_cats: parent
		merchantBusinessID, // Layer 3: EXISTS signal_conversations
		merchantBusinessID, // Layer 3: COUNT signal_tags
		merchantBusinessID, // offer cap check
	).Scan(&results).Error
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
		results[i].CategoryIDs = []int64{}
		results[i].Categories = []string{}
		if item.CategoryID != nil && *item.CategoryID > 0 {
			results[i].CategoryIDs = []int64{*item.CategoryID}
			if item.Category != "" {
				results[i].Categories = []string{item.Category}
			}
		}
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

	// 3. Batch query semua kategori signal dari signal_categories dalam 1 query (jika tabel ada)
	if hasSignalCategories(r.db) {
		type catRow struct {
			SignalID     string `gorm:"column:signal_id"`
			CategoryID   int64  `gorm:"column:category_id"`
			CategoryName string `gorm:"column:name"`
		}
		var allCats []catRow
		if err := r.db.Raw(`
			SELECT sc.signal_id, sc.category_id, bc.name
			FROM signal_categories sc
			JOIN business_categories bc ON sc.category_id = bc.category_id
			WHERE sc.signal_id IN ?
			ORDER BY sc.category_id ASC
		`, signalIDs).Scan(&allCats).Error; err == nil && len(allCats) > 0 {
			catIDMap := make(map[string][]int64)
			catNameMap := make(map[string][]string)
			for _, c := range allCats {
				catIDMap[c.SignalID] = append(catIDMap[c.SignalID], c.CategoryID)
				catNameMap[c.SignalID] = append(catNameMap[c.SignalID], c.CategoryName)
			}
			for i, item := range results {
				if ids, ok := catIDMap[item.SignalID]; ok && len(ids) > 0 {
					results[i].CategoryIDs = ids
					results[i].Categories = catNameMap[item.SignalID]
				}
			}
		}
	}

	for i := range results {
		if len(results[i].Categories) == 0 {
			if results[i].Category != "" {
				results[i].Categories = []string{results[i].Category}
			} else {
				results[i].Categories = []string{}
			}
		}
		if len(results[i].CategoryIDs) == 0 {
			if results[i].CategoryID != nil && *results[i].CategoryID > 0 {
				results[i].CategoryIDs = []int64{*results[i].CategoryID}
			} else {
				results[i].CategoryIDs = []int64{}
			}
		}
		if results[i].Images == nil {
			results[i].Images = []string{}
		}
		if results[i].Tags == nil {
			results[i].Tags = []string{}
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
			CASE 
				WHEN s.status = 'active' AND s.expires_at <= NOW() THEN 'expired'
				ELSE s.status 
			END AS signal_status,
			s.is_urgent,
			s.min_budget,
			s.max_budget,
			s.customer_id AS peer_id,
			TRIM(CONCAT(ca.first_name, ' ', ca.last_name)) AS peer_name,
			ca.profile_picture_url AS peer_avatar,
			COALESCE(b.business_name, '') AS business_name,
			b.logo_url AS business_logo,
			COALESCE(
				o.kyc_status = 'verified' OR EXISTS (
					SELECT 1 
					FROM business_has_tags bht 
					JOIN business_tags bt ON bt.tag_id = bht.tag_id 
					WHERE bht.business_id = b.business_id 
					  AND (LOWER(bt.name) LIKE '%verif%' OR LOWER(bt.name) LIKE '%(nib)%')
				),
				false
			) AS is_verified,
			COALESCE(b.nib IS NOT NULL AND TRIM(b.nib) != '', false) AS has_nib,
			lm.message_text AS last_message,
			lm.sender_type AS last_message_sender,
			COALESCE(lm.created_at, sc.created_at) AS last_message_at,
			COALESCE(unread.cnt, 0) AS unread_count
		FROM signal_conversations sc
		JOIN signals s ON sc.signal_id = s.signal_id
		LEFT JOIN accounts ca ON s.customer_id = ca.account_id
		LEFT JOIN businesses b ON sc.business_id = b.business_id
		LEFT JOIN owners o ON o.account_id = b.account_id
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
		   OR sc.business_id IN (SELECT business_id FROM business_admin_assignments WHERE admin_id = ? AND status IN ('active', 'accepted', 'scheduled_for_deletion'))
		ORDER BY last_message_at DESC
	`

	err := r.db.Raw(query, merchantAccountID, merchantAccountID, merchantAccountID).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	r.populatePostReferencesForItems(items)
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
			CASE 
				WHEN s.status = 'active' AND s.expires_at <= NOW() THEN 'expired'
				ELSE s.status 
			END AS signal_status,
			s.is_urgent,
			s.min_budget,
			s.max_budget,
			sc.merchant_id AS peer_id,
			COALESCE(b.business_name, TRIM(CONCAT(ma.first_name, ' ', ma.last_name)), 'Merchant') AS peer_name,
			COALESCE(b.logo_url, ma.profile_picture_url) AS peer_avatar,
			COALESCE(b.business_name, '') AS business_name,
			b.logo_url AS business_logo,
			COALESCE(
				o.kyc_status = 'verified' OR EXISTS (
					SELECT 1 
					FROM business_has_tags bht 
					JOIN business_tags bt ON bt.tag_id = bht.tag_id 
					WHERE bht.business_id = b.business_id 
					  AND (LOWER(bt.name) LIKE '%verif%' OR LOWER(bt.name) LIKE '%(nib)%')
				),
				false
			) AS is_verified,
			COALESCE(b.nib IS NOT NULL AND TRIM(b.nib) != '', false) AS has_nib,
			lm.message_text AS last_message,
			lm.sender_type AS last_message_sender,
			COALESCE(lm.created_at, sc.created_at) AS last_message_at,
			COALESCE(unread.cnt, 0) AS unread_count
		FROM signal_conversations sc
		JOIN signals s ON sc.signal_id = s.signal_id
		LEFT JOIN accounts ma ON sc.merchant_id = ma.account_id
		LEFT JOIN businesses b ON sc.business_id = b.business_id
		LEFT JOIN owners o ON o.account_id = b.account_id
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
	r.populatePostReferencesForItems(items)
	return items, nil
}

func (r *signalRepository) populatePostReferencesForItems(items []models.ConversationListItem) {
	if len(items) == 0 {
		return
	}
	signalIDs := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, it := range items {
		if it.SignalID != "" && !seen[it.SignalID] {
			seen[it.SignalID] = true
			signalIDs = append(signalIDs, it.SignalID)
		}
	}

	imgMap := make(map[string][]string)
	if len(signalIDs) > 0 {
		type imgRow struct {
			SignalID string `gorm:"column:signal_id"`
			ImageURL string `gorm:"column:image_url"`
		}
		var imgs []imgRow
		if err := r.db.Raw("SELECT signal_id, image_url FROM signal_images WHERE signal_id IN ?", signalIDs).Scan(&imgs).Error; err == nil {
			for _, img := range imgs {
				if img.ImageURL != "" {
					imgMap[img.SignalID] = append(imgMap[img.SignalID], img.ImageURL)
				}
			}
		}
	}

	for i := range items {
		items[i].SignalTitle = items[i].SignalDescription
		var budgetStr string
		if items[i].MinBudget != nil && items[i].MaxBudget != nil {
			budgetStr = fmt.Sprintf("Rp %s - Rp %s", models.FormatRupiah(*items[i].MinBudget), models.FormatRupiah(*items[i].MaxBudget))
		} else if items[i].MinBudget != nil {
			budgetStr = fmt.Sprintf(">= Rp %s", models.FormatRupiah(*items[i].MinBudget))
		} else if items[i].MaxBudget != nil {
			budgetStr = fmt.Sprintf("<= Rp %s", models.FormatRupiah(*items[i].MaxBudget))
		} else {
			budgetStr = "-"
		}

		imgs := imgMap[items[i].SignalID]
		if imgs == nil {
			imgs = []string{}
		}

		items[i].IsActive = (items[i].SignalStatus == "active")
		items[i].SignalPostReference = &models.SignalPostReference{
			SignalID:    items[i].SignalID,
			Title:       items[i].SignalDescription,
			Description: items[i].SignalDescription,
			Category:    items[i].SignalCategory,
			Status:      items[i].SignalStatus,
			IsActive:    items[i].SignalStatus == "active",
			Budget:      budgetStr,
			MinBudget:   items[i].MinBudget,
			MaxBudget:   items[i].MaxBudget,
			IsUrgent:    items[i].IsUrgent,
			Images:      imgs,
		}
	}
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
		r.db.Model(&models.BusinessAdminAssignment{}).Where("business_id = ? AND admin_id = ? AND status IN ('active', 'accepted', 'scheduled_for_deletion')", businessID, accountID).Count(&count)
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

// GetMerchantIDsBySignal mengambil semua account_id pemilik dan staf toko yang punya conversation aktif pada signal
func (r *signalRepository) GetMerchantIDsBySignal(signalID string) ([]string, error) {
	var accountIDs []string
	query := `
		SELECT DISTINCT account_id FROM (
			-- Akun merchant yang langsung membalas
			SELECT sc.merchant_id AS account_id
			FROM signal_conversations sc
			WHERE sc.signal_id = ?
			UNION
			-- Pemilik toko (Owner)
			SELECT b.account_id
			FROM signal_conversations sc
			JOIN businesses b ON sc.business_id = b.business_id
			WHERE sc.signal_id = ?
			UNION
			-- Staf / Business Admin toko
			SELECT baa.admin_id AS account_id
			FROM signal_conversations sc
			JOIN business_admin_assignments baa ON sc.business_id = baa.business_id
			WHERE sc.signal_id = ?
			  AND baa.status IN ('active', 'accepted', 'scheduled_for_deletion')
		) combined
	`
	err := r.db.Raw(query, signalID, signalID, signalID).Scan(&accountIDs).Error
	return accountIDs, err
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

func (r *signalRepository) GetCategoriesByIDs(categoryIDs []int64) ([]models.Category, error) {
	if len(categoryIDs) == 0 {
		return []models.Category{}, nil
	}
	var categories []models.Category
	err := r.db.Where("category_id IN ?", categoryIDs).Find(&categories).Error
	return categories, err
}

// IncrementOfferCount menambah jumlah penawaran yang sudah masuk pada sinyal
func (r *signalRepository) IncrementOfferCount(signalID string) error {
	return r.db.Model(&models.Signal{}).
		Where("signal_id = ?", signalID).
		UpdateColumn("offer_count", gorm.Expr("offer_count + 1")).Error
}

// FindNearbyMatchingMerchantAccountIDs mencari account_id (owner & staff) toko dalam radius & kategori yang cocok (mendukung multi-kategori sinyal)
func (r *signalRepository) FindNearbyMatchingMerchantAccountIDs(categoryIDs []int64, lat, lng float64, radiusKm float64) ([]string, error) {
	if len(categoryIDs) == 0 {
		return []string{}, nil
	}
	if radiusKm <= 0 {
		radiusKm = 5
	}
	customerPoint := fmt.Sprintf("ST_SetSRID(ST_MakePoint(%f, %f), 4326)", lng, lat)
	query := fmt.Sprintf(`
		SELECT DISTINCT account_id FROM (
			-- 1. Owner Toko
			SELECT b.account_id
			FROM businesses b
			JOIN business_has_categories bhc ON b.business_id = bhc.business_id
			WHERE (
				bhc.category_id IN (?)
				OR bhc.category_id IN (SELECT parent_id FROM business_categories WHERE category_id IN (?) AND parent_id IS NOT NULL)
				OR bhc.category_id IN (SELECT category_id FROM business_categories WHERE parent_id IN (?))
			)
			  AND b.is_active = true
			  AND b.latitude IS NOT NULL AND b.longitude IS NOT NULL
			  AND ST_DWithin(ST_SetSRID(ST_MakePoint(b.longitude, b.latitude), 4326)::geography, %s::geography, ? * 1000)

			UNION

			-- 2. Staff / Business Admin Toko
			SELECT baa.admin_id AS account_id
			FROM businesses b
			JOIN business_has_categories bhc ON b.business_id = bhc.business_id
			JOIN business_admin_assignments baa ON b.business_id = baa.business_id
			WHERE (
				bhc.category_id IN (?)
				OR bhc.category_id IN (SELECT parent_id FROM business_categories WHERE category_id IN (?) AND parent_id IS NOT NULL)
				OR bhc.category_id IN (SELECT category_id FROM business_categories WHERE parent_id IN (?))
			)
			  AND b.is_active = true
			  AND baa.status IN ('active', 'accepted')
			  AND b.latitude IS NOT NULL AND b.longitude IS NOT NULL
			  AND ST_DWithin(ST_SetSRID(ST_MakePoint(b.longitude, b.latitude), 4326)::geography, %s::geography, ? * 1000)
		) combined
	`, customerPoint, customerPoint)

	var accountIDs []string
	err := r.db.Raw(query,
		categoryIDs, categoryIDs, categoryIDs, radiusKm,
		categoryIDs, categoryIDs, categoryIDs, radiusKm,
	).Scan(&accountIDs).Error
	return accountIDs, err
}

// FindMatchingProductsForSignal mencari rekomendasi produk toko yang cocok dengan kebutuhan sinyal
func (r *signalRepository) FindMatchingProductsForSignal(businessID string, queryText string, tagIDs []int64) ([]models.Product, error) {
	query := r.db.Model(&models.Product{}).
		Where("business_id = ? AND is_archived = false", businessID)

	cleanQuery := strings.TrimSpace(queryText)
	if cleanQuery != "" {
		words := strings.Fields(cleanQuery)
		if len(words) > 0 {
			var conditions []string
			var args []interface{}
			for _, w := range words {
				if len(w) >= 3 {
					conditions = append(conditions, "LOWER(name) LIKE ?")
					args = append(args, "%"+strings.ToLower(w)+"%")
				}
			}
			if len(conditions) > 0 {
				query = query.Where(strings.Join(conditions, " OR "), args...)
			}
		}
	}

	var products []models.Product
	err := query.Preload("ProductReviews").Limit(10).Find(&products).Error
	return products, err
}

