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
// confirmedDates (opsional): [confirmedOpenDate, confirmedOvertimeDate] dalam format string YYYY-MM-DD
func CalculateStoreStatus(isManuallyClosed bool, hours []models.BusinessHour, confirmedDates ...*string) bool {
	// 1. Filter Manual: Jika owner menekan tombol tutup darurat/manual, langsung kembalikan FALSE
	if isManuallyClosed {
		return false
	}

	// 2. Kunci Zona Waktu ke WIB (GMT+7)
	loc := GetWIBLocation()
	now := time.Now().In(loc)
	todayStr := now.Format("2006-01-02")
	currentDay := int(now.Weekday()) // 0 = Minggu, 1 = Senin, ..., 6 = Sabtu

	var confirmedOpenDate *string
	var confirmedOvertimeDate *string
	if len(confirmedDates) > 0 {
		confirmedOpenDate = confirmedDates[0]
	}
	if len(confirmedDates) > 1 {
		confirmedOvertimeDate = confirmedDates[1]
	}

	isOvertimeConfirmedToday := confirmedOvertimeDate != nil && strings.HasPrefix(strings.TrimSpace(*confirmedOvertimeDate), todayStr)
	isOpenConfirmedToday := (confirmedOpenDate != nil && strings.HasPrefix(strings.TrimSpace(*confirmedOpenDate), todayStr)) || isOvertimeConfirmedToday

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

						// Belum masuk jam buka hari ini (izinkan jika merchant sengaja mengaktifkan buka di luar jam reguler)
						if nowMinutes < openTotalMinutes {
							return isOvertimeConfirmedToday
						}

						// Toko harus sudah dikonfirmasi buka untuk hari ini
						if !isOpenConfirmedToday {
							return false
						}

						// Toko buka 24 jam (misal 00:00 - 24:00 atau 00:00 - 23:59)
						if openTotalMinutes == 0 && (closeTotalMinutes == 0 || closeTotalMinutes >= 1439) {
							return true
						}

						// Jadwal reguler (misal 08:00 - 21:00)
						if openTotalMinutes <= closeTotalMinutes {
							if nowMinutes <= closeTotalMinutes {
								return true
							}
							// Jika sudah lewat jam tutup reguler, hanya buka jika dikonfirmasi lembur (maksimal tengah malam)
							return isOvertimeConfirmedToday
						}

						// Jadwal lintas tengah malam (misal 18:00 - 02:00)
						if nowMinutes >= openTotalMinutes || nowMinutes <= closeTotalMinutes {
							return true
						}
						return isOvertimeConfirmedToday
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

	return false
}