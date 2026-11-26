package notification

import (
	"fmt"
	"log"
	"strings"
	"time"

	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// NotifyPromotionCreated broadcasts promotional notifications to store bookmarkers.
// Dispatches alerts to store bookmarkers and product wishlisters.
func NotifyPromotionCreated(
	db *gorm.DB,
	notifService NotificationService,
	businessID string,
	promo *models.Promotion,
	mainProduct *models.Product,
	productIDs []string,
	actorID string,
) {
	if db == nil || notifService == nil || businessID == "" || promo == nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ [PROMOTION-NOTIF-PANIC] Terjadi panic saat mengirim notifikasi promosi: %v", r)
			}
		}()

		var biz models.Business
		if err := db.Select("business_name").Where("business_id = ?", businessID).First(&biz).Error; err != nil {
			return
		}
		businessName := "Toko"
		if biz.BusinessName != nil && strings.TrimSpace(*biz.BusinessName) != "" {
			businessName = strings.TrimSpace(*biz.BusinessName)
		}

		var bookmarkIDs []string
		_ = db.Table("business_bookmarks").
			Where("business_id = ?", businessID).
			Pluck("account_id", &bookmarkIDs).Error

		var wishlistAccountIDs []string
		if len(productIDs) > 0 {
			_ = db.Table("product_wishlists").
				Where("product_id IN ?", productIDs).
				Distinct("account_id").
				Pluck("account_id", &wishlistAccountIDs).Error
		}

		recipientMap := make(map[string]bool)
		for _, id := range bookmarkIDs {
			if id != "" && id != actorID {
				recipientMap[id] = true
			}
		}
		for _, id := range wishlistAccountIDs {
			if id != "" && id != actorID {
				recipientMap[id] = true
			}
		}

		if len(recipientMap) == 0 {
			return
		}

		recipients := make([]string, 0, len(recipientMap))
		for id := range recipientMap {
			recipients = append(recipients, id)
		}

		imageURL := ""
		if promo.PromotionType == models.PromotionTypeBundling && promo.PackageImage != nil && *promo.PackageImage != "" {
			imageURL = *promo.PackageImage
		} else if mainProduct != nil && len(mainProduct.ProductPreview) > 0 {
			imageURL = mainProduct.ProductPreview[0]
		}

		title := fmt.Sprintf("Promosi dari Toko %s", businessName)
		body := promo.Caption

		data := map[string]interface{}{
			"action":               "open_promotion",
			"business_id":          businessID,
			"business_name":        businessName,
			"promotion_id":         promo.PromotionID,
			"promotion_type":       promo.PromotionType,
			"package_name":         promo.PackageName,
			"caption":              promo.Caption,
			"image_url":            imageURL,
			"main_product_id":      promo.MainProductID,
			"promotion_created_at": promo.CreatedAt.Format(time.RFC3339),
		}

		notifService.SendNotification(
			recipients,
			actorID,
			"business_promotion",
			promo.PromotionID,
			title,
			body,
			data,
			nil,
		)
	}()
}
