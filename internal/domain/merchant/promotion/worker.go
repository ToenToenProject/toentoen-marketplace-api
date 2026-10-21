package promotion

import (
	"log"
	"time"

	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

// StartPromotionWorker menjalankan background scheduler setiap 1 menit untuk:
// 1. Mengaktifkan promosi terjadwal (scheduled -> active), membuat post feed, dan mengirim notifikasi broadcast.
// 2. Menyelesaikan promosi yang kedaluwarsa (active -> ended) dan mengembalikan diskon produk ke kondisi semula.
func StartPromotionWorker(db *gorm.DB, notifService notification.NotificationService) {
	if db == nil {
		return
	}

	repo := NewPromotionRepository(db)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			processScheduledPromotions(db, repo, notifService, now)
			processExpiredPromotions(repo, now)
			processExpiredPromotionalPosts(db, now)
		}
	}()
}

func processScheduledPromotions(
	db *gorm.DB,
	repo PromotionRepository,
	notifService notification.NotificationService,
	now time.Time,
) {
	promos, err := repo.GetScheduledPromotionsToActivate(now)
	if err != nil || len(promos) == 0 {
		return
	}

	for _, promo := range promos {
		pCopy := promo
		err := db.Transaction(func(tx *gorm.DB) error {
			// 1. Update status promosi menjadi active
			if err := tx.Model(&models.Promotion{}).
				Where("promotion_id = ? AND status = ?", pCopy.PromotionID, models.PromotionStatusScheduled).
				Updates(map[string]interface{}{
					"status":      models.PromotionStatusActive,
					"notified_at": now,
					"updated_at":  now,
				}).Error; err != nil {
				return err
			}

			// 2. Buat auto-post feed promosi
			img := ""
			if pCopy.MainProduct != nil && len(pCopy.MainProduct.ProductPreview) > 0 {
				img = pCopy.MainProduct.ProductPreview[0]
			}
			images := []string{}
			if img != "" {
				images = append(images, img)
			}

			autoPost := models.BusinessPost{
				PostID:         utils.GenerateUUID("POST"),
				BusinessID:     pCopy.BusinessID,
				Content:        pCopy.Caption,
				Images:         images,
				IsPromotion:    true,
				PromotionID:    &pCopy.PromotionID,
				PromotionEndAt: &pCopy.EndAt,
				LikeCount:      0,
				CommentCount:   0,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&autoPost).Error; err != nil {
				return err
			}

			// Simpan link post_id ke promosi
			if err := tx.Model(&models.Promotion{}).
				Where("promotion_id = ?", pCopy.PromotionID).
				Update("post_id", autoPost.PostID).Error; err != nil {
				return err
			}

			// 3. Tulis diskon ke tabel products
			for _, pp := range pCopy.Products {
				if err := tx.Model(&models.Product{}).
					Where("product_id = ? AND business_id = ?", pp.ProductID, pCopy.BusinessID).
					Updates(map[string]interface{}{
						"discount_type":     pp.DiscountType,
						"discount":          pp.Discount,
						"max_discount":      pp.MaxDiscount,
						"discount_start_at": &pCopy.StartAt,
						"discount_end_at":   &pCopy.EndAt,
						"updated_at":        now,
					}).Error; err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("⚠️ [PROMOTION-WORKER] Gagal mengaktifkan promosi %s: %v", pCopy.PromotionID, err)
			continue
		}

		log.Printf("🎉 [PROMOTION-WORKER] Berhasil mengaktifkan promosi %s untuk bisnis %s", pCopy.PromotionID, pCopy.BusinessID)

		// 4. Kirim notifikasi broadcast & sync search
		if notifService != nil {
			productIDs := make([]string, 0, len(pCopy.Products))
			for _, pp := range pCopy.Products {
				productIDs = append(productIDs, pp.ProductID)
			}
			notification.NotifyPromotionCreated(database.DB, notifService, pCopy.BusinessID, &pCopy, pCopy.MainProduct, productIDs, pCopy.CreatedBy)
		}
		go meilisearch.SyncBusiness(database.DB, pCopy.BusinessID)
	}
}

func processExpiredPromotions(repo PromotionRepository, now time.Time) {
	promos, err := repo.GetActivePromotionsToExpire(now)
	if err != nil || len(promos) == 0 {
		return
	}

	for _, promo := range promos {
		pCopy := promo
		if err := repo.ExpirePromotionAndRestoreDiscounts(&pCopy); err != nil {
			log.Printf("⚠️ [PROMOTION-WORKER] Gagal menyelesaikan promosi %s: %v", pCopy.PromotionID, err)
			continue
		}
		log.Printf("⏱️ [PROMOTION-WORKER] Promosi %s telah berakhir secara otomatis", pCopy.PromotionID)
		go meilisearch.SyncBusiness(database.DB, pCopy.BusinessID)
	}
}

var hasPromotionEndAtColumn bool

func processExpiredPromotionalPosts(db *gorm.DB, now time.Time) {
	if !hasPromotionEndAtColumn {
		if db.Migrator().HasTable(&models.BusinessPost{}) && db.Migrator().HasColumn(&models.BusinessPost{}, "promotion_end_at") {
			hasPromotionEndAtColumn = true
		} else {
			// Coba buat kolom jika belum ada
			if err := db.Exec("ALTER TABLE business_posts ADD COLUMN IF NOT EXISTS promotion_end_at TIMESTAMPTZ").Error; err == nil {
				hasPromotionEndAtColumn = true
			} else {
				// Kolom belum ada dan belum bisa dibuat, tunda hingga migrasi dijalankan
				return
			}
		}
	}

	err := db.Model(&models.BusinessPost{}).
		Where("is_promotion = true AND promotion_end_at IS NOT NULL AND promotion_end_at <= ?", now).
		Updates(map[string]interface{}{
			"is_promotion": false,
			"updated_at":   now,
		}).Error
	if err != nil {
		log.Printf("⚠️ [PROMOTION-WORKER] Gagal menghentikan postingan promosi yang expired: %v", err)
	}
}
