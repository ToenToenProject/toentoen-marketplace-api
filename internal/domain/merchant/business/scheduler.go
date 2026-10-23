package business

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"
)

var (
	storeStatusCacheMu   sync.RWMutex
	lastKnownStoreStatus = make(map[string]bool)
)

// SetLastKnownStoreStatus updates the in-memory cache of store open status
func SetLastKnownStoreStatus(businessID string, isOpen bool) {
	storeStatusCacheMu.Lock()
	defer storeStatusCacheMu.Unlock()
	lastKnownStoreStatus[businessID] = isOpen
}

// StartStoreStatusSyncWorker menjalankan background scheduler setiap 1 menit (sesuai zona waktu WIB)
// untuk mengevaluasi status operasional (buka/tutup) seluruh toko secara real-time berdasarkan jam operasional toko,
// serta menyinkronkan status tersebut ke Meilisearch jika terjadi perubahan status.
func StartStoreStatusSyncWorker(db *gorm.DB) {
	if db == nil {
		return
	}

	go func() {
		// Evaluasi langsung saat worker pertama kali dijalankan
		EvaluateAndSyncAllStoreStatuses(db)

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			EvaluateAndSyncAllStoreStatuses(db)
		}
	}()
}

// EvaluateAndSyncAllStoreStatuses memeriksa semua bisnis aktif dan jam operasionalnya saat ini di WIB.
// Jika status buka/tutup toko berubah (atau belum tersinkron), otomatis update dokumen Meilisearch.
func EvaluateAndSyncAllStoreStatuses(db *gorm.DB) {
	if meilisearch.Client == nil || db == nil {
		return
	}

	var businesses []models.Business
	err := db.Preload("Hours").Where("is_active = ? AND status != ?", true, "deleted").Find(&businesses).Error
	if err != nil || len(businesses) == 0 {
		return
	}

	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)
	currentDayInt := int(now.Weekday())

	bizUpdates := make([]meilisearch.BusinessStatusBatchUpdate, 0)
	var changedBizIDs []string

	storeStatusCacheMu.Lock()
	for i := range businesses {
		b := &businesses[i]
		isOpen := utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)

		lastOpen, tracked := lastKnownStoreStatus[b.BusinessID]
		if !tracked || lastOpen != isOpen {
			lastKnownStoreStatus[b.BusinessID] = isOpen
			changedBizIDs = append(changedBizIDs, b.BusinessID)

			closeTime := ""
			for _, h := range b.Hours {
				if h.DayOfWeek == currentDayInt && h.CloseTime != nil {
					closeTime = *h.CloseTime
					break
				}
			}

			bizUpdates = append(bizUpdates, meilisearch.BusinessStatusBatchUpdate{
				ID:               b.BusinessID,
				IsOpen:           isOpen,
				IsManuallyClosed: b.IsManuallyClosed,
				CloseTime:        closeTime,
			})
		}
	}
	storeStatusCacheMu.Unlock()

	if len(bizUpdates) > 0 {
		if err := meilisearch.BatchUpdateBusinessStatuses(bizUpdates); err != nil {
			log.Printf("⚠️ [StoreStatusWorker] Gagal sync status %d bisnis ke Meilisearch: %v", len(bizUpdates), err)
		} else {
			log.Printf("⏰ [StoreStatusWorker] Berhasil memperbarui status buka/tutup %d bisnis ke Meilisearch", len(bizUpdates))
		}

		// Update juga status is_open produk milik toko-toko yang berubah
		for _, bid := range changedBizIDs {
			_ = meilisearch.UpdateBusinessStatus(db, bid)
		}
	}
}

// StartScheduleReminderWorker menjalankan scheduler berkala setiap pergantian jam (menit ke-00 WIB)
// untuk mengirimkan notifikasi konfirmasi buka, bersambung/24 jam, dan tutup kepada merchant.
func StartScheduleReminderWorker(businessRepo BusinessRepository, notifService notification.NotificationService) {
	go func() {
		for {
			loc := utils.GetWIBLocation()
			now := time.Now().In(loc)

			// Hitung durasi hingga menit ke-00 jam berikutnya
			nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, loc)
			sleepDuration := nextHour.Sub(now)

			time.Sleep(sleepDuration)

			// Waktu saat ini setelah bangun di jam baru
			runTime := time.Now().In(loc)

			// 1. Jika tepat pukul 00:00:00 (Tengah Malam), jalankan auto-close & reset live
			if runTime.Hour() == 0 {
				log.Println("[CRON-MIDNIGHT] Menutup otomatis toko yang lupa dimatikan dan mereset status live...")
				_ = businessRepo.ResetManualCloseStatus()
				_ = businessRepo.ResetLiveStatus()
				EvaluateAndSyncAllStoreStatuses(database.DB)
			}

			// 2. Periksa jadwal toko dan kirim pengingat konfirmasi 1 jam sebelumnya
			processScheduleReminders(runTime, businessRepo, notifService)
		}
	}()
}

func parseHourMinute(timeStr *string) (int, int, bool) {
	if timeStr == nil {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimSpace(*timeStr), ":")
	if len(parts) < 2 {
		return 0, 0, false
	}
	h, errH := strconv.Atoi(parts[0])
	m, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil {
		return 0, 0, false
	}
	return h, m, true
}

func processScheduleReminders(now time.Time, repo BusinessRepository, notifService notification.NotificationService) {
	if notifService == nil {
		return
	}

	businesses, err := repo.GetBusinessesWithHours()
	if err != nil || len(businesses) == 0 {
		return
	}

	currentDay := int(now.Weekday())
	nextDay := (currentDay + 1) % 7
	currentHour := now.Hour()
	targetNextHour := (currentHour + 1) % 24

	for _, b := range businesses {
		if b.AccountID == "" {
			continue
		}

		bName := "Toko Anda"
		if b.BusinessName != nil && strings.TrimSpace(*b.BusinessName) != "" {
			bName = strings.TrimSpace(*b.BusinessName)
		}

		// Cari jadwal hari ini dan jadwal besok
		var todayHour *models.BusinessHour
		var nextDayHour *models.BusinessHour
		for i := range b.Hours {
			if b.Hours[i].DayOfWeek == currentDay {
				todayHour = &b.Hours[i]
			} else if b.Hours[i].DayOfWeek == nextDay {
				nextDayHour = &b.Hours[i]
			}
		}

		// [A] Pengingat Buka 1 Jam Sebelumnya (Poin 4):
		// Target jam buka adalah jam berikutnya (targetNextHour)
		if todayHour != nil {
			openH, openM, ok := parseHourMinute(todayHour.OpenTime)
			if ok && openH == targetNextHour {
				openTimeFormatted := fmt.Sprintf("%02d:%02d", openH, openM)
				title := fmt.Sprintf("⏰ Pengingat Buka Toko: %s", bName)
				body := fmt.Sprintf("Apakah toko %s akan buka 1 jam lagi sesuai jadwal (jam %s)?", bName, openTimeFormatted)

				data := map[string]interface{}{
					"action_type":       "schedule_confirmation",
					"confirmation_type": "confirm_open",
					"business_id":       b.BusinessID,
					"business_name":     bName,
					"target_time":       openTimeFormatted,
					"screen":            "merchant_dashboard",
					"click_action":      "CONFIRM_OPEN_CLICK",
				}

				notifService.SendNotification([]string{b.AccountID}, "system", "schedule_open_reminder", b.BusinessID, title, body, data, nil)
			}
		}

		// [B] Pengingat Bersambung / Toko 24 Jam pada Jam 23:00 WIB (Poin 3):
		// Cek apakah jadwal hari ini bersambung ke hari berikutnya atau 24 jam
		if currentHour == 23 && todayHour != nil {
			isContinuing := false
			targetTomorrowTime := "00:00"

			// Cek apakah 24 jam
			openH, openM, okOpen := parseHourMinute(todayHour.OpenTime)
			closeH, closeM, okClose := parseHourMinute(todayHour.CloseTime)
			if okOpen && okClose {
				if openH == 0 && openM == 0 && (closeH == 24 || (closeH == 23 && closeM >= 59)) {
					isContinuing = true
					targetTomorrowTime = "24 Jam"
				}
				// Cek apakah hari ini tutup jam 24:00 dan besok buka jam 00:00
				if closeH == 24 || (closeH == 23 && closeM >= 59) {
					if nextDayHour != nil {
						nextOpenH, _, nextOk := parseHourMinute(nextDayHour.OpenTime)
						if nextOk && nextOpenH == 0 {
							isContinuing = true
							if nextDayHour.CloseTime != nil {
								targetTomorrowTime = *nextDayHour.CloseTime
							}
						}
					}
				}
			}

			if isContinuing {
				title := fmt.Sprintf("🌙 Konfirmasi Lanjut Buka: %s", bName)
				body := fmt.Sprintf("Apakah toko %s akan lanjut buka setelah jam 24.00 sesuai jadwal toko?", bName)

				data := map[string]interface{}{
					"action_type":       "schedule_confirmation",
					"confirmation_type": "confirm_open",
					"business_id":       b.BusinessID,
					"business_name":     bName,
					"target_time":       targetTomorrowTime,
					"screen":            "merchant_dashboard",
					"click_action":      "CONFIRM_CONTINUE_CLICK",
				}

				notifService.SendNotification([]string{b.AccountID}, "system", "schedule_continue_reminder", b.BusinessID, title, body, data, nil)
			}
		}

		// [C] Pengingat Tutup 1 Jam Sebelumnya (Poin 5a):
		// Target jam tutup reguler adalah jam berikutnya (targetNextHour)
		if todayHour != nil {
			closeH, closeM, ok := parseHourMinute(todayHour.CloseTime)
			if ok && closeH == targetNextHour && closeH != 24 {
				closeTimeFormatted := fmt.Sprintf("%02d:%02d", closeH, closeM)
				title := fmt.Sprintf("⏱️ Pengingat Tutup Toko: %s", bName)
				body := fmt.Sprintf("Apakah toko %s akan tutup sesuai jadwal di jam %s?", bName, closeTimeFormatted)

				data := map[string]interface{}{
					"action_type":       "schedule_confirmation",
					"confirmation_type": "confirm_overtime",
					"business_id":       b.BusinessID,
					"business_name":     bName,
					"target_time":       closeTimeFormatted,
					"screen":            "merchant_dashboard",
					"click_action":      "CONFIRM_OVERTIME_CLICK",
				}

				notifService.SendNotification([]string{b.AccountID}, "system", "schedule_close_reminder", b.BusinessID, title, body, data, nil)
			}
		}
	}
}
