package explore

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"toentoen-backend/internal/domain/shared/models"
	myMeili "toentoen-backend/pkg/meilisearch"

	"github.com/meilisearch/meilisearch-go"
	"gorm.io/gorm"
)

type SearchSummary struct {
	WithinRadiusCount  int64 `json:"within_radius_count"`
	OutsideRadiusCount int64 `json:"outside_radius_count"`
	TotalHits          int64 `json:"total_hits"`
}

type ExploreRepository interface {
	SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, SearchSummary, error)
	SearchProducts(req SearchRequest) ([]MeiliProductDocument, SearchSummary, error)
	GetTimelinePosts(limit, offset int) ([]models.BusinessPost, error)
	GetCandidatePosts(categoryID int64, limit int) ([]models.BusinessPost, error)
	GetPostLikesByAccount(accountID string, postIDs []string) (map[string]bool, error)
	GetBookmarkedBusinessIDs(accountID string) (map[string]bool, error)
	GetUserCategoryAffinities(accountID string) (map[int64]float64, error)
	GetSeenPostIDs(accountID string, since time.Time) (map[string]bool, error)
	BatchSaveInteractions(events []models.UserPostInteraction) error
	UpsertUserCategoryAffinity(accountID string, categoryID int64, deltaScore float64) error
}

type exploreRepository struct {
	db *gorm.DB
}

func NewExploreRepository(db *gorm.DB) ExploreRepository {
	return &exploreRepository{db: db}
}

type SearchRequest struct {
	Scope       string // "all", "store", "product"
	Keyword     string
	CategoryIDs []int64
	Category    string
	TagIDs      []int64
	TagNames    []string
	RawTags     string
	Hashtags    []string
	Lat         float64
	Lng         float64
	Radius      int // in meters
	Sort        string
	Limit       int64
	Offset      int64

	// Gatekeeping filters
	IsOpenNow  *bool
	IsPromo    *bool
	StoreTypes []string
	MinPrice   *float64
	MaxPrice   *float64
	StockTypes []string
}

type MeiliBusinessDocument = myMeili.MeiliBusinessDocument
type MeiliProductDocument = myMeili.MeiliProductDocument
type MatchedProductPreview = myMeili.MatchedProductPreview
type GeoLocation = myMeili.GeoLocation

func containsInt64(slice []int64, val int64) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func containsStringCaseInsensitive(slice []string, val string) bool {
	lowerVal := strings.ToLower(val)
	for _, v := range slice {
		if strings.ToLower(v) == lowerVal {
			return true
		}
	}
	return false
}

func businessMatchesHashtags(doc MeiliBusinessDocument, targetHashtags []string) bool {
	if len(targetHashtags) == 0 {
		return false
	}
	for _, t := range targetHashtags {
		lowerT := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "#")))
		if lowerT == "" {
			continue
		}
		for _, h := range doc.Hashtags {
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(h, "#")), lowerT) {
				return true
			}
		}
		for _, tag := range doc.TagNames {
			if strings.EqualFold(strings.TrimSpace(tag), lowerT) {
				return true
			}
		}
		for _, ct := range doc.CustomTags {
			if strings.EqualFold(strings.TrimSpace(ct), lowerT) {
				return true
			}
		}
	}
	return false
}

func productMatchesHashtags(doc MeiliProductDocument, targetHashtags []string) bool {
	if len(targetHashtags) == 0 {
		return false
	}
	for _, t := range targetHashtags {
		lowerT := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "#")))
		if lowerT == "" {
			continue
		}
		for _, h := range doc.Hashtags {
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(h, "#")), lowerT) {
				return true
			}
		}
		for _, cn := range doc.CategoryNames {
			if strings.EqualFold(strings.TrimSpace(cn), lowerT) {
				return true
			}
		}
	}
	return false
}

// rankBusinessDocuments membagi hasil pencarian toko ke dalam dua kelompok:
// Tier 1 (within_radius) & Tier 2 (outside_radius).
// Di dalam masing-masing kelompok, hashtag booster diutamakan kemudian diurutkan sesuai sort pilihan user.
func rankBusinessDocuments(docs []MeiliBusinessDocument, req SearchRequest, radiusKm float64) ([]MeiliBusinessDocument, int64, int64) {
	var within, outside []MeiliBusinessDocument

	for i := range docs {
		if req.Lat != 0 && req.Lng != 0 {
			if docs[i].Distance == 0 && (docs[i].Geo.Lat != 0 || docs[i].Geo.Lng != 0) {
				docs[i].Distance = haversineDistance(req.Lat, req.Lng, docs[i].Geo.Lat, docs[i].Geo.Lng)
			}
		}
		if radiusKm > 0 && req.Lat != 0 && req.Lng != 0 {
			docs[i].IsWithinRadius = docs[i].Distance <= radiusKm
		} else {
			docs[i].IsWithinRadius = true
		}

		if docs[i].IsWithinRadius {
			within = append(within, docs[i])
		} else {
			outside = append(outside, docs[i])
		}
	}

	sortTier := func(items []MeiliBusinessDocument) {
		sort.SliceStable(items, func(i, j int) bool {
			// 1. Soft Filter: Hashtag Booster
			if len(req.Hashtags) > 0 {
				iMatch := businessMatchesHashtags(items[i], req.Hashtags)
				jMatch := businessMatchesHashtags(items[j], req.Hashtags)
				if iMatch && !jMatch {
					return true
				}
				if !iMatch && jMatch {
					return false
				}
			}

			// 2. Sort Order Sesuai Pilihan User
			switch req.Sort {
			case "nearest":
				return items[i].Distance < items[j].Distance
			case "rating":
				if items[i].Rating != items[j].Rating {
					return items[i].Rating > items[j].Rating
				}
				return items[i].ReviewCount > items[j].ReviewCount
			case "price_asc":
				return items[i].MinPrice < items[j].MinPrice
			case "price_desc":
				return items[i].MaxPrice > items[j].MaxPrice
			default:
				if req.Lat != 0 && req.Lng != 0 {
					return items[i].Distance < items[j].Distance
				}
				return items[i].Rating > items[j].Rating
			}
		})
	}

	sortTier(within)
	sortTier(outside)

	result := make([]MeiliBusinessDocument, 0, len(docs))
	result = append(result, within...)
	result = append(result, outside...)
	return result, int64(len(within)), int64(len(outside))
}

// rankProductDocuments membagi hasil pencarian produk ke dalam dua kelompok:
// Tier 1 (within_radius) & Tier 2 (outside_radius).
// Di dalam masing-masing kelompok, hashtag booster diutamakan kemudian diurutkan sesuai sort pilihan user.
func rankProductDocuments(docs []MeiliProductDocument, req SearchRequest, radiusKm float64) ([]MeiliProductDocument, int64, int64) {
	var within, outside []MeiliProductDocument

	for i := range docs {
		if req.Lat != 0 && req.Lng != 0 {
			if docs[i].Distance == 0 && (docs[i].Geo.Lat != 0 || docs[i].Geo.Lng != 0) {
				docs[i].Distance = haversineDistance(req.Lat, req.Lng, docs[i].Geo.Lat, docs[i].Geo.Lng)
			}
		}
		if radiusKm > 0 && req.Lat != 0 && req.Lng != 0 {
			docs[i].IsWithinRadius = docs[i].Distance <= radiusKm
		} else {
			docs[i].IsWithinRadius = true
		}

		if docs[i].IsWithinRadius {
			within = append(within, docs[i])
		} else {
			outside = append(outside, docs[i])
		}
	}

	sortTier := func(items []MeiliProductDocument) {
		sort.SliceStable(items, func(i, j int) bool {
			// 1. Soft Filter: Hashtag Booster
			if len(req.Hashtags) > 0 {
				iMatch := productMatchesHashtags(items[i], req.Hashtags)
				jMatch := productMatchesHashtags(items[j], req.Hashtags)
				if iMatch && !jMatch {
					return true
				}
				if !iMatch && jMatch {
					return false
				}
			}

			// 2. Sort Order Sesuai Pilihan User
			switch req.Sort {
			case "nearest":
				return items[i].Distance < items[j].Distance
			case "rating":
				if items[i].Rating != items[j].Rating {
					return items[i].Rating > items[j].Rating
				}
				return items[i].ReviewCount > items[j].ReviewCount
			case "price_asc":
				return items[i].FinalPrice < items[j].FinalPrice
			case "price_desc":
				return items[i].FinalPrice > items[j].FinalPrice
			case "latest":
				return items[i].CreatedAt > items[j].CreatedAt
			default:
				if req.Lat != 0 && req.Lng != 0 {
					return items[i].Distance < items[j].Distance
				}
				return items[i].Rating > items[j].Rating
			}
		})
	}

	sortTier(within)
	sortTier(outside)

	result := make([]MeiliProductDocument, 0, len(docs))
	result = append(result, within...)
	result = append(result, outside...)
	return result, int64(len(within)), int64(len(outside))
}

func (r *exploreRepository) SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, SearchSummary, error) {
	// 1. Resolusi Kategori Dua Arah (Nama <-> ID)
	if req.Category != "" && len(req.CategoryIDs) == 0 {
		var foundCatIDs []int64
		r.db.Table("business_categories").Where("LOWER(name) = LOWER(?)", strings.TrimSpace(req.Category)).Pluck("category_id", &foundCatIDs)
		req.CategoryIDs = append(req.CategoryIDs, foundCatIDs...)
	} else if len(req.CategoryIDs) > 0 && req.Category == "" {
		var catNames []string
		r.db.Table("business_categories").Where("category_id IN (?)", req.CategoryIDs).Pluck("name", &catNames)
		if len(catNames) > 0 {
			req.Category = catNames[0]
		}
	}

	// 2. Resolusi Tag Dua Arah (Nama String <-> ID Integer)
	if len(req.TagNames) > 0 {
		lowerNames := make([]string, len(req.TagNames))
		for i, n := range req.TagNames {
			lowerNames[i] = strings.ToLower(strings.TrimSpace(n))
		}
		var foundTagIDs []int64
		r.db.Table("business_tags").Where("LOWER(name) IN (?)", lowerNames).Pluck("tag_id", &foundTagIDs)
		for _, id := range foundTagIDs {
			if !containsInt64(req.TagIDs, id) {
				req.TagIDs = append(req.TagIDs, id)
			}
		}
	}
	if len(req.TagIDs) > 0 {
		var foundTagNames []string
		r.db.Table("business_tags").Where("tag_id IN (?)", req.TagIDs).Pluck("name", &foundTagNames)
		for _, n := range foundTagNames {
			if !containsStringCaseInsensitive(req.TagNames, n) {
				req.TagNames = append(req.TagNames, n)
			}
		}
	}

	radiusKm := float64(req.Radius) / 1000.0

	// 3. Eksekusi Pencarian via Meilisearch jika client aktif
	if myMeili.Client != nil {
		index := myMeili.Client.Index("businesses")

		fetchLimit := int64(300)
		if req.Limit > 0 && (req.Offset+req.Limit)*3 > fetchLimit {
			fetchLimit = (req.Offset + req.Limit) * 3
		}

		searchReq := &meilisearch.SearchRequest{
			Limit:  fetchLimit,
			Offset: 0,
		}

		var filters []string

		// Gatekeeping: is_open_now
		if req.IsOpenNow != nil {
			filters = append(filters, fmt.Sprintf("is_open = %t", *req.IsOpenNow))
		}

		// Gatekeeping: is_promo
		if req.IsPromo != nil {
			filters = append(filters, fmt.Sprintf("has_promo = %t", *req.IsPromo))
		}

		// Gatekeeping: store_type
		if len(req.StoreTypes) > 0 {
			var stConds []string
			for _, st := range req.StoreTypes {
				stConds = append(stConds, fmt.Sprintf(`store_type = "%s"`, models.NormalizeStoreType(st)))
			}
			filters = append(filters, "("+strings.Join(stConds, " OR ")+")")
		}

		// Gatekeeping: min_price & max_price
		if req.MinPrice != nil {
			filters = append(filters, fmt.Sprintf("max_price >= %f", *req.MinPrice))
		}
		if req.MaxPrice != nil {
			filters = append(filters, fmt.Sprintf("min_price <= %f", *req.MaxPrice))
		}

		// Gatekeeping: stock_types
		if len(req.StockTypes) > 0 {
			var stConds []string
			for _, st := range req.StockTypes {
				stConds = append(stConds, fmt.Sprintf(`stock_types = "%s"`, strings.TrimSpace(st)))
			}
			filters = append(filters, "("+strings.Join(stConds, " OR ")+")")
		}

		// Filter Kategori
		var catFilters []string
		for _, cid := range req.CategoryIDs {
			catFilters = append(catFilters, fmt.Sprintf("category_ids = %d", cid))
		}
		if req.Category != "" {
			cleanCat := strings.ReplaceAll(req.Category, "\"", "\\\"")
			catFilters = append(catFilters, fmt.Sprintf("category_names = \"%s\"", cleanCat))
		}
		if len(catFilters) > 0 {
			filters = append(filters, "("+strings.Join(catFilters, " OR ")+")")
		}

		// Filter Tag (Mendukung ID dan Nama)
		var tagFilters []string
		for _, tid := range req.TagIDs {
			tagFilters = append(tagFilters, fmt.Sprintf("tag_ids = %d", tid))
		}
		for _, tname := range req.TagNames {
			cleanTag := strings.ReplaceAll(tname, "\"", "\\\"")
			tagFilters = append(tagFilters, fmt.Sprintf("tag_names = \"%s\"", cleanTag))
		}
		if len(tagFilters) > 0 {
			filters = append(filters, "("+strings.Join(tagFilters, " OR ")+")")
		}

		if len(filters) > 0 {
			searchReq.Filter = strings.Join(filters, " AND ")
		}

		if req.Lat != 0 && req.Lng != 0 {
			searchReq.Sort = []string{"_geoPoint(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + "):asc"}
		}

		res, err := index.Search(req.Keyword, searchReq)
		if err == nil && len(res.Hits) > 0 {
			var docs []MeiliBusinessDocument
			for _, hit := range res.Hits {
				var doc MeiliBusinessDocument
				if err := hit.DecodeInto(&doc); err == nil {
					doc.Distance = doc.Distance / 1000 // Convert meters to km
					docs = append(docs, doc)
				}
			}

			// Lakukan Skala Prioritas Radius (Tier 1 vs Tier 2) & Soft Ranking Booster
			rankedDocs, withinCount, outsideCount := rankBusinessDocuments(docs, req, radiusKm)
			totalHits := int64(len(rankedDocs))
			if res.TotalHits > totalHits {
				totalHits = res.TotalHits
			}

			start := req.Offset
			if start < 0 {
				start = 0
			}
			if start >= int64(len(rankedDocs)) {
				return []MeiliBusinessDocument{}, SearchSummary{WithinRadiusCount: withinCount, OutsideRadiusCount: outsideCount, TotalHits: totalHits}, nil
			}
			end := start + req.Limit
			if end > int64(len(rankedDocs)) {
				end = int64(len(rankedDocs))
			}

			return rankedDocs[start:end], SearchSummary{
				WithinRadiusCount:  withinCount,
				OutsideRadiusCount: outsideCount,
				TotalHits:          totalHits,
			}, nil
		}
		if err != nil {
			log.Printf("⚠️ Meilisearch query error: %v, beralih ke fallback PostgreSQL", err)
		}
	}

	// 4. PostgreSQL Fallback jika Meilisearch mengembalikan 0 data atau offline/error
	log.Printf("ℹ️ Menjalankan fallback PostgreSQL untuk explore bisnis (keyword='%s', category='%s', catIDs=%v, tags=%v)", req.Keyword, req.Category, req.CategoryIDs, req.TagNames)
	return r.searchBusinessesFromPostgres(req)
}

func (r *exploreRepository) SearchProducts(req SearchRequest) ([]MeiliProductDocument, SearchSummary, error) {
	// Resolusi Kategori
	if req.Category != "" && len(req.CategoryIDs) == 0 {
		var foundCatIDs []int64
		r.db.Table("business_categories").Where("LOWER(name) = LOWER(?)", strings.TrimSpace(req.Category)).Pluck("category_id", &foundCatIDs)
		req.CategoryIDs = append(req.CategoryIDs, foundCatIDs...)
	}

	radiusKm := float64(req.Radius) / 1000.0

	// Eksekusi Meilisearch jika client aktif
	if myMeili.Client != nil {
		index := myMeili.Client.Index("products")

		fetchLimit := int64(300)
		if req.Limit > 0 && (req.Offset+req.Limit)*3 > fetchLimit {
			fetchLimit = (req.Offset + req.Limit) * 3
		}

		searchReq := &meilisearch.SearchRequest{
			Limit:  fetchLimit,
			Offset: 0,
		}

		var filters []string

		// Gatekeeping: is_open
		if req.IsOpenNow != nil {
			filters = append(filters, fmt.Sprintf("is_open = %t", *req.IsOpenNow))
		}

		// Gatekeeping: is_promo / is_discounted
		if req.IsPromo != nil {
			filters = append(filters, fmt.Sprintf("is_discounted = %t", *req.IsPromo))
		}

		// Gatekeeping: store_type
		if len(req.StoreTypes) > 0 {
			var stConds []string
			for _, st := range req.StoreTypes {
				stConds = append(stConds, fmt.Sprintf(`store_type = "%s"`, models.NormalizeStoreType(st)))
			}
			filters = append(filters, "("+strings.Join(stConds, " OR ")+")")
		}

		// Gatekeeping: min_price & max_price
		if req.MinPrice != nil {
			filters = append(filters, fmt.Sprintf("final_price >= %f", *req.MinPrice))
		}
		if req.MaxPrice != nil {
			filters = append(filters, fmt.Sprintf("final_price <= %f", *req.MaxPrice))
		}

		// Gatekeeping: stock_type
		if len(req.StockTypes) > 0 {
			var stConds []string
			for _, st := range req.StockTypes {
				stConds = append(stConds, fmt.Sprintf(`stock_type = "%s"`, strings.TrimSpace(st)))
			}
			filters = append(filters, "("+strings.Join(stConds, " OR ")+")")
		}

		// Filter Kategori
		var catFilters []string
		for _, cid := range req.CategoryIDs {
			catFilters = append(catFilters, fmt.Sprintf("category_ids = %d", cid))
		}
		if req.Category != "" {
			cleanCat := strings.ReplaceAll(req.Category, "\"", "\\\"")
			catFilters = append(catFilters, fmt.Sprintf("category_names = \"%s\"", cleanCat))
		}
		if len(catFilters) > 0 {
			filters = append(filters, "("+strings.Join(catFilters, " OR ")+")")
		}

		if len(filters) > 0 {
			searchReq.Filter = strings.Join(filters, " AND ")
		}

		if req.Lat != 0 && req.Lng != 0 {
			searchReq.Sort = []string{"_geoPoint(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + "):asc"}
		}

		res, err := index.Search(req.Keyword, searchReq)
		if err == nil && len(res.Hits) > 0 {
			var docs []MeiliProductDocument
			for _, hit := range res.Hits {
				var doc MeiliProductDocument
				if err := hit.DecodeInto(&doc); err == nil {
					doc.Distance = doc.Distance / 1000 // Convert meters to km
					docs = append(docs, doc)
				}
			}

			// Skala Prioritas Radius & Soft Booster
			rankedDocs, withinCount, outsideCount := rankProductDocuments(docs, req, radiusKm)
			totalHits := int64(len(rankedDocs))
			if res.TotalHits > totalHits {
				totalHits = res.TotalHits
			}

			start := req.Offset
			if start < 0 {
				start = 0
			}
			if start >= int64(len(rankedDocs)) {
				return []MeiliProductDocument{}, SearchSummary{WithinRadiusCount: withinCount, OutsideRadiusCount: outsideCount, TotalHits: totalHits}, nil
			}
			end := start + req.Limit
			if end > int64(len(rankedDocs)) {
				end = int64(len(rankedDocs))
			}

			return rankedDocs[start:end], SearchSummary{
				WithinRadiusCount:  withinCount,
				OutsideRadiusCount: outsideCount,
				TotalHits:          totalHits,
			}, nil
		}
		if err != nil {
			log.Printf("⚠️ Meilisearch query error products: %v, beralih ke fallback PostgreSQL", err)
		}
	}

	return r.searchProductsFromPostgres(req)
}

func (r *exploreRepository) searchBusinessesFromPostgres(req SearchRequest) ([]MeiliBusinessDocument, SearchSummary, error) {
	query := r.db.Model(&models.Business{}).
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		Where("businesses.is_active = ? AND businesses.status != ?", true, "deleted")

	if strings.TrimSpace(req.Keyword) != "" {
		expandedTerms := myMeili.ExpandKeywordWithSynonyms(req.Keyword)
		var conds []string
		var args []interface{}
		for _, term := range expandedTerms {
			kw := "%" + strings.TrimSpace(term) + "%"
			conds = append(conds, "(businesses.business_name ILIKE ? OR businesses.handle ILIKE ? OR businesses.business_details ILIKE ? OR businesses.city ILIKE ? OR businesses.business_id IN (SELECT business_id FROM products WHERE product_name ILIKE ?))")
			args = append(args, kw, kw, kw, kw, kw)
		}
		if len(conds) > 0 {
			query = query.Where("("+strings.Join(conds, " OR ")+")", args...)
		}
	}

	if len(req.StoreTypes) > 0 {
		var normTypes []string
		for _, st := range req.StoreTypes {
			normTypes = append(normTypes, models.NormalizeStoreType(st))
		}
		query = query.Where("businesses.store_type IN (?)", normTypes)
	}

	if req.IsPromo != nil {
		if *req.IsPromo {
			query = query.Where("(businesses.business_id IN (SELECT business_id FROM products WHERE discount > 0 AND is_active = true) OR businesses.business_id IN (SELECT business_id FROM promotions WHERE status = 'active'))")
		}
	}

	if req.MinPrice != nil {
		query = query.Where("businesses.business_id IN (SELECT business_id FROM products WHERE price >= ? AND is_active = true)", *req.MinPrice)
	}
	if req.MaxPrice != nil {
		query = query.Where("businesses.business_id IN (SELECT business_id FROM products WHERE price <= ? AND is_active = true)", *req.MaxPrice)
	}

	if len(req.StockTypes) > 0 {
		query = query.Where("businesses.business_id IN (SELECT business_id FROM products WHERE stock_type IN (?) AND is_active = true)", req.StockTypes)
	}

	if len(req.CategoryIDs) > 0 || req.Category != "" {
		var catConds []string
		var catArgs []interface{}
		if len(req.CategoryIDs) > 0 {
			catConds = append(catConds, "businesses.business_id IN (SELECT business_id FROM business_has_categories WHERE category_id IN (?))")
			catArgs = append(catArgs, req.CategoryIDs)
		}
		if req.Category != "" {
			catConds = append(catConds, "businesses.business_id IN (SELECT bhc.business_id FROM business_has_categories bhc JOIN business_categories bc ON bc.category_id = bhc.category_id WHERE LOWER(bc.name) = LOWER(?))")
			catArgs = append(catArgs, strings.TrimSpace(req.Category))
			catConds = append(catConds, "LOWER(businesses.custom_category) = LOWER(?)")
			catArgs = append(catArgs, strings.TrimSpace(req.Category))
		}
		if len(catConds) > 0 {
			query = query.Where("("+strings.Join(catConds, " OR ")+")", catArgs...)
		}
	}

	if len(req.TagIDs) > 0 || len(req.TagNames) > 0 {
		var tagConds []string
		var tagArgs []interface{}
		if len(req.TagIDs) > 0 {
			tagConds = append(tagConds, "businesses.business_id IN (SELECT business_id FROM business_has_tags WHERE tag_id IN (?))")
			tagArgs = append(tagArgs, req.TagIDs)
		}
		if len(req.TagNames) > 0 {
			tagConds = append(tagConds, "businesses.business_id IN (SELECT bht.business_id FROM business_has_tags bht JOIN business_tags bt ON bt.tag_id = bht.tag_id WHERE LOWER(bt.name) IN (?))")
			lowerNames := make([]string, len(req.TagNames))
			for i, n := range req.TagNames {
				lowerNames[i] = strings.ToLower(strings.TrimSpace(n))
			}
			tagArgs = append(tagArgs, lowerNames)
		}
		if len(tagConds) > 0 {
			query = query.Where("("+strings.Join(tagConds, " OR ")+")", tagArgs...)
		}
	}

	var businesses []models.Business
	if err := query.Find(&businesses).Error; err != nil {
		return nil, SearchSummary{}, err
	}

	var allDocs []MeiliBusinessDocument
	for i := range businesses {
		b := &businesses[i]
		doc := myMeili.BuildBusinessDocument(r.db, b)
		if req.IsOpenNow != nil && doc.IsOpen != *req.IsOpenNow {
			continue
		}
		if req.Lat != 0 && req.Lng != 0 && b.Latitude != nil && b.Longitude != nil {
			doc.Distance = haversineDistance(req.Lat, req.Lng, *b.Latitude, *b.Longitude)
		}
		allDocs = append(allDocs, doc)
	}

	radiusKm := float64(req.Radius) / 1000.0
	rankedDocs, withinCount, outsideCount := rankBusinessDocuments(allDocs, req, radiusKm)
	totalHits := int64(len(rankedDocs))

	start := req.Offset
	if start < 0 {
		start = 0
	}
	if start >= totalHits {
		return []MeiliBusinessDocument{}, SearchSummary{WithinRadiusCount: withinCount, OutsideRadiusCount: outsideCount, TotalHits: totalHits}, nil
	}
	end := start + req.Limit
	if end > totalHits {
		end = totalHits
	}
	paginated := rankedDocs[start:end]

	for _, doc := range paginated {
		go myMeili.SyncBusiness(r.db, doc.ID)
	}

	return paginated, SearchSummary{
		WithinRadiusCount:  withinCount,
		OutsideRadiusCount: outsideCount,
		TotalHits:          totalHits,
	}, nil
}

func (r *exploreRepository) searchProductsFromPostgres(req SearchRequest) ([]MeiliProductDocument, SearchSummary, error) {
	query := r.db.Model(&models.Product{}).
		Joins("JOIN businesses ON businesses.business_id = products.business_id").
		Where("products.is_active = ? AND businesses.is_active = ? AND businesses.status != ?", true, true, "deleted")

	if strings.TrimSpace(req.Keyword) != "" {
		expandedTerms := myMeili.ExpandKeywordWithSynonyms(req.Keyword)
		var conds []string
		var args []interface{}
		for _, term := range expandedTerms {
			kw := "%" + strings.TrimSpace(term) + "%"
			conds = append(conds, "(products.product_name ILIKE ? OR businesses.business_name ILIKE ?)")
			args = append(args, kw, kw)
		}
		if len(conds) > 0 {
			query = query.Where("("+strings.Join(conds, " OR ")+")", args...)
		}
	}

	if len(req.StoreTypes) > 0 {
		var normTypes []string
		for _, st := range req.StoreTypes {
			normTypes = append(normTypes, models.NormalizeStoreType(st))
		}
		query = query.Where("businesses.store_type IN (?)", normTypes)
	}

	if req.IsPromo != nil {
		if *req.IsPromo {
			query = query.Where("products.discount > 0")
		}
	}

	if req.MinPrice != nil {
		query = query.Where("products.price >= ?", *req.MinPrice)
	}
	if req.MaxPrice != nil {
		query = query.Where("products.price <= ?", *req.MaxPrice)
	}

	if len(req.StockTypes) > 0 {
		query = query.Where("products.stock_type IN (?)", req.StockTypes)
	}

	if len(req.CategoryIDs) > 0 || req.Category != "" {
		if len(req.CategoryIDs) > 0 {
			query = query.Where("businesses.business_id IN (SELECT business_id FROM business_has_categories WHERE category_id IN (?))", req.CategoryIDs)
		}
		if req.Category != "" {
			query = query.Where("(businesses.business_id IN (SELECT bhc.business_id FROM business_has_categories bhc JOIN business_categories bc ON bc.category_id = bhc.category_id WHERE LOWER(bc.name) = LOWER(?)) OR LOWER(businesses.custom_category) = LOWER(?))", strings.TrimSpace(req.Category), strings.TrimSpace(req.Category))
		}
	}

	var products []models.Product
	if err := query.Find(&products).Error; err != nil {
		return nil, SearchSummary{}, err
	}

	bizMap := make(map[string]*models.Business)
	if len(products) > 0 {
		bizIDs := make([]string, 0, len(products))
		for _, p := range products {
			bizIDs = append(bizIDs, p.BusinessID)
		}
		var bizList []models.Business
		r.db.Preload("Hours").Preload("Categories").Where("business_id IN (?)", bizIDs).Find(&bizList)
		for i := range bizList {
			bizMap[bizList[i].BusinessID] = &bizList[i]
		}
	}

	var allDocs []MeiliProductDocument
	for i := range products {
		p := &products[i]
		b := bizMap[p.BusinessID]
		doc := myMeili.BuildProductDocument(r.db, p, b)
		if req.IsOpenNow != nil && doc.IsOpen != *req.IsOpenNow {
			continue
		}
		if req.Lat != 0 && req.Lng != 0 && b != nil && b.Latitude != nil && b.Longitude != nil {
			doc.Distance = haversineDistance(req.Lat, req.Lng, *b.Latitude, *b.Longitude)
		}
		allDocs = append(allDocs, doc)
	}

	radiusKm := float64(req.Radius) / 1000.0
	rankedDocs, withinCount, outsideCount := rankProductDocuments(allDocs, req, radiusKm)
	totalHits := int64(len(rankedDocs))

	start := req.Offset
	if start < 0 {
		start = 0
	}
	if start >= totalHits {
		return []MeiliProductDocument{}, SearchSummary{WithinRadiusCount: withinCount, OutsideRadiusCount: outsideCount, TotalHits: totalHits}, nil
	}
	end := start + req.Limit
	if end > totalHits {
		end = totalHits
	}
	paginated := rankedDocs[start:end]

	return paginated, SearchSummary{
		WithinRadiusCount:  withinCount,
		OutsideRadiusCount: outsideCount,
		TotalHits:          totalHits,
	}, nil
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%f", f)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}

func (r *exploreRepository) GetTimelinePosts(limit, offset int) ([]models.BusinessPost, error) {
	var posts []models.BusinessPost
	err := r.db.Preload("Business").
		Preload("Business.Categories").
		Preload("Business.Hours").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	return posts, err
}

// GetCandidatePosts mengambil kandidat postingan (hingga 200 item) untuk diproses oleh scoring engine
func (r *exploreRepository) GetCandidatePosts(categoryID int64, limit int) ([]models.BusinessPost, error) {
	if limit <= 0 {
		limit = 200
	}

	query := r.db.Model(&models.BusinessPost{}).
		Preload("Business").
		Preload("Business.Categories").
		Preload("Business.Hours").
		Joins("JOIN businesses ON businesses.business_id = business_posts.business_id").
		Where("businesses.is_active = ?", true)

	if categoryID > 0 {
		query = query.Joins("JOIN business_has_categories ON business_has_categories.business_id = businesses.business_id").
			Where("business_has_categories.category_id = ?", categoryID)
	}

	var posts []models.BusinessPost
	err := query.Order("business_posts.created_at desc").Limit(limit).Find(&posts).Error
	return posts, err
}

func (r *exploreRepository) GetPostLikesByAccount(accountID string, postIDs []string) (map[string]bool, error) {
	likedMap := make(map[string]bool)
	if len(postIDs) == 0 || strings.TrimSpace(accountID) == "" {
		return likedMap, nil
	}
	var likes []models.BusinessPostLike
	err := r.db.Where("account_id = ? AND post_id IN ?", accountID, postIDs).Find(&likes).Error
	for _, like := range likes {
		likedMap[like.PostID] = true
	}
	return likedMap, err
}

func (r *exploreRepository) GetBookmarkedBusinessIDs(accountID string) (map[string]bool, error) {
	bookmarkMap := make(map[string]bool)
	if strings.TrimSpace(accountID) == "" {
		return bookmarkMap, nil
	}
	var bookmarks []models.BusinessBookmark
	err := r.db.Where("account_id = ?", accountID).Find(&bookmarks).Error
	for _, b := range bookmarks {
		bookmarkMap[b.BusinessID] = true
	}
	return bookmarkMap, err
}

func (r *exploreRepository) GetUserCategoryAffinities(accountID string) (map[int64]float64, error) {
	affinityMap := make(map[int64]float64)
	if strings.TrimSpace(accountID) == "" {
		return affinityMap, nil
	}
	var affinities []models.UserCategoryAffinity
	err := r.db.Where("account_id = ?", accountID).Find(&affinities).Error
	for _, a := range affinities {
		affinityMap[a.CategoryID] = a.AffinityScore
	}
	return affinityMap, err
}

func (r *exploreRepository) GetSeenPostIDs(accountID string, since time.Time) (map[string]bool, error) {
	seenMap := make(map[string]bool)
	if strings.TrimSpace(accountID) == "" {
		return seenMap, nil
	}
	var seenPostIDs []string
	err := r.db.Model(&models.UserPostInteraction{}).
		Where("account_id = ? AND action_type IN ('impression', 'skip') AND created_at >= ?", accountID, since).
		Pluck("post_id", &seenPostIDs).Error
	for _, id := range seenPostIDs {
		seenMap[id] = true
	}
	return seenMap, err
}

func (r *exploreRepository) BatchSaveInteractions(events []models.UserPostInteraction) error {
	if len(events) == 0 {
		return nil
	}
	return r.db.Create(&events).Error
}

func (r *exploreRepository) UpsertUserCategoryAffinity(accountID string, categoryID int64, deltaScore float64) error {
	if accountID == "" || categoryID <= 0 {
		return nil
	}
	query := `
		INSERT INTO user_category_affinities (account_id, category_id, affinity_score, interaction_count, last_interacted_at, created_at, updated_at)
		VALUES (?, ?, GREATEST(0.1, ?), 1, NOW(), NOW(), NOW())
		ON CONFLICT (account_id, category_id) DO UPDATE SET
			affinity_score = GREATEST(0.1, user_category_affinities.affinity_score + EXCLUDED.affinity_score),
			interaction_count = user_category_affinities.interaction_count + 1,
			last_interacted_at = NOW(),
			updated_at = NOW()
	`
	return r.db.Exec(query, accountID, categoryID, deltaScore).Error
}
