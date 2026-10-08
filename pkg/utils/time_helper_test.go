package utils

import (
	"testing"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

func TestGetWIBLocation(t *testing.T) {
	loc := GetWIBLocation()
	if loc == nil {
		t.Fatal("Expected non-nil WIB location")
	}

	// Pastikan offset zona waktu adalah +7 jam (25200 detik)
	now := time.Now().In(loc)
	_, offset := now.Zone()
	if offset != 7*3600 {
		t.Errorf("Expected offset +0700 (25200 seconds), got %d seconds", offset)
	}
}

func TestCalculateStoreStatus(t *testing.T) {
	loc := GetWIBLocation()
	now := time.Now().In(loc)
	currentDay := int(now.Weekday())

	// Format jam buka dan tutup hari ini di sekitar jam saat ini
	currentH := now.Hour()
	currentM := now.Minute()

	openTimeActive := "00:00:00"
	closeTimeActive := "23:59:59"

	openTimeClosed := "01:00:00"
	closeTimeClosed := "01:05:00"
	if currentH == 1 && currentM <= 5 {
		openTimeClosed = "02:00:00"
		closeTimeClosed = "02:05:00"
	}

	t.Run("Toko Buka Normal (Jadwal 00:00 - 23:59)", func(t *testing.T) {
		hours := []models.BusinessHour{
			{DayOfWeek: currentDay, OpenTime: &openTimeActive, CloseTime: &closeTimeActive},
		}
		if !CalculateStoreStatus(false, hours) {
			t.Errorf("Expected store to be open, got closed")
		}
	})

	t.Run("Toko Tutup Manual Walaupun Jam Operasional Sedang Aktif", func(t *testing.T) {
		hours := []models.BusinessHour{
			{DayOfWeek: currentDay, OpenTime: &openTimeActive, CloseTime: &closeTimeActive},
		}
		if CalculateStoreStatus(true, hours) {
			t.Errorf("Expected store to be closed due to isManuallyClosed=true, got open")
		}
	})

	t.Run("Toko di Luar Jam Operasional", func(t *testing.T) {
		hours := []models.BusinessHour{
			{DayOfWeek: currentDay, OpenTime: &openTimeClosed, CloseTime: &closeTimeClosed},
		}
		if CalculateStoreStatus(false, hours) {
			t.Errorf("Expected store to be closed outside hours, got open")
		}
	})

	t.Run("Format Tanpa Detik (00:00 dan 23:59)", func(t *testing.T) {
		openShort := "00:00"
		closeShort := "23:59"
		hours := []models.BusinessHour{
			{DayOfWeek: currentDay, OpenTime: &openShort, CloseTime: &closeShort},
		}
		if !CalculateStoreStatus(false, hours) {
			t.Errorf("Expected store to be open with HH:mm format, got closed")
		}
	})

	t.Run("Tidak Ada Jadwal Hari Ini Tapi Toggle Buka", func(t *testing.T) {
		otherDay := (currentDay + 1) % 7
		hours := []models.BusinessHour{
			{DayOfWeek: otherDay, OpenTime: &openTimeActive, CloseTime: &closeTimeActive},
		}
		if !CalculateStoreStatus(false, hours) {
			t.Errorf("Expected store without today's schedule to be open when isManuallyClosed=false")
		}
	})

	t.Run("Tidak Ada Jadwal Hari Ini dan Toggle Tutup", func(t *testing.T) {
		otherDay := (currentDay + 1) % 7
		hours := []models.BusinessHour{
			{DayOfWeek: otherDay, OpenTime: &openTimeActive, CloseTime: &closeTimeActive},
		}
		if CalculateStoreStatus(true, hours) {
			t.Errorf("Expected store without today's schedule to be closed when isManuallyClosed=true")
		}
	})
}
