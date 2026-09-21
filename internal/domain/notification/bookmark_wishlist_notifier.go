package notification

import (
	"fmt"
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// NotifyBusinessPostCreated mengirimkan notifikasi ke seluruh customer yang mem-bookmark toko
// ketika toko menerbitkan postingan (feed) baru.
func NotifyBusinessPostCreated(
	db *gorm.DB,
	notifService NotificationService,
	businessID string,
	postID string,
	postContent string,
	actorID string,
) {
	if db == nil || notifService == nil || businessID == "" || postID == "" {
		return
	}

	go func() {
		// 1. Ambil nama toko
		var biz models.Business
		if err := db.Select("business_name").Where("business_id = ?", businessID).First(&biz).Error; err != nil {
			return
		}
		businessName := "Toko"
		if biz.BusinessName != nil && strings.TrimSpace(*biz.BusinessName) != "" {
			businessName = strings.TrimSpace(*biz.BusinessName)
		}

		// 2. Ambil seluruh customer yang mem-bookmark toko ini
		var bookmarkAccountIDs []string
		if err := db.Table("business_bookmarks").
			Where("business_id = ?", businessID).
			Pluck("account_id", &bookmarkAccountIDs).Error; err != nil || len(bookmarkAccountIDs) == 0 {
			return
		}

		// Saring actorID jika ada
		var recipients []string
		for _, accID := range bookmarkAccountIDs {
			if accID != actorID {
				recipients = append(recipients, accID)
			}
		}
		if len(recipients) == 0 {
			return
		}

		// Cuplikan teks konten postingan
		cleanContent := strings.TrimSpace(postContent)
		snippet := cleanContent
		runes := []rune(cleanContent)
		if len(runes) > 60 {
			snippet = string(runes[:60]) + "..."
		}

		title := fmt.Sprintf("📢 Postingan Baru dari %s", businessName)
		body := fmt.Sprintf("%s baru saja membagikan postingan baru. Ketuk untuk melihat!", businessName)
		if snippet != "" {
			body = fmt.Sprintf("%s: \"%s\"", businessName, snippet)
		}

		data := map[string]interface{}{
			"business_id":   businessID,
			"business_name": businessName,
			"post_id":       postID,
			"action":        "open_post",
		}

		notifService.SendNotification(
			recipients,
			actorID,
			"business_new_post",
			postID,
			title,
			body,
			data,
			nil,
		)
	}()
}

// NotifyBusinessPromo mengirimkan notifikasi promo berdasarkan aturan Wishlist vs Bookmark:
// 1. Promo 1 Produk:
//    - User yang memiliki produk di wishlist HANYA menerima notif promo produk wishlist (notif promo toko dilewati).
//    - User yang hanya mem-bookmark toko (tanpa wishlist produk tsb) menerima notif promo toko.
// 2. Promo Banyak Produk Sekaligus:
//    - Seluruh user yang mem-bookmark toko menerima notif promo toko.
//    - User yang memiliki salah satu/lebih produk tsb di wishlist JUGA menerima notif wishlist spesifik untuk highlight.
func NotifyBusinessPromo(
	db *gorm.DB,
	notifService NotificationService,
	businessID string,
	productIDs []string,
	actorID string,
) {
	if db == nil || notifService == nil || businessID == "" || len(productIDs) == 0 {
		return
	}

	go func() {
		// 1. Ambil nama toko
		var biz models.Business
		if err := db.Select("business_name").Where("business_id = ?", businessID).First(&biz).Error; err != nil {
			return
		}
		businessName := "Toko"
		if biz.BusinessName != nil && strings.TrimSpace(*biz.BusinessName) != "" {
			businessName = strings.TrimSpace(*biz.BusinessName)
		}

		// 2. Ambil seluruh customer yang mem-bookmark toko ini
		var rawBookmarkIDs []string
		_ = db.Table("business_bookmarks").
			Where("business_id = ?", businessID).
			Pluck("account_id", &rawBookmarkIDs).Error

		var bookmarkAccountIDs []string
		for _, id := range rawBookmarkIDs {
			if id != actorID {
				bookmarkAccountIDs = append(bookmarkAccountIDs, id)
			}
		}

		// KASUS 1: Promo diadakan untuk 1 produk saja
		if len(productIDs) == 1 {
			targetProductID := productIDs[0]

			// Ambil nama produk
			var prod models.Product
			prodName := "Produk Pilihan"
			if err := db.Select("product_name").Where("product_id = ?", targetProductID).First(&prod).Error; err == nil {
				if prod.ProductName != "" {
					prodName = prod.ProductName
				}
			}

			// Ambil seluruh user yang me-wishlist produk ini
			var rawWishlistIDs []string
			_ = db.Table("product_wishlists").
				Where("product_id = ?", targetProductID).
				Pluck("account_id", &rawWishlistIDs).Error

			wishlistMap := make(map[string]bool)
			var wishlistRecipients []string
			for _, id := range rawWishlistIDs {
				if id != actorID {
					wishlistMap[id] = true
					wishlistRecipients = append(wishlistRecipients, id)
				}
			}

			// 🎯 Kirim notifikasi Promo Produk Wishlist ke user yang me-wishlist produk ini
			if len(wishlistRecipients) > 0 {
				wishlistTitle := "🏷️ Produk di Wishlist Anda Sedang Promo!"
				wishlistBody := fmt.Sprintf("Kabar gembira! Produk \"%s\" di toko %s sekarang sedang promo diskon. Dapatkan sekarang sebelum kehabisan!", prodName, businessName)
				wishlistData := map[string]interface{}{
					"business_id":   businessID,
					"business_name": businessName,
					"product_id":    targetProductID,
					"product_name":  prodName,
					"action":        "open_product",
				}

				notifService.SendNotification(
					wishlistRecipients,
					actorID,
					"product_wishlist_promo",
					targetProductID,
					wishlistTitle,
					wishlistBody,
					wishlistData,
					nil,
				)
			}

			// 🎯 Saring user bookmark: yang SUDAH dapat notif wishlist TIDAK dikirimkan notif promo toko
			var onlyBookmarkRecipients []string
			for _, bID := range bookmarkAccountIDs {
				if !wishlistMap[bID] {
					onlyBookmarkRecipients = append(onlyBookmarkRecipients, bID)
				}
			}

			if len(onlyBookmarkRecipients) > 0 {
				storeTitle := fmt.Sprintf("🎉 Toko %s Sedang Mengadakan Promo!", businessName)
				storeBody := fmt.Sprintf("Toko %s sedang memberikan penawaran diskon spesial untuk produknya. Kunjungi tokonya sekarang!", businessName)
				storeData := map[string]interface{}{
					"business_id":   businessID,
					"business_name": businessName,
					"product_id":    targetProductID,
					"action":        "open_business",
				}

				notifService.SendNotification(
					onlyBookmarkRecipients,
					actorID,
					"business_promo",
					targetProductID,
					storeTitle,
					storeBody,
					storeData,
					nil,
				)
			}

			return
		}

		// KASUS 2: Promo diadakan untuk beberapa produk sekaligus
		// 1. Seluruh customer yang mem-bookmark toko menerima Notif Promo Bisnis
		if len(bookmarkAccountIDs) > 0 {
			storeTitle := fmt.Sprintf("🎉 Toko %s Mengadakan Promo Spesial!", businessName)
			storeBody := fmt.Sprintf("Toko %s sedang mengadakan promo diskon untuk berbagai produk pilihan. Kunjungi tokonya sekarang!", businessName)
			storeData := map[string]interface{}{
				"business_id":   businessID,
				"business_name": businessName,
				"action":        "open_business",
			}

			notifService.SendNotification(
				bookmarkAccountIDs,
				actorID,
				"business_promo",
				businessID,
				storeTitle,
				storeBody,
				storeData,
				nil,
			)
		}

		// 2. Cari customer yang memiliki salah satu/lebih dari produk tersebut di wishlist mereka
		type WishlistRow struct {
			AccountID   string `gorm:"column:account_id"`
			ProductID   string `gorm:"column:product_id"`
			ProductName string `gorm:"column:product_name"`
		}
		var rows []WishlistRow
		err := db.Table("product_wishlists").
			Select("product_wishlists.account_id, product_wishlists.product_id, products.product_name").
			Joins("JOIN products ON products.product_id = product_wishlists.product_id").
			Where("product_wishlists.product_id IN (?)", productIDs).
			Scan(&rows).Error

		if err == nil && len(rows) > 0 {
			userProducts := make(map[string][]string)
			for _, r := range rows {
				if r.AccountID != actorID {
					userProducts[r.AccountID] = append(userProducts[r.AccountID], r.ProductName)
				}
			}

			// Kirim notifikasi wishlist highlight ke masing-masing pengguna
			for uID, names := range userProducts {
				previewNames := strings.Join(names, ", ")
				if len(names) > 2 {
					previewNames = fmt.Sprintf("%s, %s, dan %d produk lainnya", names[0], names[1], len(names)-2)
				}

				wTitle := fmt.Sprintf("🏷️ Produk Wishlist Anda Sedang Promo di %s!", businessName)
				wBody := fmt.Sprintf("Produk favorit di wishlist Anda (%s) sedang promo di toko %s. Segera dapatkan penawaran spesial ini!", previewNames, businessName)
				wData := map[string]interface{}{
					"business_id":   businessID,
					"business_name": businessName,
					"action":        "open_wishlist",
				}

				notifService.SendNotification(
					[]string{uID},
					actorID,
					"product_wishlist_promo",
					businessID,
					wTitle,
					wBody,
					wData,
					nil,
				)
			}
		}
	}()
}
