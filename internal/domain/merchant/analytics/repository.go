package analytics

import (
	"sync"
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetDailyAnalytics(businessID, today string, limit int) (*models.DailyAnalyticsResponse, error)
	GetBasicStatsByDateRange(businessID string, startDate, endDate string) (int64, int64, error)
	GetHistoryDetails(businessID string, startDate, endDate string, limit int) (*models.HistoryAnalyticsResponse, error)
	GetExploreAnalytics(businessID, keyword string, radiusKm float64, limit int) (*models.ExploreAnalyticsResponse, error)

	// Event Tracking
	TrackVisitor(businessID, sourceType string, searchedTag *string, productID *string) error
	TrackImpression(businessID, sourceType string) error
	TrackImpressionsBatch(events []models.MerchantAnalytics) error

	// Signal Stats
	GetSignalStats(businessID string, startDate, endDate string) (int64, int64, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db}
}

func (r *analyticsRepository) GetDailyAnalytics(businessID, today string, limit int) (*models.DailyAnalyticsResponse, error) {
	var (
		totalVisitors   int64
		totalImpression int64
		trafficSources  []models.TrafficSourceStat
		visitorByTags   []models.TagStat
		visitorByProd   []models.ProductStat
	)

	var wg sync.WaitGroup
	wg.Add(5)

	// 1. Total Visitors (Parallel)
	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) = ?", businessID, today).
			Count(&totalVisitors)
	}()

	// 2. Total Impression (Parallel)
	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Where("business_id = ? AND event_type = 'impression' AND DATE(created_at) = ?", businessID, today).
			Count(&totalImpression)
	}()

	// 3. Traffic Sources (Parallel)
	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Select("source_type as source, COUNT(log_id) as count").
			Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) = ?", businessID, today).
			Group("source_type").
			Scan(&trafficSources)
	}()

	// 4. Visitor by Tags (Parallel)
	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Select("searched_tag as tag, COUNT(log_id) as count").
			Where("business_id = ? AND event_type = 'visitor' AND source_type = 'tag_search' AND searched_tag IS NOT NULL AND searched_tag != '' AND DATE(created_at) = ?", businessID, today).
			Group("searched_tag").
			Order("count DESC").Limit(limit).
			Scan(&visitorByTags)
	}()

	// 5. Visitor by Products (Parallel)
	go func() {
		defer wg.Done()
		r.db.Table("merchant_analytics").
			Select("products.product_name, COUNT(merchant_analytics.log_id) as count").
			Joins("JOIN products ON merchant_analytics.product_id = products.product_id AND products.deleted_at IS NULL").
			Where("merchant_analytics.business_id = ? AND merchant_analytics.event_type = 'visitor' AND merchant_analytics.source_type = 'product_search' AND DATE(merchant_analytics.created_at) = ?", businessID, today).
			Group("products.product_name").
			Order("count DESC").Limit(limit).
			Scan(&visitorByProd)
	}()

	wg.Wait()

	if trafficSources == nil {
		trafficSources = []models.TrafficSourceStat{}
	}
	if visitorByTags == nil {
		visitorByTags = []models.TagStat{}
	}
	if visitorByProd == nil {
		visitorByProd = []models.ProductStat{}
	}

	return &models.DailyAnalyticsResponse{
		TotalVisitors:   totalVisitors,
		TotalImpression: totalImpression,
		TrafficSources:  trafficSources,
		VisitorByTags:   visitorByTags,
		VisitorByProd:   visitorByProd,
	}, nil
}

// Mengambil angka dasar (Visitors & Impressions) untuk dihitung persentasenya secara paralel
func (r *analyticsRepository) GetBasicStatsByDateRange(businessID string, startDate, endDate string) (int64, int64, error) {
	var visitors, impressions int64
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
			Count(&visitors)
	}()

	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Where("business_id = ? AND event_type = 'impression' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
			Count(&impressions)
	}()

	wg.Wait()
	return visitors, impressions, nil
}

// Mengambil list detail (Traffic, Tags, Products) secara paralel
func (r *analyticsRepository) GetHistoryDetails(businessID string, startDate, endDate string, limit int) (*models.HistoryAnalyticsResponse, error) {
	var (
		trafficSources []models.TrafficSourceStat
		visitorByTags  []models.TagStat
		visitorByProd  []models.ProductStat
	)

	var wg sync.WaitGroup
	wg.Add(3)

	// 1. Traffic Sources (Parallel)
	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Select("source_type as source, COUNT(log_id) as count").
			Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
			Group("source_type").
			Scan(&trafficSources)
	}()

	// 2. Visitor by Tags (Parallel)
	go func() {
		defer wg.Done()
		r.db.Model(&models.MerchantAnalytics{}).
			Select("searched_tag as tag, COUNT(log_id) as count").
			Where("business_id = ? AND event_type = 'visitor' AND source_type = 'tag_search' AND searched_tag IS NOT NULL AND searched_tag != '' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
			Group("searched_tag").Order("count DESC").Limit(limit).
			Scan(&visitorByTags)
	}()

	// 3. Visitor by Products (Parallel)
	go func() {
		defer wg.Done()
		r.db.Table("merchant_analytics").
			Select("products.product_name, COUNT(merchant_analytics.log_id) as count").
			Joins("JOIN products ON merchant_analytics.product_id = products.product_id AND products.deleted_at IS NULL").
			Where("merchant_analytics.business_id = ? AND merchant_analytics.event_type = 'visitor' AND merchant_analytics.source_type = 'product_search' AND DATE(merchant_analytics.created_at) >= ? AND DATE(merchant_analytics.created_at) <= ?", businessID, startDate, endDate).
			Group("products.product_name").Order("count DESC").Limit(limit).
			Scan(&visitorByProd)
	}()

	wg.Wait()

	if trafficSources == nil {
		trafficSources = []models.TrafficSourceStat{}
	}
	if visitorByTags == nil {
		visitorByTags = []models.TagStat{}
	}
	if visitorByProd == nil {
		visitorByProd = []models.ProductStat{}
	}

	return &models.HistoryAnalyticsResponse{
		TrafficSources: trafficSources,
		VisitorByTags:  visitorByTags,
		VisitorByProd:  visitorByProd,
	}, nil
}

func (r *analyticsRepository) GetExploreAnalytics(businessID, keyword string, radiusKm float64, limit int) (*models.ExploreAnalyticsResponse, error) {
	// 1. Dapatkan koordinat toko kita sendiri (Pusat Radius)
	var centerBiz struct {
		Latitude  *float64
		Longitude *float64
	}
	if err := r.db.Table("businesses").Select("latitude, longitude").Where("business_id = ? AND deleted_at IS NULL", businessID).Scan(&centerBiz).Error; err != nil {
		return nil, err
	}

	emptyResponse := &models.ExploreAnalyticsResponse{
		TagsSimilarities:    []models.TagSimilarity{},
		ProductSimilarities: []models.ProductSimilarity{},
		CompetitorsRelated:  []models.CompetitorStat{},
	}

	// Jika toko belum memiliki koordinat lokasi, kembalikan response kosong
	if centerBiz.Latitude == nil || centerBiz.Longitude == nil || (*centerBiz.Latitude == 0 && *centerBiz.Longitude == 0) {
		return emptyResponse, nil
	}

	lat := *centerBiz.Latitude
	lng := *centerBiz.Longitude

	// 2. Cari ID kompetitor dalam radius 5 KM menggunakan Haversine Formula yang terlindungi
	var competitorIDs []string
	r.db.Table("businesses").
		Select("business_id").
		Where("business_id != ? AND latitude IS NOT NULL AND longitude IS NOT NULL AND deleted_at IS NULL", businessID).
		Where("(6371 * acos(LEAST(1.0, GREATEST(-1.0, cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))))) <= ?", 
			lat, lng, lat, radiusKm).
		Pluck("business_id", &competitorIDs)

	// Jika tidak ada kompetitor di sekitar, kembalikan response kosong
	if len(competitorIDs) == 0 {
		return emptyResponse, nil
	}

	// Persiapkan pencarian keyword (Jika ada)
	searchKeyword := "%" + keyword + "%"

	var (
		tagsSimilarities    []models.TagSimilarity
		productSimilarities []models.ProductSimilarity
		competitorsRelated  []models.CompetitorStat
	)

	var wg sync.WaitGroup
	wg.Add(3)

	// 3. Ambil Tags Similarities dari kompetitor tersebut (Parallel)
	go func() {
		defer wg.Done()
		queryTags := r.db.Model(&models.MerchantAnalytics{}).
			Select("searched_tag as tag, COUNT(log_id) as audience_count").
			Where("business_id IN ? AND event_type = 'visitor' AND source_type = 'tag_search' AND searched_tag IS NOT NULL AND searched_tag != ''", competitorIDs)
		if keyword != "" {
			queryTags = queryTags.Where("searched_tag ILIKE ?", searchKeyword)
		}
		queryTags.Group("searched_tag").Order("audience_count DESC").Limit(limit).Scan(&tagsSimilarities)
	}()

	// 4. Ambil Product Similarities dari kompetitor tersebut (Parallel)
	go func() {
		defer wg.Done()
		queryProds := r.db.Table("merchant_analytics").
			Select("products.product_name, COUNT(merchant_analytics.log_id) as audience_count").
			Joins("JOIN products ON merchant_analytics.product_id = products.product_id AND products.deleted_at IS NULL").
			Where("merchant_analytics.business_id IN ? AND merchant_analytics.event_type = 'visitor' AND merchant_analytics.source_type = 'product_search'", competitorIDs)
		if keyword != "" {
			queryProds = queryProds.Where("products.product_name ILIKE ?", searchKeyword)
		}
		queryProds.Group("products.product_name").Order("audience_count DESC").Limit(limit).Scan(&productSimilarities)
	}()

	// 5. Ambil Top Competitors (Parallel)
	go func() {
		defer wg.Done()
		queryComps := r.db.Table("merchant_analytics").
			Select("businesses.business_id, businesses.business_name, COUNT(merchant_analytics.log_id) as audience_count").
			Joins("JOIN businesses ON merchant_analytics.business_id = businesses.business_id AND businesses.deleted_at IS NULL").
			Where("merchant_analytics.business_id IN ? AND merchant_analytics.event_type = 'visitor'", competitorIDs)
		if keyword != "" {
			queryComps = queryComps.Where("businesses.business_name ILIKE ?", searchKeyword)
		}
		queryComps.Group("businesses.business_id, businesses.business_name").Order("audience_count DESC").Limit(limit).Scan(&competitorsRelated)
	}()

	wg.Wait()

	if tagsSimilarities == nil {
		tagsSimilarities = []models.TagSimilarity{}
	}
	if productSimilarities == nil {
		productSimilarities = []models.ProductSimilarity{}
	}
	if competitorsRelated == nil {
		competitorsRelated = []models.CompetitorStat{}
	}

	return &models.ExploreAnalyticsResponse{
		TagsSimilarities:    tagsSimilarities,
		ProductSimilarities: productSimilarities,
		CompetitorsRelated:  competitorsRelated,
	}, nil
}

// === EVENT TRACKING ===

func (r *analyticsRepository) TrackVisitor(businessID, sourceType string, searchedTag *string, productID *string) error {
	event := models.MerchantAnalytics{
		BusinessID:  businessID,
		EventType:   "visitor",
		SourceType:  sourceType,
		SearchedTag: searchedTag,
		ProductID:   productID,
	}
	return r.db.Create(&event).Error
}

func (r *analyticsRepository) TrackImpression(businessID, sourceType string) error {
	event := models.MerchantAnalytics{
		BusinessID: businessID,
		EventType:  "impression",
		SourceType: sourceType,
	}
	return r.db.Create(&event).Error
}

func (r *analyticsRepository) TrackImpressionsBatch(events []models.MerchantAnalytics) error {
	if len(events) == 0 {
		return nil
	}
	return r.db.Create(&events).Error
}

// GetSignalStats menghitung signal yang diterima (masuk ke kategori toko) dan yang sudah direspons secara paralel
func (r *analyticsRepository) GetSignalStats(businessID string, startDate, endDate string) (int64, int64, error) {
	var received, responded int64
	var wg sync.WaitGroup
	wg.Add(2)

	// Signal yang relevan dengan toko ini (Parallel)
	go func() {
		defer wg.Done()
		r.db.Raw(`
			SELECT COUNT(DISTINCT s.signal_id)
			FROM signals s
			WHERE s.category_id IN (
				SELECT category_id FROM business_has_categories WHERE business_id = ?
			)
			AND DATE(s.created_at) >= ?
			AND DATE(s.created_at) <= ?
		`, businessID, startDate, endDate).Scan(&received)
	}()

	// Signal yang sudah direspons oleh bisnis ini (Parallel)
	go func() {
		defer wg.Done()
		r.db.Raw(`
			SELECT COUNT(DISTINCT sc.signal_id)
			FROM signal_conversations sc
			WHERE sc.business_id = ?
			AND DATE(sc.created_at) >= ?
			AND DATE(sc.created_at) <= ?
		`, businessID, startDate, endDate).Scan(&responded)
	}()

	wg.Wait()
	return received, responded, nil
}