package utils

import (
	"log"
	"strings"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

// CalculateStoreStatus menghitung apakah toko buka SECARA REAL TIME
func CalculateStoreStatus(isManuallyClosed bool, hours []models.BusinessHour) bool {
	// 1. Filter Manual: Jika owner menekan tombol tutup darurat, langsung kembalikan FALSE
	if isManuallyClosed {
		return false
	}

	// 2. Kunci Zona Waktu ke WIB (GMT+7)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Println("Gagal load timezone:", err)
		loc = time.FixedZone("WIB", 7*3600) // Fallback manual ke WIB
	}
	
	now := time.Now().In(loc)
	currentDay := int(now.Weekday()) // 0 = Minggu, 1 = Senin, dst.

	// 3. Filter Sistem: Cek jadwal hari ini
	for _, h := range hours {
		if h.DayOfWeek == currentDay {
			// Parse OpenTime dan CloseTime (Format dari DB biasanya "15:00:00" atau "15:00")
			if h.OpenTime != nil && h.CloseTime != nil {
				openStr := strings.Split(*h.OpenTime, ":")
				closeStr := strings.Split(*h.CloseTime, ":")
				
				if len(openStr) >= 2 && len(closeStr) >= 2 {
					openHour, _ := time.Parse("15", openStr[0])
					openMinute, _ := time.Parse("04", openStr[1])
					
					closeHour, _ := time.Parse("15", closeStr[0])
					closeMinute, _ := time.Parse("04", closeStr[1])

					// Konversi jam ke menit total (misal 15:30 -> 930 menit) agar mudah dibandingkan
					nowMinutes := now.Hour()*60 + now.Minute()
					openTotalMinutes := openHour.Hour()*60 + openMinute.Minute()
					closeTotalMinutes := closeHour.Hour()*60 + closeMinute.Minute()

					// Pengecekan krusial
					return nowMinutes >= openTotalMinutes && nowMinutes <= closeTotalMinutes
				}
			}
		}
	}

	// Jika tidak ada data hari ini, anggap tutup
	return false 
}