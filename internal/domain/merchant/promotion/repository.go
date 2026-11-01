package promotion

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

type PromotionRepository interface {
	HasAccessToBusiness(accountID, role, businessID string) bool
	HasPermission(accountID, businessID, permKey string) bool

	GetEligibleProducts(businessID string, targetType string) ([]models.EligibleProductItem, error)
	GetActivePromotionProductIDs(businessID string, promoType string) (map[string]string, error) // product_id -> promotion_id
	GetActiveBundlingPromotionProductIDs(businessID string) (map[string]string, error)
	CreatePromotion(promo *models.Promotion, promoProducts []models.PromotionProduct, productsToUpdate []models.Product, autoPost *models.BusinessPost) error
	CreateBundlingPromotion(promo *models.Promotion, bundleItems []models.PromotionBundleItem, autoPost *models.BusinessPost) error
	GetProductsByIDs(businessID string, productIDs []string) ([]models.Product, error)
	GetActiveBundlingPromotions(businessID string) ([]models.Promotion, error)

	GetPromotionSummary(businessID string) (models.PromotionSummary, error)
	GetPromotions(businessID, tab, cursor string, limit int) ([]models.Promotion, error)
	GetPromotionByID(businessID, promotionID string) (*models.Promotion, error)
	UpdatePromotionNotifiedAt(businessID, promotionID string, notifiedAt time.Time) error
	GetBusinessByID(businessID string) (*models.Business, error)

	ExtendPromotion(businessID, promotionID string, newEndAt time.Time, productIDs []string) error
	StopPromotion(businessID, promotionID string, promoProducts []models.PromotionProduct, postID *string) error

	HasActivePromotions(businessID string) (bool, error)
	GetProductsInActivePromotions(businessID string) ([]models.Product, error)
	RecordPromotionEvents(events []models.PromotionEvent) error

	// Post Promotions integration
	GetPromotionalPosts(businessID, tab string) ([]models.BusinessPost, error)
	GetPostByID(businessID, postID string) (*models.BusinessPost, error)
	UpdatePostPromotion(post *models.BusinessPost) error

	// Worker methods
	GetScheduledPromotionsToActivate(now time.Time) ([]models.Promotion, error)
	GetActivePromotionsToExpire(now time.Time) ([]models.Promotion, error)
	ExpirePromotionAndRestoreDiscounts(promo *models.Promotion) error
}

type promotionRepository struct {
	db *gorm.DB
}

func NewPromotionRepository(db *gorm.DB) PromotionRepository {
	return &promotionRepository{db: db}
}

func (r *promotionRepository) HasAccessToBusiness(accountID, role, businessID string) bool {
	var count int64
	if role == "owner" {
		r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count)
	} else if role == "business_admin" {
		r.db.Model(&models.BusinessAdminAssignment{}).Where("business_id = ? AND admin_id = ? AND status IN ('active', 'accepted', 'scheduled_for_deletion')", businessID, accountID).Count(&count)
	}
	return count > 0
}

func (r *promotionRepository) HasPermission(accountID, businessID, permKey string) bool {
	return permissions.HasPermission(r.db, accountID, businessID, permKey)
}

func (r *promotionRepository) GetActivePromotionProductIDs(businessID string, promoType string) (map[string]string, error) {
	type Row struct {
		ProductID   string `gorm:"column:product_id"`
		PromotionID string `gorm:"column:promotion_id"`
	}
	var rows []Row
	query := r.db.Table("promotion_products pp").
		Select("pp.product_id, pp.promotion_id").
		Joins("JOIN promotions p ON p.promotion_id = pp.promotion_id").
		Where("p.business_id = ? AND p.status IN ('active', 'scheduled')", businessID)

	if promoType != "" && database.HasPromotionTypeColumn(r.db) {
		query = query.Where("p.promotion_type = ?", promoType)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.ProductID] = row.PromotionID
	}
	return result, nil
}

func (r *promotionRepository) GetActiveBundlingPromotionProductIDs(businessID string) (map[string]string, error) {
	if !database.HasBundleItemsTable(r.db) || !database.HasPromotionTypeColumn(r.db) {
		return map[string]string{}, nil
	}

	type Row struct {
		ProductID   string `gorm:"column:product_id"`
		PromotionID string `gorm:"column:promotion_id"`
	}
	var rows []Row
	err := r.db.Table("promotion_bundle_items pbi").
		Select("pbi.product_id, pbi.promotion_id").
		Joins("JOIN promotions p ON p.promotion_id = pbi.promotion_id").
		Where("p.business_id = ? AND p.status IN ('active', 'scheduled') AND p.promotion_type = ?", businessID, models.PromotionTypeBundling).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.ProductID] = row.PromotionID
	}
	return result, nil
}

func (r *promotionRepository) GetEligibleProducts(businessID string, targetType string) ([]models.EligibleProductItem, error) {
	var products []models.Product
	err := r.db.Where("business_id = ? AND is_active = ?", businessID, true).
		Order("created_at DESC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}

	targetType = strings.ToLower(strings.TrimSpace(targetType))
	if targetType == "" {
		targetType = models.PromotionTypePerProduct
	}

	// 1. Ambil produk yang sedang berada dalam promosi per-produk
	activePerProductMap, err := r.GetActivePromotionProductIDs(businessID, models.PromotionTypePerProduct)
	if err != nil {
		return nil, err
	}

	// 2. Ambil produk yang sedang berada dalam promosi paket bundling
	activeBundlingMap, err := r.GetActiveBundlingPromotionProductIDs(businessID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]models.EligibleProductItem, 0, len(products))
	for _, p := range products {
		imageURL := ""
		if len(p.ProductPreview) > 0 {
			imageURL = p.ProductPreview[0]
		}

		inPerProductPromoID, inPerProduct := activePerProductMap[p.ProductID]
		inBundlingPromoID, inBundling := activeBundlingMap[p.ProductID]

		activePromos := make([]models.ActivePromotionInfo, 0)
		if inPerProduct {
			activePromos = append(activePromos, models.ActivePromotionInfo{
				PromotionID:   inPerProductPromoID,
				PromotionType: models.PromotionTypePerProduct,
				Status:        "active",
			})
		}
		if inBundling {
			activePromos = append(activePromos, models.ActivePromotionInfo{
				PromotionID:   inBundlingPromoID,
				PromotionType: models.PromotionTypeBundling,
				Status:        "active",
			})
		}

		isEligible := true
		ineligibleReason := ""
		isActivePromoted := false
		promoStatus := "available"
		promoStatusLabel := "Tersedia untuk promosi"
		activePromoID := ""

		if !p.IsActive {
			isEligible = false
			ineligibleReason = "Produk tidak aktif"
			promoStatus = "inactive"
			promoStatusLabel = "Produk Tidak Aktif"
		} else if targetType == models.PromotionTypeBundling {
			isEligible = true
			if inBundling {
				isActivePromoted = true
				promoStatus = "active_promoted"
				promoStatusLabel = "Sedang Aktif Bundling"
				activePromoID = inBundlingPromoID
			}
		} else {
			// Aturan Per-Produk (Katalog):
			// Produk yang sedang ada di promosi per-produk TIDAK BISA didaftarkan lagi (mencegah konflik diskon katalog).
			if inPerProduct {
				isEligible = false
				isActivePromoted = true
				promoStatus = "active_promoted"
				promoStatusLabel = "Sedang Aktif Promosi"
				activePromoID = inPerProductPromoID
				ineligibleReason = fmt.Sprintf("Produk sedang aktif dalam promosi (%s)", inPerProductPromoID)
			}
		}

		hasDiscount := utils.IsDiscountActive(p.Discount, p.DiscountStartAt, p.DiscountEndAt, now)

		items = append(items, models.EligibleProductItem{
			ProductID:              p.ProductID,
			ProductName:            p.ProductName,
			ImageURL:               imageURL,
			Price:                  p.Price,
			IsEligible:             isEligible,
			IneligibleReason:       ineligibleReason,
			IsActivePromoted:       isActivePromoted,
			PromotionStatus:        promoStatus,
			PromotionStatusLabel:   promoStatusLabel,
			ActivePromotionID:      activePromoID,
			HasExistingDiscount:    hasDiscount,
			CurrentDiscountType:    p.DiscountType,
			CurrentDiscount:        p.Discount,
			CurrentMaxDiscount:     p.MaxDiscount,
			CurrentDiscountStartAt: p.DiscountStartAt,
			CurrentDiscountEndAt:   p.DiscountEndAt,
			InPerProductPromotion:  inPerProduct,
			InBundlingPromotion:    inBundling,
			ActivePromotions:       activePromos,
		})
	}

	// Aturan Urutan: Produk yang sedang aktif promosi (isActivePromoted == true)
	// otomatis diletakkan di PALING BAWAH list.
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsActivePromoted != items[j].IsActivePromoted {
			return !items[i].IsActivePromoted // false (belum aktif promo) ditaruh di atas
		}
		if items[i].IsEligible != items[j].IsEligible {
			return items[i].IsEligible // yang eligible ditaruh lebih dulu
		}
		return false // pertahankan urutan default (created_at DESC)
	})

	return items, nil
}

func (r *promotionRepository) CreatePromotion(
	promo *models.Promotion,
	promoProducts []models.PromotionProduct,
	productsToUpdate []models.Product,
	autoPost *models.BusinessPost,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Simpan master promosi terlebih dahulu (tanpa PostID) agar baris promotions tersedia untuk FK business_posts
		promo.PostID = nil
		promoQuery := tx
		if !database.HasPromotionTypeColumn(tx) {
			promoQuery = promoQuery.Omit("PromotionType", "PackageName", "PackagePrice", "PackageImage", "OriginalTotalPrice", "SavingsAmount", "SavingsPercentage")
		}
		if err := promoQuery.Create(promo).Error; err != nil {
			return fmt.Errorf("gagal menyimpan promosi: %w", err)
		}

		// 2. Simpan auto-post feed jika ada (sekarang aman karena promotions(promotion_id) sudah tersimpan)
		if autoPost != nil {
			if err := tx.Create(autoPost).Error; err != nil {
				return fmt.Errorf("gagal membuat postingan feed promosi: %w", err)
			}
			promo.PostID = &autoPost.PostID

			// Tautkan post_id kembali ke promosi
			if err := tx.Model(&models.Promotion{}).
				Where("promotion_id = ?", promo.PromotionID).
				Update("post_id", autoPost.PostID).Error; err != nil {
				return fmt.Errorf("gagal menautkan postingan ke promosi: %w", err)
			}
		}

		// 3. Simpan relasi produk promosi
		if len(promoProducts) > 0 {
			if err := tx.Create(&promoProducts).Error; err != nil {
				return fmt.Errorf("gagal menyimpan item promosi: %w", err)
			}
		}

		// 4. Update data diskon pada tabel products jika promosi langsung aktif
		for _, p := range productsToUpdate {
			updates := map[string]interface{}{
				"discount_type":     p.DiscountType,
				"discount":          p.Discount,
				"max_discount":      p.MaxDiscount,
				"discount_start_at": p.DiscountStartAt,
				"discount_end_at":   p.DiscountEndAt,
				"updated_at":        time.Now(),
			}
			if err := tx.Model(&models.Product{}).
				Where("product_id = ? AND business_id = ?", p.ProductID, promo.BusinessID).
				Updates(updates).Error; err != nil {
				return fmt.Errorf("gagal memperbarui diskon produk %s: %w", p.ProductID, err)
			}
		}

		return nil
	})
}

func (r *promotionRepository) CreateBundlingPromotion(
	promo *models.Promotion,
	bundleItems []models.PromotionBundleItem,
	autoPost *models.BusinessPost,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Simpan master promosi bundling (tanpa PostID)
		if !database.HasPromotionTypeColumn(tx) || !database.HasBundleItemsTable(tx) {
			return fmt.Errorf("skema database untuk promosi bundling belum siap. Mohon jalankan migrasi database terlebih dahulu")
		}
		promo.PostID = nil
		promo.PromotionType = models.PromotionTypeBundling
		if err := tx.Create(promo).Error; err != nil {
			return fmt.Errorf("gagal menyimpan promosi bundling: %w", err)
		}

		// 2. Simpan auto-post feed jika ada
		if autoPost != nil {
			if err := tx.Create(autoPost).Error; err != nil {
				return fmt.Errorf("gagal membuat postingan feed promosi bundling: %w", err)
			}
			promo.PostID = &autoPost.PostID

			// Tautkan post_id kembali ke promosi
			if err := tx.Model(&models.Promotion{}).
				Where("promotion_id = ?", promo.PromotionID).
				Update("post_id", autoPost.PostID).Error; err != nil {
				return fmt.Errorf("gagal menautkan postingan ke promosi: %w", err)
			}
		}

		// 3. Simpan item-item bundling
		if len(bundleItems) > 0 {
			if err := tx.Create(&bundleItems).Error; err != nil {
				return fmt.Errorf("gagal menyimpan item bundling: %w", err)
			}
		}

		return nil
	})
}

func (r *promotionRepository) GetProductsByIDs(businessID string, productIDs []string) ([]models.Product, error) {
	var products []models.Product
	if len(productIDs) == 0 {
		return products, nil
	}
	err := r.db.Where("business_id = ? AND product_id IN ? AND is_active = true", businessID, productIDs).Find(&products).Error
	return products, err
}

func (r *promotionRepository) GetActiveBundlingPromotions(businessID string) ([]models.Promotion, error) {
	if !database.HasPromotionTypeColumn(r.db) || !database.HasBundleItemsTable(r.db) {
		return []models.Promotion{}, nil
	}

	var promos []models.Promotion
	err := r.db.
		Preload("BundleItems.Product").
		Where("business_id = ? AND promotion_type = ? AND status = ?", businessID, models.PromotionTypeBundling, models.PromotionStatusActive).
		Order("created_at DESC").
		Find(&promos).Error
	return promos, err
}

func (r *promotionRepository) GetPromotionSummary(businessID string) (models.PromotionSummary, error) {
	var summary models.PromotionSummary
	type Row struct {
		ActiveCount      int64 `gorm:"column:active_count"`
		TotalImpressions int64 `gorm:"column:total_impressions"`
		TotalClicks      int64 `gorm:"column:total_clicks"`
	}
	var row Row
	err := r.db.Model(&models.Promotion{}).
		Select("COUNT(*) as active_count, COALESCE(SUM(impression_count), 0) as total_impressions, COALESCE(SUM(click_count), 0) as total_clicks").
		Where("business_id = ? AND status = ?", businessID, models.PromotionStatusActive).
		Scan(&row).Error
	if err != nil {
		return summary, err
	}
	summary.ActiveCount = row.ActiveCount
	summary.TotalImpressions = row.TotalImpressions
	summary.TotalClicks = row.TotalClicks

	// Tambahkan jumlah postingan promosi yang sedang aktif
	now := time.Now()
	var activePostCount int64
	_ = r.db.Model(&models.BusinessPost{}).
		Where("business_id = ? AND is_promotion = true AND (promotion_end_at IS NULL OR promotion_end_at > ?)", businessID, now).
		Count(&activePostCount).Error
	summary.ActiveCount += activePostCount

	return summary, nil
}

func (r *promotionRepository) GetPromotions(businessID, tab, cursor string, limit int) ([]models.Promotion, error) {
	if limit <= 0 {
		limit = 10
	}

	query := r.db.Model(&models.Promotion{}).
		Preload("MainProduct").
		Preload("Products").
		Where("business_id = ?", businessID)

	if database.HasBundleItemsTable(r.db) {
		query = query.Preload("BundleItems.Product")
	}

	switch tab {
	case "popular":
		query = query.Where("status = ?", models.PromotionStatusActive).
			Order("click_count DESC, created_at DESC")
	case "ended":
		query = query.Where("status IN ('ended', 'stopped')").
			Order("end_at DESC, created_at DESC")
	default: // "active"
		query = query.Where("status IN ('active', 'scheduled')").
			Order("end_at ASC, created_at DESC")
	}

	if cursor != "" {
		var cursorPromo models.Promotion
		if err := r.db.Select("promotion_id", "click_count", "end_at", "created_at").
			Where("promotion_id = ?", cursor).First(&cursorPromo).Error; err == nil {
			switch tab {
			case "popular":
				query = query.Where("click_count < ? OR (click_count = ? AND created_at < ?)",
					cursorPromo.ClickCount, cursorPromo.ClickCount, cursorPromo.CreatedAt)
			case "ended":
				query = query.Where("end_at < ? OR (end_at = ? AND promotion_id < ?)",
					cursorPromo.EndAt, cursorPromo.EndAt, cursorPromo.PromotionID)
			default:
				query = query.Where("end_at > ? OR (end_at = ? AND promotion_id > ?)",
					cursorPromo.EndAt, cursorPromo.EndAt, cursorPromo.PromotionID)
			}
		}
	}

	var promos []models.Promotion
	err := query.Limit(limit + 1).Find(&promos).Error
	return promos, err
}

func (r *promotionRepository) GetPromotionByID(businessID, promotionID string) (*models.Promotion, error) {
	var promo models.Promotion
	query := r.db.
		Preload("Business").
		Preload("MainProduct").
		Preload("Post").
		Preload("Products.Product").
		Where("promotion_id = ?", strings.TrimSpace(promotionID))

	if database.HasBundleItemsTable(r.db) {
		query = query.Preload("BundleItems.Product")
	}

	if strings.TrimSpace(businessID) != "" {
		query = query.Where("business_id = ?", strings.TrimSpace(businessID))
	}

	err := query.First(&promo).Error
	if err != nil {
		return nil, err
	}
	return &promo, nil
}

func (r *promotionRepository) UpdatePromotionNotifiedAt(businessID, promotionID string, notifiedAt time.Time) error {
	return r.db.Model(&models.Promotion{}).
		Where("business_id = ? AND promotion_id = ?", businessID, promotionID).
		Update("notified_at", notifiedAt).Error
}

func (r *promotionRepository) GetBusinessByID(businessID string) (*models.Business, error) {
	var biz models.Business
	if err := r.db.Where("business_id = ?", businessID).First(&biz).Error; err != nil {
		return nil, err
	}
	return &biz, nil
}

func (r *promotionRepository) ExtendPromotion(businessID, promotionID string, newEndAt time.Time, productIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Update promosi
		if err := tx.Model(&models.Promotion{}).
			Where("business_id = ? AND promotion_id = ?", businessID, promotionID).
			Updates(map[string]interface{}{
				"end_at":     newEndAt,
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}

		// Update discount_end_at pada seluruh produk promosi
		if len(productIDs) > 0 {
			if err := tx.Model(&models.Product{}).
				Where("product_id IN ? AND business_id = ?", productIDs, businessID).
				Updates(map[string]interface{}{
					"discount_end_at": newEndAt,
					"updated_at":      time.Now(),
				}).Error; err != nil {
				return err
			}
		}

		// Update postingan promosi terkait
		_ = tx.Model(&models.BusinessPost{}).
			Where("promotion_id = ?", promotionID).
			Updates(map[string]interface{}{
				"is_promotion":     true,
				"promotion_end_at": newEndAt,
				"updated_at":       time.Now(),
			}).Error

		return nil
	})
}

func (r *promotionRepository) StopPromotion(
	businessID, promotionID string,
	promoProducts []models.PromotionProduct,
	postID *string,
) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update status promosi
		if err := tx.Model(&models.Promotion{}).
			Where("business_id = ? AND promotion_id = ?", businessID, promotionID).
			Updates(map[string]interface{}{
				"status":     models.PromotionStatusStopped,
				"stopped_at": now,
				"end_at":     now,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		// 2. Kembalikan / bersihkan diskon produk
		for _, pp := range promoProducts {
			if pp.DiscountSource == models.PromotionDiscountSourceNew {
				if pp.HadPreviousDiscount && pp.PrevDiscount != nil && *pp.PrevDiscount > 0 {
					// Rollback ke diskon lama
					prevType := "nominal"
					if pp.PrevDiscountType != nil {
						prevType = *pp.PrevDiscountType
					}
					if err := tx.Model(&models.Product{}).
						Where("product_id = ? AND business_id = ?", pp.ProductID, businessID).
						Updates(map[string]interface{}{
							"discount_type":     prevType,
							"discount":          *pp.PrevDiscount,
							"max_discount":      pp.PrevMaxDiscount,
							"discount_start_at": pp.PrevDiscountStartAt,
							"discount_end_at":   pp.PrevDiscountEndAt,
							"updated_at":        now,
						}).Error; err != nil {
						return err
					}
				} else {
					// Hapus diskon
					if err := tx.Model(&models.Product{}).
						Where("product_id = ? AND business_id = ?", pp.ProductID, businessID).
						Updates(map[string]interface{}{
							"discount_type":     "nominal",
							"discount":          0,
							"max_discount":      nil,
							"discount_start_at": nil,
							"discount_end_at":   nil,
							"updated_at":        now,
						}).Error; err != nil {
						return err
					}
				}
			} else if pp.DiscountSource == models.PromotionDiscountSourceKept {
				// Kembalikan batas tanggal diskon lama
				if err := tx.Model(&models.Product{}).
					Where("product_id = ? AND business_id = ?", pp.ProductID, businessID).
					Updates(map[string]interface{}{
						"discount_end_at": pp.PrevDiscountEndAt,
						"updated_at":      now,
					}).Error; err != nil {
					return err
				}
			}
		}

		// 3. Tandai post promosi tidak aktif lagi jika ada
		postQuery := tx.Model(&models.BusinessPost{}).Where("promotion_id = ?", promotionID)
		if postID != nil && *postID != "" {
			postQuery = tx.Model(&models.BusinessPost{}).Where("promotion_id = ? OR (business_id = ? AND post_id = ?)", promotionID, businessID, *postID)
		}
		_ = postQuery.Updates(map[string]interface{}{
			"is_promotion": false,
			"updated_at":   now,
		}).Error

		return nil
	})
}

func (r *promotionRepository) HasActivePromotions(businessID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Promotion{}).
		Where("business_id = ? AND status = ?", businessID, models.PromotionStatusActive).
		Count(&count).Error
	return count > 0, err
}

func (r *promotionRepository) GetActivePromotionIDsForBusiness(businessID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&models.Promotion{}).
		Where("business_id = ? AND status = ?", businessID, models.PromotionStatusActive).
		Pluck("promotion_id", &ids).Error
	return ids, err
}

func (r *promotionRepository) GetProductsInActivePromotions(businessID string) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Model(&models.Product{}).
		Joins("JOIN promotion_products pp ON pp.product_id = products.product_id").
		Joins("JOIN promotions p ON p.promotion_id = pp.promotion_id").
		Where("p.business_id = ? AND p.status = ? AND products.is_active = ?",
			businessID, models.PromotionStatusActive, true).
		Distinct("products.*").
		Order("products.created_at DESC").
		Find(&products).Error
	return products, err
}

func (r *promotionRepository) RecordPromotionEvents(events []models.PromotionEvent) error {
	if len(events) == 0 {
		return nil
	}

	ctx := context.Background()
	now := time.Now()

	// Hitung delta per promosi dengan deduplikasi impression via Redis
	impressionDeltas := make(map[string]int)
	clickDeltas := make(map[string]int)
	validEvents := make([]models.PromotionEvent, 0, len(events))

	for _, ev := range events {
		if ev.PromotionID == "" {
			continue
		}
		if ev.EventID == "" {
			ev.EventID = utils.GenerateUUID("PEV")
		}
		if ev.CreatedAt.IsZero() {
			ev.CreatedAt = now
		}

		if ev.EventType == models.PromotionEventImpression {
			// Deduplikasi 30 menit per user
			shouldRecord := true
			if config.RDB != nil && ev.AccountID != nil && *ev.AccountID != "" {
				key := fmt.Sprintf("promo:imp:%s:%s", ev.PromotionID, *ev.AccountID)
				set, err := config.RDB.SetNX(ctx, key, "1", 30*time.Minute).Result()
				if err == nil && !set {
					shouldRecord = false // Sudah tercatat dalam 30 menit terakhir
				}
			}
			if shouldRecord {
				impressionDeltas[ev.PromotionID]++
				validEvents = append(validEvents, ev)
			}
		} else if ev.EventType == models.PromotionEventClick {
			clickDeltas[ev.PromotionID]++
			validEvents = append(validEvents, ev)
		}
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(validEvents) > 0 {
			if err := tx.Create(&validEvents).Error; err != nil {
				return err
			}
		}

		for promoID, count := range impressionDeltas {
			if count > 0 {
				_ = tx.Model(&models.Promotion{}).
					Where("promotion_id = ?", promoID).
					UpdateColumn("impression_count", gorm.Expr("impression_count + ?", count)).Error
			}
		}

		for promoID, count := range clickDeltas {
			if count > 0 {
				_ = tx.Model(&models.Promotion{}).
					Where("promotion_id = ?", promoID).
					UpdateColumn("click_count", gorm.Expr("click_count + ?", count)).Error
			}
		}

		return nil
	})
}

func (r *promotionRepository) GetScheduledPromotionsToActivate(now time.Time) ([]models.Promotion, error) {
	var promos []models.Promotion
	query := r.db.
		Preload("MainProduct").
		Preload("Products.Product").
		Where("status = ? AND start_at <= ?", models.PromotionStatusScheduled, now)

	if database.HasBundleItemsTable(r.db) {
		query = query.Preload("BundleItems.Product")
	}

	err := query.Find(&promos).Error
	return promos, err
}

func (r *promotionRepository) GetActivePromotionsToExpire(now time.Time) ([]models.Promotion, error) {
	var promos []models.Promotion
	err := r.db.
		Preload("Products").
		Where("status = ? AND end_at <= ?", models.PromotionStatusActive, now).
		Find(&promos).Error
	return promos, err
}

func (r *promotionRepository) ExpirePromotionAndRestoreDiscounts(promo *models.Promotion) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Promotion{}).
			Where("promotion_id = ? AND status = ?", promo.PromotionID, models.PromotionStatusActive).
			Updates(map[string]interface{}{
				"status":     models.PromotionStatusEnded,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		for _, pp := range promo.Products {
			if pp.DiscountSource == models.PromotionDiscountSourceNew {
				if pp.HadPreviousDiscount && pp.PrevDiscount != nil && *pp.PrevDiscount > 0 {
					prevType := "nominal"
					if pp.PrevDiscountType != nil {
						prevType = *pp.PrevDiscountType
					}
					_ = tx.Model(&models.Product{}).
						Where("product_id = ? AND business_id = ?", pp.ProductID, promo.BusinessID).
						Updates(map[string]interface{}{
							"discount_type":     prevType,
							"discount":          *pp.PrevDiscount,
							"max_discount":      pp.PrevMaxDiscount,
							"discount_start_at": pp.PrevDiscountStartAt,
							"discount_end_at":   pp.PrevDiscountEndAt,
							"updated_at":        now,
						}).Error
				} else {
					_ = tx.Model(&models.Product{}).
						Where("product_id = ? AND business_id = ?", pp.ProductID, promo.BusinessID).
						Updates(map[string]interface{}{
							"discount_type":     "nominal",
							"discount":          0,
							"max_discount":      nil,
							"discount_start_at": nil,
							"discount_end_at":   nil,
							"updated_at":        now,
						}).Error
				}
			} else if pp.DiscountSource == models.PromotionDiscountSourceKept {
				_ = tx.Model(&models.Product{}).
					Where("product_id = ? AND business_id = ?", pp.ProductID, promo.BusinessID).
					Updates(map[string]interface{}{
						"discount_end_at": pp.PrevDiscountEndAt,
						"updated_at":      now,
					}).Error
			}
		}

		// Matikan status promosi pada postingan feed terkait
		_ = tx.Model(&models.BusinessPost{}).
			Where("promotion_id = ?", promo.PromotionID).
			Updates(map[string]interface{}{
				"is_promotion": false,
				"updated_at":   now,
			}).Error

		return nil
	})
}

func (r *promotionRepository) GetPromotionalPosts(businessID, tab string) ([]models.BusinessPost, error) {
	var posts []models.BusinessPost
	now := time.Now()
	query := r.db.Model(&models.BusinessPost{}).Where("business_id = ?", businessID)

	switch tab {
	case "popular":
		query = query.Where("is_promotion = true AND (promotion_end_at IS NULL OR promotion_end_at > ?)", now).
			Order("like_count DESC, created_at DESC")
	case "ended":
		query = query.Where("(is_promotion = false AND promotion_end_at IS NOT NULL) OR (is_promotion = true AND promotion_end_at <= ?)", now).
			Order("promotion_end_at DESC, created_at DESC")
	default: // "active"
		query = query.Where("is_promotion = true AND (promotion_end_at IS NULL OR promotion_end_at > ?)", now).
			Order("promotion_end_at ASC, created_at DESC")
	}

	err := query.Find(&posts).Error
	return posts, err
}

func (r *promotionRepository) GetPostByID(businessID, postID string) (*models.BusinessPost, error) {
	var post models.BusinessPost
	query := r.db.Model(&models.BusinessPost{}).Where("post_id = ?", postID)
	if businessID != "" {
		query = query.Where("business_id = ?", businessID)
	}
	if err := query.First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *promotionRepository) UpdatePostPromotion(post *models.BusinessPost) error {
	return r.db.Save(post).Error
}
