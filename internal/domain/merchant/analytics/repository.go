package analytics

import (
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetDailyAnalytics(businessID string, limit int) (*models.DailyAnalyticsResponse, error)
	GetBasicStatsByDateRange(businessID string, startDate, endDate string) (int64, int64, error)
	GetHistoryDetails(businessID string, startDate, endDate string, limit int) (*models.HistoryAnalyticsResponse, error)
	GetExploreAnalytics(businessID, keyword string, radiusKm float64, limit int) (*models.ExploreAnalyticsResponse, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db}
}

func (r *analyticsRepository) GetDailyAnalytics(businessID string, limit int) (*models.DailyAnalyticsResponse, error) {
	var response models.DailyAnalyticsResponse

	r.db.Model(&models.MerchantAnalytics{}).
		Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) = CURRENT_DATE", businessID).
		Count(&response.TotalVisitors)

	r.db.Model(&models.MerchantAnalytics{}).
		Where("business_id = ? AND event_type = 'impression' AND DATE(created_at) = CURRENT_DATE", businessID).
		Count(&response.TotalImpression)

	r.db.Model(&models.MerchantAnalytics{}).
		Select("source_type as source, COUNT(log_id) as count").
		Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) = CURRENT_DATE", businessID).
		Group("source_type").
		Scan(&response.TrafficSources)

	r.db.Model(&models.MerchantAnalytics{}).
		Select("searched_tag as tag, COUNT(log_id) as count").
		Where("business_id = ? AND event_type = 'visitor' AND source_type = 'tag_search' AND DATE(created_at) = CURRENT_DATE", businessID).
		Group("searched_tag").
		Order("count DESC").Limit(limit).
		Scan(&response.VisitorByTags)

	r.db.Table("merchant_analytics").
		Select("products.product_name, COUNT(merchant_analytics.log_id) as count").
		Joins("JOIN products ON merchant_analytics.product_id = products.product_id").
		Where("merchant_analytics.business_id = ? AND merchant_analytics.event_type = 'visitor' AND merchant_analytics.source_type = 'product_search' AND DATE(merchant_analytics.created_at) = CURRENT_DATE", businessID).
		Group("products.product_name").
		Order("count DESC").Limit(limit).
		Scan(&response.VisitorByProd)

	return &response, nil
}

// Mengambil angka dasar (Visitors & Impressions) untuk dihitung persentasenya
func (r *analyticsRepository) GetBasicStatsByDateRange(businessID string, startDate, endDate string) (int64, int64, error) {
	var visitors, impressions int64

	r.db.Model(&models.MerchantAnalytics{}).
		Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
		Count(&visitors)

	r.db.Model(&models.MerchantAnalytics{}).
		Where("business_id = ? AND event_type = 'impression' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
		Count(&impressions)

	return visitors, impressions, nil
}

// Mengambil list detail (Traffic, Tags, Products)
func (r *analyticsRepository) GetHistoryDetails(businessID string, startDate, endDate string, limit int) (*models.HistoryAnalyticsResponse, error) {
	var response models.HistoryAnalyticsResponse

	// Traffic Sources
	r.db.Model(&models.MerchantAnalytics{}).
		Select("source_type as source, COUNT(log_id) as count").
		Where("business_id = ? AND event_type = 'visitor' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
		Group("source_type").
		Scan(&response.TrafficSources)

	// Visitor by Tags
	r.db.Model(&models.MerchantAnalytics{}).
		Select("searched_tag as tag, COUNT(log_id) as count").
		Where("business_id = ? AND event_type = 'visitor' AND source_type = 'tag_search' AND DATE(created_at) >= ? AND DATE(created_at) <= ?", businessID, startDate, endDate).
		Group("searched_tag").Order("count DESC").Limit(limit).
		Scan(&response.VisitorByTags)

	// Visitor by Products
	r.db.Table("merchant_analytics").
		Select("products.product_name, COUNT(merchant_analytics.log_id) as count").
		Joins("JOIN products ON merchant_analytics.product_id = products.product_id").
		Where("merchant_analytics.business_id = ? AND merchant_analytics.event_type = 'visitor' AND merchant_analytics.source_type = 'product_search' AND DATE(merchant_analytics.created_at) >= ? AND DATE(merchant_analytics.created_at) <= ?", businessID, startDate, endDate).
		Group("products.product_name").Order("count DESC").Limit(limit).
		Scan(&response.VisitorByProd)

	return &response, nil
}

func (r *analyticsRepository) GetExploreAnalytics(businessID, keyword string, radiusKm float64, limit int) (*models.ExploreAnalyticsResponse, error) {
	var response models.ExploreAnalyticsResponse

	// 1. Dapatkan koordinat toko kita sendiri (Pusat Radius)
	// Asumsi: Model Business Anda punya field Latitude dan Longitude float64
	var centerBiz struct {
		Latitude  float64
		Longitude float64
	}
	if err := r.db.Table("businesses").Select("latitude, longitude").Where("business_id = ?", businessID).Scan(&centerBiz).Error; err != nil {
		return nil, err
	}

	// 2. MAGIC: Cari ID kompetitor dalam radius 5 KM menggunakan Haversine Formula
	var competitorIDs []string
	r.db.Table("businesses").
		Select("business_id").
		Where("business_id != ?", businessID). // Jangan masukkan toko kita sendiri
		Where("(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) <= ?", 
			centerBiz.Latitude, centerBiz.Longitude, centerBiz.Latitude, radiusKm).
		Pluck("business_id", &competitorIDs)

	// Jika tidak ada kompetitor di sekitar, kembalikan response kosong
	if len(competitorIDs) == 0 {
		return &response, nil
	}

	// Persiapkan pencarian keyword (Jika ada)
	searchKeyword := "%" + keyword + "%"

	// 3. Ambil Tags Similarities dari kompetitor tersebut
	queryTags := r.db.Model(&models.MerchantAnalytics{}).
		Select("searched_tag as tag, COUNT(log_id) as audience_count").
		Where("business_id IN ? AND event_type = 'visitor' AND source_type = 'tag_search'", competitorIDs)
	if keyword != "" {
		queryTags = queryTags.Where("searched_tag ILIKE ?", searchKeyword)
	}
	queryTags.Group("searched_tag").Order("audience_count DESC").Limit(limit).Scan(&response.TagsSimilarities)

	// 4. Ambil Product Similarities dari kompetitor tersebut
	queryProds := r.db.Table("merchant_analytics").
		Select("products.product_name, COUNT(merchant_analytics.log_id) as audience_count").
		Joins("JOIN products ON merchant_analytics.product_id = products.product_id").
		Where("merchant_analytics.business_id IN ? AND merchant_analytics.event_type = 'visitor' AND merchant_analytics.source_type = 'product_search'", competitorIDs)
	if keyword != "" {
		queryProds = queryProds.Where("products.product_name ILIKE ?", searchKeyword)
	}
	queryProds.Group("products.product_name").Order("audience_count DESC").Limit(limit).Scan(&response.ProductSimilarities)

	// 5. Ambil Top Competitors (Siapa yang paling banyak dikunjungi audiens di radius ini?)
	queryComps := r.db.Table("merchant_analytics").
		Select("businesses.business_id, businesses.business_name, COUNT(merchant_analytics.log_id) as audience_count").
		Joins("JOIN businesses ON merchant_analytics.business_id = businesses.business_id").
		Where("merchant_analytics.business_id IN ? AND merchant_analytics.event_type = 'visitor'", competitorIDs)
	if keyword != "" {
		// Jika ada keyword, cari kompetitor yang namanya mengandung keyword tersebut
		queryComps = queryComps.Where("businesses.business_name ILIKE ?", searchKeyword)
	}
	queryComps.Group("businesses.business_id, businesses.business_name").Order("audience_count DESC").Limit(limit).Scan(&response.CompetitorsRelated)

	return &response, nil
}