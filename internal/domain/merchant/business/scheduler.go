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

// StartStoreStatusSyncWorker runs a background scheduler every minute (in WIB timezone)
// to evaluate real-time store operating hours and synchronize status changes to Meilisearch.
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

// EvaluateAndSyncAllStoreStatuses evaluates operational open/closed status for all active businesses locked to WIB.
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

		for _, bid := range changedBizIDs {
			_ = meilisearch.UpdateBusinessStatus(db, bid)
		}
	}
}

// StartScheduleReminderWorker menjalankan scheduler berkala setiap pergantian jam (menit ke-00 WIB)
// Dispatch opening schedule notifications to merchants.
func StartScheduleReminderWorker(businessRepo BusinessRepository, notifService notification.NotificationService) {
	go func() {
		for {
			loc := utils.GetWIBLocation()
			now := time.Now().In(loc)

			nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, loc)
			sleepDuration := nextHour.Sub(now)

			time.Sleep(sleepDuration)

			// Current timestamp at hourly boundary.
			runTime := time.Now().In(loc)

			// Reset live streaming and execute auto-close at midnight (00:00 WIB).
			if runTime.Hour() == 0 {
				log.Println("[CRON-MIDNIGHT] Menutup otomatis toko yang lupa dimatikan dan mereset status live...")
				_ = businessRepo.ResetManualCloseStatus()
				_ = businessRepo.ResetLiveStatus()
				EvaluateAndSyncAllStoreStatuses(database.DB)
			}

			// Send opening reminder notification 1 hour prior to scheduled open time.
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

		// Look up operational schedules for today and tomorrow.
		var todayHour *models.BusinessHour
		var nextDayHour *models.BusinessHour
		for i := range b.Hours {
			if b.Hours[i].DayOfWeek == currentDay {
				todayHour = &b.Hours[i]
			} else if b.Hours[i].DayOfWeek == nextDay {
				nextDayHour = &b.Hours[i]
			}
		}

		// Opening reminder 1 hour prior to scheduled open time.
		// Target open time is next hour.
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

		// 24-hour schedule reminder at 23:00 WIB.
		if currentHour == 23 && todayHour != nil {
			isContinuing := false
			targetTomorrowTime := "00:00"

			openH, openM, okOpen := parseHourMinute(todayHour.OpenTime)
			closeH, closeM, okClose := parseHourMinute(todayHour.CloseTime)
			if okOpen && okClose {
				if openH == 0 && openM == 0 && (closeH == 24 || (closeH == 23 && closeM >= 59)) {
					isContinuing = true
					targetTomorrowTime = "24 Jam"
				}
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

		// Closing reminder 1 hour prior to regular closing time.
		// Target regular closing time is next hour.
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
