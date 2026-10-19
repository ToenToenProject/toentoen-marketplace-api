package promotion

import (
	"context"
	"fmt"
	"time"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

type PromotionRepository interface {
	HasAccessToBusiness(accountID, role, businessID string) bool
	HasPermission(accountID, businessID, permKey string) bool

	GetEligibleProducts(businessID string) ([]models.EligibleProductItem, error)
	GetActivePromotionProductIDs(businessID string) (map[string]string, error) // product_id -> promotion_id
	CreatePromotion(promo *models.Promotion, promoProducts []models.PromotionProduct, productsToUpdate []models.Product, autoPost *models.BusinessPost) error

	GetPromotionSummary(businessID string) (models.PromotionSummary, error)
	GetPromotions(businessID, tab, cursor string, limit int) ([]models.Promotion, error)
	GetPromotionByID(businessID, promotionID string) (*models.Promotion, error)

	ExtendPromotion(businessID, promotionID string, newEndAt time.Time, productIDs []string) error
	StopPromotion(businessID, promotionID string, promoProducts []models.PromotionProduct, postID *string) error

	HasActivePromotions(businessID string) (bool, error)
	GetProductsInActivePromotions(businessID string) ([]models.Product, error)
	GetActivePromotionIDsForBusiness(businessID string) ([]string, error)

	RecordPromotionEvents(events []models.PromotionEvent) error

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

func (r *promotionRepository) GetActivePromotionProductIDs(businessID string) (map[string]string, error) {
	type Row struct {
		ProductID   string `gorm:"column:product_id"`
		PromotionID string `gorm:"column:promotion_id"`
	}
	var rows []Row
	err := r.db.Table("promotion_products pp").
		Select("pp.product_id, pp.promotion_id").
		Joins("JOIN promotions p ON p.promotion_id = pp.promotion_id").
		Where("p.business_id = ? AND p.status IN ('active', 'scheduled')", businessID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, r := range rows {
		result[r.ProductID] = r.PromotionID
	}
	return result, nil
}

func (r *promotionRepository) GetEligibleProducts(businessID string) ([]models.EligibleProductItem, error) {
	var products []models.Product
	err := r.db.Where("business_id = ? AND is_active = ?", businessID, true).
		Order("created_at DESC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}

	activePromoMap, err := r.GetActivePromotionProductIDs(businessID)
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

		inPromoID, inPromo := activePromoMap[p.ProductID]
		isEligible := !inPromo
		ineligibleReason := ""
		if inPromo {
			ineligibleReason = fmt.Sprintf("Produk sedang dalam promosi (%s)", inPromoID)
		}

		hasDiscount := utils.IsDiscountActive(p.Discount, p.DiscountStartAt, p.DiscountEndAt, now)

		items = append(items, models.EligibleProductItem{
			ProductID:              p.ProductID,
			ProductName:            p.ProductName,
			ImageURL:               imageURL,
			Price:                  p.Price,
			IsEligible:             isEligible,
			IneligibleReason:       ineligibleReason,
			HasExistingDiscount:    hasDiscount,
			CurrentDiscountType:    p.DiscountType,
			CurrentDiscount:        p.Discount,
			CurrentMaxDiscount:     p.MaxDiscount,
			CurrentDiscountStartAt: p.DiscountStartAt,
			CurrentDiscountEndAt:   p.DiscountEndAt,
		})
	}

	return items, nil
}

func (r *promotionRepository) CreatePromotion(
	promo *models.Promotion,
	promoProducts []models.PromotionProduct,
	productsToUpdate []models.Product,
	autoPost *models.BusinessPost,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Simpan auto-post feed jika ada
		if autoPost != nil {
			if err := tx.Create(autoPost).Error; err != nil {
				return fmt.Errorf("gagal membuat postingan feed promosi: %w", err)
			}
			promo.PostID = &autoPost.PostID
		}

		// 2. Simpan master promosi
		if err := tx.Create(promo).Error; err != nil {
			return fmt.Errorf("gagal menyimpan promosi: %w", err)
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
	err := r.db.
		Preload("MainProduct").
		Preload("Post").
		Preload("Products.Product").
		Where("business_id = ? AND promotion_id = ?", businessID, promotionID).
		First(&promo).Error
	if err != nil {
		return nil, err
	}
	return &promo, nil
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
		if postID != nil && *postID != "" {
			_ = tx.Model(&models.BusinessPost{}).
				Where("post_id = ? AND business_id = ?", *postID, businessID).
				Updates(map[string]interface{}{
					"is_promotion": false,
					"updated_at":   now,
				}).Error
		}

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
	err := r.db.
		Preload("MainProduct").
		Preload("Products.Product").
		Where("status = ? AND start_at <= ?", models.PromotionStatusScheduled, now).
		Find(&promos).Error
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

		return nil
	})
}
