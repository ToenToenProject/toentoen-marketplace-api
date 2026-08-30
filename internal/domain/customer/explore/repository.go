package explore

import (
	"fmt"
	"log"
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
	Keyword string
	Lat     float64
	Lng     float64
	Radius  int // in meters
	Sort    string
	Limit   int64
	Offset  int64
}

type MeiliBusinessDocument struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Products      []string    `json:"products"`
	Rating        float64     `json:"rating"`
	ReviewCount   int64       `json:"review_count"`
	PreviewImages []string    `json:"preview_images"`
	IsOpen        bool        `json:"is_open"`
	CloseTime     string      `json:"close_time"`
	Geo           GeoLocation `json:"_geo"`
	Distance      float64     `json:"_geoDistance,omitempty"`
}

type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (r *exploreRepository) SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, int64, error) {
	index := myMeili.Client.Index("businesses")

	// Prepare Search Request
	searchReq := &meilisearch.SearchRequest{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	// Geo filtering
	if req.Lat != 0 && req.Lng != 0 {
		// Filter by radius (e.g., 10km)
		searchReq.Filter = []string{
			"_geoRadius(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + ", " + formatInt(req.Radius) + ")",
		}

		if req.Sort == "nearest" {
			searchReq.Sort = []string{
				"_geoPoint(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + "):asc",
			}
		}
	}

	if req.Sort == "rating" {
		searchReq.Sort = []string{"rating:desc"}
	}

	res, err := index.Search(req.Keyword, searchReq)
	if err != nil {
		log.Printf("Meilisearch error: %v", err)
		return nil, 0, err
	}

	var docs []MeiliBusinessDocument
	for _, hit := range res.Hits {
		var doc MeiliBusinessDocument
		if err := hit.DecodeInto(&doc); err == nil {
			// Convert meters to km for response
			doc.Distance = doc.Distance / 1000
			docs = append(docs, doc)
		}
	}

	return docs, res.TotalHits, nil
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

