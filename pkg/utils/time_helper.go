package utils

import (
	"strconv"
	"strings"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

var defaultWIBLocation = time.FixedZone("WIB", 7*3600)

// GetWIBLocation mengembalikan pointer ke *time.Location untuk zona Asia/Jakarta (WIB / UTC+7).
// Jika sistem OS tidak memiliki tzdata, fungsi ini secara aman fallback ke FixedZone ("WIB", +7 jam).
func GetWIBLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil || loc == nil {
		return defaultWIBLocation
	}
	return loc
}

// CalculateStoreStatus menghitung apakah toko buka SECARA REAL TIME terkunci pada zona WIB (UTC+7)
func CalculateStoreStatus(isManuallyClosed bool, hours []models.BusinessHour) bool {
	// 1. Filter Manual: Jika owner menekan tombol tutup darurat/manual, langsung kembalikan FALSE
	if isManuallyClosed {
		return false
	}

	// 2. Kunci Zona Waktu ke WIB (GMT+7)
	loc := GetWIBLocation()
	now := time.Now().In(loc)
	currentDay := int(now.Weekday()) // 0 = Minggu, 1 = Senin, ..., 6 = Sabtu

	nowMinutes := now.Hour()*60 + now.Minute()

	// 3. Filter Sistem: Cek jadwal operasional hari ini
	hasScheduleToday := false
	for _, h := range hours {
		if h.DayOfWeek == currentDay {
			// Parse OpenTime dan CloseTime (Format dari DB bisa "08:00:00" atau "08:00")
			if h.OpenTime != nil && h.CloseTime != nil {
				hasScheduleToday = true
				openStr := strings.Split(strings.TrimSpace(*h.OpenTime), ":")
				closeStr := strings.Split(strings.TrimSpace(*h.CloseTime), ":")

				if len(openStr) >= 2 && len(closeStr) >= 2 {
					openHour, errOH := strconv.Atoi(openStr[0])
					openMinute, errOM := strconv.Atoi(openStr[1])
					closeHour, errCH := strconv.Atoi(closeStr[0])
					closeMinute, errCM := strconv.Atoi(closeStr[1])

					if errOH == nil && errOM == nil && errCH == nil && errCM == nil {
						openTotalMinutes := openHour*60 + openMinute
						closeTotalMinutes := closeHour*60 + closeMinute

						// Toko buka 24 jam (misal 00:00 - 24:00 atau 00:00 - 23:59)
						if openTotalMinutes == 0 && (closeTotalMinutes == 0 || closeTotalMinutes >= 1439) {
							return true
						}

						// Jadwal operasional reguler di hari yang sama (misal 08:00 - 17:00)
						if openTotalMinutes <= closeTotalMinutes {
							return nowMinutes >= openTotalMinutes && nowMinutes <= closeTotalMinutes
						}

						// Jadwal lintas tengah malam (misal 18:00 - 02:00 dini hari)
						return nowMinutes >= openTotalMinutes || nowMinutes <= closeTotalMinutes
					}
				}
			}
		}
	}

	// Jika toko TIDAK mengatur jadwal hari ini, status buka ditentukan oleh toggle manual:
	// Jika !isManuallyClosed (toggle dinyalakan), toko buka dan akan otomatis ditutup saat reset tengah malam (00:00).
	if !hasScheduleToday {
		return !isManuallyClosed
	}

	// Jika ada jadwal teratur tetapi sekarang di luar jam buka, toko tutup
	return false
}