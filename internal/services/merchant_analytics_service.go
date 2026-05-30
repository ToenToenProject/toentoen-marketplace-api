package services

import (
	"errors"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
	"time"
)

type AnalyticsService interface {
	GetDailyAnalytics(requesterID, role, businessID string) (*models.DailyAnalyticsResponse, error)
	GetHistoryAnalytics(requesterID, role, businessID, startDate, endDate string) (*models.HistoryAnalyticsResponse, error)
	GetExploreAnalytics(requesterID, role, businessID, keyword string) (*models.ExploreAnalyticsResponse, error)
}

type analyticsService struct {
	analyticsRepo repositories.AnalyticsRepository
	productRepo   repositories.ProductRepository // Pinjam repo produk untuk fungsi HasAccessToBusiness
}

func NewAnalyticsService(ar repositories.AnalyticsRepository, pr repositories.ProductRepository) AnalyticsService {
	return &analyticsService{ar, pr}
}

func (s *analyticsService) GetDailyAnalytics(requesterID, role, businessID string) (*models.DailyAnalyticsResponse, error) {
	// Cek apakah user yang login berhak melihat data toko ini
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: you don't have access to this business analytics")
	}

	// Limit list tag dan produk (sementara kita set 6 sesuai desain UI)
	limit := 6

	return s.analyticsRepo.GetDailyAnalytics(businessID, limit)
}

func (s *analyticsService) GetHistoryAnalytics(requesterID, role, businessID, startDate, endDate string) (*models.HistoryAnalyticsResponse, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: you don't have access to this business analytics")
	}

	limit := 6 // Sesuai desain UI

	// 1. Ambil detail data untuk rentang waktu yang dipilih
	response, err := s.analyticsRepo.GetHistoryDetails(businessID, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}

	// 2. Ambil angka dasar (Current Period)
	currentVisitors, currentImpressions, _ := s.analyticsRepo.GetBasicStatsByDateRange(businessID, startDate, endDate)
	response.TotalVisitors = currentVisitors
	response.TotalImpression = currentImpressions

	// 3. MAGIC LOGIC: Hitung Persentase Pertumbuhan
	// Parse tanggal string menjadi format Time Golang
	layout := "2006-01-02"
	startTime, _ := time.Parse(layout, startDate)
	endTime, _ := time.Parse(layout, endDate)

	// Hitung durasi hari yang dipilih (misal: pilih 7 hari)
	daysDiff := int(endTime.Sub(startTime).Hours() / 24)

	// Mundurkan tanggal untuk mendapatkan "Previous Period" (7 hari sebelumnya)
	prevEndTime := startTime.AddDate(0, 0, -1)
	prevStartTime := prevEndTime.AddDate(0, 0, -daysDiff)

	// Ambil angka dasar di periode sebelumnya
	prevVisitors, prevImpressions, _ := s.analyticsRepo.GetBasicStatsByDateRange(businessID, prevStartTime.Format(layout), prevEndTime.Format(layout))

	// Fungsi helper untuk menghitung persen
	calculateGrowth := func(current, previous int64) float64 {
		if previous == 0 {
			if current > 0 { return 100.0 } // Jika dulu 0, sekarang ada, berarti naik 100%
			return 0.0
		}
		return float64(current-previous) / float64(previous) * 100.0
	}

	// Masukkan hasil persen ke response
	response.VisitorsGrowth = calculateGrowth(currentVisitors, prevVisitors)
	response.ImpressionGrowth = calculateGrowth(currentImpressions, prevImpressions)

	// Asumsi UI: Traffic Growth sama dengan rata-rata visitors growth (bisa disesuaikan)
	response.TrafficGrowth = response.VisitorsGrowth 

	return response, nil
}

func (s *analyticsService) GetExploreAnalytics(requesterID, role, businessID, keyword string) (*models.ExploreAnalyticsResponse, error) {
	// Cek otoritas
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: you don't have access to this business analytics")
	}

	limit := 5        // Batasi tampilan (misal Top 5 pencarian tertinggi)
	radiusKm := 5.0   // Radius pencarian = 5 Kilometer

	return s.analyticsRepo.GetExploreAnalytics(businessID, keyword, radiusKm, limit)
}