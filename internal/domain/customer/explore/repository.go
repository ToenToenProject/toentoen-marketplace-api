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

type ExploreRepository interface {
	SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, int64, error)
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
	Keyword     string
	CategoryIDs []int64
	Category    string
	TagIDs      []int64
	TagNames    []string
	RawTags     string
	Lat         float64
	Lng         float64
	Radius      int // in meters
	Sort        string
	Limit       int64
	Offset      int64
}

type MeiliBusinessDocument = myMeili.MeiliBusinessDocument
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

func (r *exploreRepository) SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, int64, error) {
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

	// 3. Eksekusi Pencarian via Meilisearch jika client aktif
	if myMeili.Client != nil {
		index := myMeili.Client.Index("businesses")

		searchReq := &meilisearch.SearchRequest{
			Limit:  req.Limit,
			Offset: req.Offset,
		}

		var filters []string

		// Filter GeoRadius
		if req.Lat != 0 && req.Lng != 0 && req.Radius > 0 {
			filters = append(filters, "_geoRadius("+formatFloat(req.Lat)+", "+formatFloat(req.Lng)+", "+formatInt(req.Radius)+")")
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

		if req.Sort == "nearest" && req.Lat != 0 && req.Lng != 0 {
			searchReq.Sort = []string{"_geoPoint(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + "):asc"}
		} else if req.Sort == "rating" {
			searchReq.Sort = []string{"rating:desc"}
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
			return docs, res.TotalHits, nil
		}
		if err != nil {
			log.Printf("⚠️ Meilisearch query error: %v, beralih ke fallback PostgreSQL", err)
		}
	}

	// 4. PostgreSQL Fallback jika Meilisearch mengembalikan 0 data atau offline/error
	log.Printf("ℹ️ Menjalankan fallback PostgreSQL untuk explore bisnis (keyword='%s', category='%s', catIDs=%v, tags=%v)", req.Keyword, req.Category, req.CategoryIDs, req.TagNames)
	return r.searchBusinessesFromPostgres(req)
}

func (r *exploreRepository) searchBusinessesFromPostgres(req SearchRequest) ([]MeiliBusinessDocument, int64, error) {
	query := r.db.Model(&models.Business{}).
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		Where("businesses.is_active = ? AND businesses.status != ?", true, "deleted")

	if strings.TrimSpace(req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		query = query.Where(
			"(businesses.business_name ILIKE ? OR businesses.handle ILIKE ? OR businesses.business_details ILIKE ? OR businesses.business_id IN (SELECT business_id FROM products WHERE product_name ILIKE ?))",
			kw, kw, kw, kw,
		)
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

	if req.Sort == "rating" {
		query = query.Order("businesses.average_rating desc, businesses.review_count desc")
	}

	var businesses []models.Business
	if err := query.Find(&businesses).Error; err != nil {
		return nil, 0, err
	}

	var allDocs []MeiliBusinessDocument
	maxRadiusKm := float64(req.Radius) / 1000.0

	for i := range businesses {
		b := &businesses[i]
		distKm := 0.0
		if req.Lat != 0 && req.Lng != 0 {
			if b.Latitude == nil || b.Longitude == nil {
				if req.Radius > 0 {
					continue
				}
			} else {
				distKm = haversineDistance(req.Lat, req.Lng, *b.Latitude, *b.Longitude)
				if req.Radius > 0 && distKm > maxRadiusKm {
					continue
				}
			}
		}

		doc := myMeili.BuildBusinessDocument(r.db, b)
		doc.Distance = distKm
		allDocs = append(allDocs, doc)
	}

	if req.Sort == "nearest" && req.Lat != 0 && req.Lng != 0 {
		sort.Slice(allDocs, func(i, j int) bool {
			return allDocs[i].Distance < allDocs[j].Distance
		})
	}

	totalHits := int64(len(allDocs))
	if totalHits == 0 {
		return []MeiliBusinessDocument{}, 0, nil
	}

	start := req.Offset
	if start < 0 {
		start = 0
	}
	if start >= totalHits {
		return []MeiliBusinessDocument{}, totalHits, nil
	}

	end := start + req.Limit
	if end > totalHits {
		end = totalHits
	}

	paginated := allDocs[start:end]

	// Sembuhkan indeks Meilisearch di latar belakang untuk dokumen yang ditemukan
	for _, doc := range paginated {
		go myMeili.SyncBusiness(r.db, doc.ID)
	}

	return paginated, totalHits, nil
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
	// Ambil postingan terbaru (misal dalam 30 hari terakhir) atau order created_at desc
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

