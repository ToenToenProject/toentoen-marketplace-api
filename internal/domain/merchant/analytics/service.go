package analytics

import (
	"errors"
	"fmt"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/merchant/catalog"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type AnalyticsService interface {
	GetDailyAnalytics(requesterID, role, businessID string) (*models.DailyAnalyticsResponse, error)
	GetHistoryAnalytics(requesterID, role, businessID, startDate, endDate string) (*models.HistoryAnalyticsResponse, error)
	GetExploreAnalytics(requesterID, role, businessID, keyword string) (*models.ExploreAnalyticsResponse, error)

	// Event Tracking (dipanggil dari customer domain, fire-and-forget)
	TrackVisitor(businessID, accountID, sourceType, searchedTag, productID string)
	TrackImpressions(impressions []ImpressionItem)
}

// ImpressionItem digunakan untuk batch impression dari Flutter
type ImpressionItem struct {
	BusinessID string `json:"business_id"`
	Source     string `json:"source"`
}

type analyticsService struct {
	analyticsRepo AnalyticsRepository
	productRepo   catalog.ProductRepository
}

func NewAnalyticsService(ar AnalyticsRepository, pr catalog.ProductRepository) AnalyticsService {
	return &analyticsService{ar, pr}
}

func (s *analyticsService) GetDailyAnalytics(requesterID, role, businessID string) (*models.DailyAnalyticsResponse, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke analitik toko ini")
	}
	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermViewAnalyticsDaily) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk melihat analitik harian toko")
	}

	limit := 6
	loc := utils.GetWIBLocation()
	today := time.Now().In(loc).Format("2006-01-02")

	response, err := s.analyticsRepo.GetDailyAnalytics(businessID, today, limit)
	if err != nil {
		return nil, err
	}

	// Isi signal stats untuk hari ini
	received, responded, _ := s.analyticsRepo.GetSignalStats(businessID, today, today)
	response.SignalReceived = received
	response.SignalResponded = responded

	return response, nil
}

func (s *analyticsService) GetHistoryAnalytics(requesterID, role, businessID, startDate, endDate string) (*models.HistoryAnalyticsResponse, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke analitik toko ini")
	}
	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermViewAnalyticsHistory) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk melihat riwayat analitik toko")
	}

	// Validasi format tanggal dan urutan
	layout := "2006-01-02"
	startTime, errStart := time.Parse(layout, startDate)
	endTime, errEnd := time.Parse(layout, endDate)
	if errStart != nil || errEnd != nil {
		return nil, errors.New("format tanggal tidak valid, gunakan format YYYY-MM-DD")
	}
	if startTime.After(endTime) {
		return nil, errors.New("start_date tidak boleh lebih besar dari end_date")
	}

	limit := 6

	response, err := s.analyticsRepo.GetHistoryDetails(businessID, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}

	currentVisitors, currentImpressions, _ := s.analyticsRepo.GetBasicStatsByDateRange(businessID, startDate, endDate)
	response.TotalVisitors = currentVisitors
	response.TotalImpression = currentImpressions

	// Hitung persentase pertumbuhan
	daysDiff := int(endTime.Sub(startTime).Hours() / 24)
	prevEndTime := startTime.AddDate(0, 0, -1)
	prevStartTime := prevEndTime.AddDate(0, 0, -daysDiff)

	prevVisitors, prevImpressions, _ := s.analyticsRepo.GetBasicStatsByDateRange(
		businessID, prevStartTime.Format(layout), prevEndTime.Format(layout))

	calculateGrowth := func(current, previous int64) float64 {
		if previous == 0 {
			if current > 0 {
				return 100.0
			}
			return 0.0
		}
		return float64(current-previous) / float64(previous) * 100.0
	}

	response.VisitorsGrowth = calculateGrowth(currentVisitors, prevVisitors)
	response.ImpressionGrowth = calculateGrowth(currentImpressions, prevImpressions)

	// Hitung signal stats untuk rentang waktu
	currentSignals, currentResponded, _ := s.analyticsRepo.GetSignalStats(businessID, startDate, endDate)
	prevSignals, prevResponded, _ := s.analyticsRepo.GetSignalStats(
		businessID, prevStartTime.Format(layout), prevEndTime.Format(layout))

	response.TotalSignalReceived = currentSignals
	response.TotalSignalResponded = currentResponded
	response.SignalReceivedGrowth = calculateGrowth(currentSignals, prevSignals)
	response.SignalRespondedGrowth = calculateGrowth(currentResponded, prevResponded)

	// Conversion rate: berapa persen dari impression yang menjadi visitor
	if currentImpressions > 0 {
		response.ConversionRate = fmt.Sprintf("%.1f%%", float64(currentVisitors)/float64(currentImpressions)*100)
	} else {
		response.ConversionRate = "0.0%"
	}

	// TrafficGrowth = rata-rata growth visitor & impression (lebih bermakna dari sekedar copy visitors_growth)
	response.TrafficGrowth = (response.VisitorsGrowth + response.ImpressionGrowth) / 2.0

	return response, nil
}

func (s *analyticsService) GetExploreAnalytics(requesterID, role, businessID, keyword string) (*models.ExploreAnalyticsResponse, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke analitik toko ini")
	}
	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermViewAnalyticsExplore) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk melihat analitik jelajah pasar")
	}

	limit := 5
	radiusKm := 5.0

	return s.analyticsRepo.GetExploreAnalytics(businessID, keyword, radiusKm, limit)
}

// TrackVisitor mencatat event kunjungan ke profil toko — dipanggil async (goroutine)
func (s *analyticsService) TrackVisitor(businessID, accountID, sourceType, searchedTag, productID string) {
	// Anti-Spam / Deduplication: Cegah spam refresh berulang dalam waktu 5 menit untuk akun yang sama
	if accountID != "" && config.RDB != nil {
		dedupKey := fmt.Sprintf("analytics:visit:%s:%s", businessID, accountID)
		isNew, err := config.RDB.SetNX(config.Ctx, dedupKey, 1, 5*time.Minute).Result()
		if err == nil && !isNew {
			// Kunjungan berulang dalam 5 menit dilewati
			return
		}
	}

	var tagPtr *string
	if searchedTag != "" {
		tagPtr = &searchedTag
	}
	var prodPtr *string
	if productID != "" {
		prodPtr = &productID
	}
	// Fire and forget — error diabaikan agar tidak mempengaruhi response customer
	_ = s.analyticsRepo.TrackVisitor(businessID, sourceType, tagPtr, prodPtr)
}

// TrackImpressions mencatat batch impression dari Flutter dalam 1 query batch insert SQL
func (s *analyticsService) TrackImpressions(impressions []ImpressionItem) {
	if len(impressions) == 0 {
		return
	}

	var events []models.MerchantAnalytics
	for _, imp := range impressions {
		if imp.BusinessID == "" {
			continue
		}
		source := imp.Source
		if source == "" {
			source = "explore"
		}
		events = append(events, models.MerchantAnalytics{
			BusinessID: imp.BusinessID,
			EventType:  "impression",
			SourceType: source,
		})
	}

	if len(events) > 0 {
		_ = s.analyticsRepo.TrackImpressionsBatch(events)
	}
}
