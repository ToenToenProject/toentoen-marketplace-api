package notification

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type NotificationService interface {
	SendNotification(recipientIDs []string, triggerActorID string, notifType, referenceID, defaultTitle, defaultBody string, data map[string]interface{}, fcmTokensMap map[string][]string)
	SendWelcomeMerchantNotification(accountID string)
	GetMyNotifications(accountID string, page, limit int) ([]models.NotificationResponse, int64, error)
	GetUnreadCount(accountID string) (int64, error)
	MarkAsRead(accountID, notifID string) error
	MarkAllAsRead(accountID string) error
	DeleteNotification(accountID, notifID string) error
	DeleteAllNotifications(accountID string) error
	SendPushToAccount(accountID string, title, body string, data map[string]string) error
	SendPushToAccounts(accountIDs []string, title, body string, data map[string]string) error
}

type notificationService struct {
	repo NotificationRepository
	hub  *realtime.Hub
}

func NewNotificationService(repo NotificationRepository, hub *realtime.Hub) NotificationService {
	return &notificationService{
		repo: repo,
		hub:  hub,
	}
}

func (s *notificationService) SendNotification(recipientIDs []string, triggerActorID string, notifType, referenceID, defaultTitle, defaultBody string, data map[string]interface{}, fcmTokensMap map[string][]string) {
	dataBytes, _ := json.Marshal(data)

	for _, recipientID := range recipientIDs {
		// Jangan kirim notifikasi ke diri sendiri (misal owner yang melike produknya sendiri)
		if recipientID == triggerActorID {
			continue
		}

		var notifToSave *models.UserNotification
		var isNew bool

		// 1. Cek apakah ada notifikasi unread dengan tipe dan referenceID yang sama (AGREGASI)
		existingNotif, err := s.repo.GetUnreadNotificationByRef(recipientID, notifType, referenceID)
		if err == nil && existingNotif != nil {
			// Parse actor_ids existing
			var actors []string
			_ = json.Unmarshal(existingNotif.ActorIDs, &actors)
			
			// Cek apakah actor sudah ada
			actorExists := false
			for _, a := range actors {
				if a == triggerActorID {
					actorExists = true
					break
				}
			}

			if !actorExists {
				actors = append(actors, triggerActorID)
				actorsBytes, _ := json.Marshal(actors)
				existingNotif.ActorIDs = actorsBytes
				existingNotif.UpdatedAt = time.Now()
				
				// Ubah title & body berdasarkan jumlah actor (contoh khusus product_like)
				if notifType == "product_like" {
					if len(actors) == 2 {
						existingNotif.Title = "Produk Anda disukai!"
						existingNotif.Body = fmt.Sprintf("Seorang pengguna dan 1 lainnya menyukai produk Anda.") // Idealnya fetch nama dari DB, tapi kita pakai teks generic jika tidak ada data nama
						
						// Jika di Data ada informasi nama user, kita bisa parse
						if triggerName, ok := data["trigger_name"].(string); ok {
							existingNotif.Body = fmt.Sprintf("%s dan 1 lainnya menyukai produk Anda.", triggerName)
						}
					} else if len(actors) > 2 {
						existingNotif.Title = "Banyak yang menyukai produk Anda!"
						if triggerName, ok := data["trigger_name"].(string); ok {
							existingNotif.Body = fmt.Sprintf("%s dan %d lainnya menyukai produk Anda.", triggerName, len(actors)-1)
						} else {
							existingNotif.Body = fmt.Sprintf("Seorang pengguna dan %d lainnya menyukai produk Anda.", len(actors)-1)
						}
					}
				}
				
				// Simpan pembaruan
				_ = s.repo.UpdateNotification(existingNotif)
			}
			notifToSave = existingNotif
			isNew = false
		} else {
			// Buat Notifikasi Baru
			actors := []string{triggerActorID}
			actorsBytes, _ := json.Marshal(actors)
			
			bodyStr := defaultBody
			if triggerName, ok := data["trigger_name"].(string); ok {
				if notifType == "product_like" {
					bodyStr = fmt.Sprintf("%s menyukai produk Anda.", triggerName)
				}
			}

			newNotif := &models.UserNotification{
				ID:                 utils.GenerateUUID("NOTIF"),
				RecipientAccountID: recipientID,
				ActorIDs:           actorsBytes,
				Type:               notifType,
				Title:              defaultTitle,
				Body:               bodyStr,
				Data:               dataBytes,
				IsRead:             false,
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			if referenceID != "" {
				newNotif.ReferenceID = &referenceID
			}

			if err := s.repo.CreateNotification(newNotif); err != nil {
				log.Printf("❌ Failed to create notification for %s: %v", recipientID, err)
			}
			notifToSave = newNotif
			isNew = true
		}

		// 2. Kirim via WebSocket jika Online
		isOnline := s.hub != nil && s.hub.IsOnline(recipientID)
		if isOnline {
			event := realtime.WSEvent{
				Type: "new_notification",
				Data: notifToSave, // Send the updated or new notification
			}
			s.hub.SendToUser(recipientID, event)
		}

		// 3. Push Notification via FCM (dikirim jika user offline atau notifikasi baru)
		if isNew || notifType == "product_like" || !isOnline {
			tokens := fcmTokensMap[recipientID]
			if len(tokens) == 0 {
				tokens, _ = s.repo.GetFCMTokensByAccountID(recipientID)
			}
			if len(tokens) > 0 {
				stringData := make(map[string]string)
				for k, v := range data {
					stringData[k] = fmt.Sprintf("%v", v)
				}
				stringData["notification_id"] = notifToSave.ID
				stringData["type"] = notifType

				_ = config.SendPushNotification(tokens, notifToSave.Title, notifToSave.Body, stringData)
			}
		}
	}
}

func (s *notificationService) GetMyNotifications(accountID string, page, limit int) ([]models.NotificationResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	notifs, total, err := s.repo.GetMyNotifications(accountID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var responses []models.NotificationResponse
	for _, n := range notifs {
		var dataMap map[string]interface{}
		_ = json.Unmarshal(n.Data, &dataMap)
		
		var actors []string
		_ = json.Unmarshal(n.ActorIDs, &actors)

		resp := models.NotificationResponse{
			NotificationID: n.ID,
			Type:           n.Type,
			Title:          n.Title,
			Body:           n.Body,
			IsRead:         n.IsRead,
			Data:           dataMap,
			ActorCount:     len(actors),
			CreatedAt:      n.CreatedAt,
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

func (s *notificationService) GetUnreadCount(accountID string) (int64, error) {
	return s.repo.GetUnreadCount(accountID)
}

func (s *notificationService) MarkAsRead(accountID, notifID string) error {
	if strings.ToLower(notifID) == "all" {
		return s.repo.MarkAllAsRead(accountID)
	}
	return s.repo.MarkAsRead(accountID, notifID)
}

func (s *notificationService) MarkAllAsRead(accountID string) error {
	return s.repo.MarkAllAsRead(accountID)
}

func (s *notificationService) DeleteNotification(accountID, notifID string) error {
	if strings.ToLower(notifID) == "all" || strings.ToLower(notifID) == "clear-all" {
		return s.repo.DeleteAllNotifications(accountID)
	}
	return s.repo.DeleteNotification(accountID, notifID)
}

func (s *notificationService) DeleteAllNotifications(accountID string) error {
	return s.repo.DeleteAllNotifications(accountID)
}

func (s *notificationService) SendWelcomeMerchantNotification(accountID string) {
	if accountID == "" {
		return
	}

	// 1. Cek idempotensi: pastikan belum pernah menerima notifikasi welcome
	hasSent, err := s.repo.HasNotificationOfType(accountID, "welcome_merchant")
	if err != nil || hasSent {
		return
	}

	title := "Selamat datang di Toentoen!"
	body := "Kembangkan bisnis kamu dengan Toentoen!\n\nSebagai tanda terimakasih karena sudah menjadi pengguna awal kami, kamu dapat menikmati Fitur Premium Toentoen Plus dalam paket Growth selama 3 bulan kedepan. (Detilnya dapat dilihat di Toentoen Plus)\n\nKami harap kamu dapat menikmati dan merasakan manfaat dari aplikasi kami.\n\nKami sangat terbuka untuk segala masukkan, sangat diharapkan kamu dapat memberikan saran - saran pada kami di fitur Feedback dalam aplikasi atau melalui saluran bantuan kami.\n\nTerima kasih!"

	data := map[string]interface{}{
		"type":     "welcome_merchant",
		"logo_url": "/assets/logo/toentoen-logo.png",
		"action":   "open_welcome_screen",
	}
	dataBytes, _ := json.Marshal(data)
	actorsBytes, _ := json.Marshal([]string{"SYSTEM"})

	newNotif := &models.UserNotification{
		ID:                 utils.GenerateUUID("NOTIF"),
		RecipientAccountID: accountID,
		ActorIDs:           actorsBytes,
		Type:               "welcome_merchant",
		Title:              title,
		Body:               body,
		Data:               dataBytes,
		IsRead:             false,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.CreateNotification(newNotif); err != nil {
		log.Printf("❌ Failed to create welcome notification for %s: %v", accountID, err)
		return
	}

	// WebSocket broadcast jika merchant sedang online
	if s.hub.IsOnline(accountID) {
		event := realtime.WSEvent{
			Type: "new_notification",
			Data: newNotif,
		}
		s.hub.SendToUser(accountID, event)
	}

	// Push notification via FCM
	tokens, _ := s.repo.GetFCMTokensByAccountID(accountID)
	if len(tokens) > 0 {
		stringData := map[string]string{
			"notification_id": newNotif.ID,
			"type":            "welcome_merchant",
			"logo_url":        "/assets/logo/toentoen-logo.png",
			"action":          "open_welcome_screen",
		}
		_ = config.SendPushNotification(tokens, newNotif.Title, newNotif.Body, stringData)
	}
}

func (s *notificationService) SendPushToAccount(accountID string, title, body string, data map[string]string) error {
	cleanID := strings.TrimSpace(accountID)
	if cleanID == "" {
		return nil
	}
	tokens, err := s.repo.GetFCMTokensByAccountID(cleanID)
	if err != nil || len(tokens) == 0 {
		return nil
	}
	return config.SendPushNotification(tokens, title, body, data)
}

func (s *notificationService) SendPushToAccounts(accountIDs []string, title, body string, data map[string]string) error {
	for _, id := range accountIDs {
		_ = s.SendPushToAccount(id, title, body, data)
	}
	return nil
}


