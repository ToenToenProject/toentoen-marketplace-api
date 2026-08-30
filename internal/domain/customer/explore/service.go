package explore

import (
	"math"
	"sort"
	"strings"
	"time"

	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ExploreService interface {
	GetExploreBusinesses(req SearchRequest) (ExploreResponse, error)
	GetTimelinePosts(query models.TimelineQueryInput, currentAccountID string) ([]models.PostResponse, error)
	RecordFeedEvents(accountID string, batch models.FeedEventBatchInput) error
}

type exploreService struct {
	repo ExploreRepository
}

func NewExploreService(repo ExploreRepository) ExploreService {
	return &exploreService{repo}
}

type ExploreResponse struct {
	Businesses []MeiliBusinessDocument `json:"businesses"`
	Pagination PaginationMeta          `json:"pagination"`
}

type PaginationMeta struct {
	CurrentPage int64 `json:"current_page"`
	TotalPages  int64 `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
}

func (s *exploreService) GetExploreBusinesses(req SearchRequest) (ExploreResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Radius <= 0 {
		req.Radius = 10000 // default 10km radius
	}

	page := req.Offset
	if page <= 0 {
		page = 1
	}
	req.Offset = (page - 1) * req.Limit

	docs, totalHits, err := s.repo.SearchBusinesses(req)
	if err != nil {
		return ExploreResponse{}, err
	}

	totalPages := totalHits / req.Limit
	if totalHits%req.Limit > 0 {
		totalPages++
	}

	if docs == nil {
		docs = []MeiliBusinessDocument{}
	}

	return ExploreResponse{
		Businesses: docs,
		Pagination: PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  totalHits,
		},
	}, nil
}

type scoredPost struct {
	post  models.BusinessPost
	score float64
}

// GetTimelinePosts mengimplementasikan Multi-Stage Recommendation Engine (Proximity + Affinity + Recency + Engagement)
func (s *exploreService) GetTimelinePosts(query models.TimelineQueryInput, currentAccountID string) ([]models.PostResponse, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}

	// 1. Candidate Retrieval (Ambil hingga 250 postingan kandidat)
	candidates, err := s.repo.GetCandidatePosts(query.CategoryID, 250)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return []models.PostResponse{}, nil
	}

	// 2. Load User Profiles & Sinyal Historis
	var likedMap map[string]bool
	var bookmarkedBusinesses map[string]bool
	var userAffinities map[int64]float64
	var seenPostIDs map[string]bool

	if strings.TrimSpace(currentAccountID) != "" {
		var candidatePostIDs []string
		for _, p := range candidates {
			candidatePostIDs = append(candidatePostIDs, p.PostID)
		}

		likedMap, _ = s.repo.GetPostLikesByAccount(currentAccountID, candidatePostIDs)
		bookmarkedBusinesses, _ = s.repo.GetBookmarkedBusinessIDs(currentAccountID)
		userAffinities, _ = s.repo.GetUserCategoryAffinities(currentAccountID)
		seenPostIDs, _ = s.repo.GetSeenPostIDs(currentAccountID, time.Now().Add(-24*time.Hour))
	}

	if likedMap == nil {
		likedMap = make(map[string]bool)
	}
	if bookmarkedBusinesses == nil {
		bookmarkedBusinesses = make(map[string]bool)
	}
	if userAffinities == nil {
		userAffinities = make(map[int64]float64)
	}
	if seenPostIDs == nil {
		seenPostIDs = make(map[string]bool)
	}

	// 3. Multi-Factor Scoring
	now := time.Now()
	var scoredList []scoredPost

	for _, p := range candidates {
		// A. Recency Decay Score (skor menyusut halus seiring umur postingan)
		hoursOld := now.Sub(p.CreatedAt).Hours()
		if hoursOld < 0 {
			hoursOld = 0
		}
		recencyScore := 1.0 / (1.0 + (hoursOld / 36.0)) // Paruh waktu ~36 jam

		// B. Proximity / Hyperlocal Score (Kedekatan GPS)
		proximityScore := 0.5
		if query.Lat != 0 && query.Lng != 0 && p.Business != nil && p.Business.Latitude != nil && p.Business.Longitude != nil {
			distKm := haversineDistance(query.Lat, query.Lng, *p.Business.Latitude, *p.Business.Longitude)
			proximityScore = 1.0 / (1.0 + (distKm / 7.0)) // Toko dalam radius 7km mendapat skor tinggi
		}

		// C. Category Affinity Match
		affinityScore := 1.0
		if p.Business != nil && len(p.Business.Categories) > 0 {
			for _, cat := range p.Business.Categories {
				if userScore, exists := userAffinities[cat.CategoryID]; exists {
					if userScore > affinityScore {
						affinityScore = math.Min(3.0, userScore)
					}
				}
			}
		}

		// D. Engagement Score (Like & Comment)
		engagementScore := math.Log10(1.0 + float64(p.LikeCount)*1.0 + float64(p.CommentCount)*2.0)
		if engagementScore > 3.0 {
			engagementScore = 3.0
		}

		// E. Bookmark / Follow Boost
		bookmarkBoost := 0.0
		if bookmarkedBusinesses[p.BusinessID] {
			bookmarkBoost = 1.5
		}

		// F. Hitung Skor Akhir
		// Bobot: Proximity (30%), Affinity (30%), Recency (20%), Engagement (20%) + Bookmark Boost
		finalScore := (proximityScore * 0.30) +
			(affinityScore * 0.30) +
			(recencyScore * 0.20) +
			(engagementScore * 0.20) +
			bookmarkBoost

		// G. Seen Penalty (Jika postingan sudah dilihat dalam 24 jam terakhir, beri penalti agar konten baru naik)
		if seenPostIDs[p.PostID] {
			finalScore *= 0.2
		}

		scoredList = append(scoredList, scoredPost{
			post:  p,
			score: finalScore,
		})
	}

	// 4. Urutkan berdasarkan Skor Tertinggi
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	// 5. Diversity Interleaving Filter (Maksimal 1 postingan per toko dalam sliding window 3 item)
	var diversePosts []models.BusinessPost
	recentBusinesses := make([]string, 0, 3)

	for _, sp := range scoredList {
		bizID := sp.post.BusinessID
		isRepeatedRecently := false

		for _, recentBiz := range recentBusinesses {
			if recentBiz == bizID {
				isRepeatedRecently = true
				break
			}
		}

		if !isRepeatedRecently || len(candidates) < 15 {
			diversePosts = append(diversePosts, sp.post)
			recentBusinesses = append(recentBusinesses, bizID)
			if len(recentBusinesses) > 3 {
				recentBusinesses = recentBusinesses[1:]
			}
		}
	}

	// Jika filter diversity menyaring terlalu banyak, tambahkan sisa postingan
	if len(diversePosts) < len(scoredList) {
		included := make(map[string]bool)
		for _, dp := range diversePosts {
			included[dp.PostID] = true
		}
		for _, sp := range scoredList {
			if !included[sp.post.PostID] {
				diversePosts = append(diversePosts, sp.post)
			}
		}
	}

	// 6. Pagination Slice
	offset := (query.Page - 1) * query.Limit
	if offset >= len(diversePosts) {
		return []models.PostResponse{}, nil
	}

	end := offset + query.Limit
	if end > len(diversePosts) {
		end = len(diversePosts)
	}

	pagedPosts := diversePosts[offset:end]

	// 7. Format Output Response
	var response []models.PostResponse
	for _, p := range pagedPosts {
		isLiked := likedMap[p.PostID]
		response = append(response, models.PostResponse{
			PostID:       p.PostID,
			BusinessID:   p.BusinessID,
			Content:      p.Content,
			Images:       p.Images,
			VideoURL:     p.VideoURL,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CreatedAt:    p.CreatedAt,
			UpdatedAt:    p.UpdatedAt,
			IsLikedByMe:  isLiked,
			Business:     p.Business,
		})
	}

	if response == nil {
		response = []models.PostResponse{}
	}

	return response, nil
}

// RecordFeedEvents memproses batch event interaksi dari mobile app secara asinkron
func (s *exploreService) RecordFeedEvents(accountID string, batch models.FeedEventBatchInput) error {
	if len(batch.Events) == 0 {
		return nil
	}

	now := time.Now()
	var interactions []models.UserPostInteraction

	for _, ev := range batch.Events {
		if strings.TrimSpace(ev.PostID) == "" {
			continue
		}

		interaction := models.UserPostInteraction{
			InteractionID: utils.GenerateUUID("EVT"),
			AccountID:     accountID,
			PostID:        ev.PostID,
			BusinessID:    ev.BusinessID,
			CategoryID:    ev.CategoryID,
			ActionType:    ev.ActionType,
			DwellTimeMs:   ev.DwellTimeMs,
			CreatedAt:     now,
		}
		interactions = append(interactions, interaction)

		// Hitung bobot delta untuk kategori
		if strings.TrimSpace(accountID) != "" && ev.CategoryID != nil && *ev.CategoryID > 0 {
			var delta float64
			switch ev.ActionType {
			case models.ActionDwell:
				if ev.DwellTimeMs >= 3000 {
					delta = 1.0
				}
			case models.ActionClickProduct:
				delta = 2.5
			case models.ActionClickChat:
				delta = 4.0
			case models.ActionClickStore, models.ActionClickMap:
				delta = 2.0
			case models.ActionLike:
				delta = 2.0
			case models.ActionBookmark:
				delta = 3.5
			case models.ActionShare:
				delta = 3.0
			case models.ActionSkip:
				delta = -0.3
			}

			if delta != 0 {
				_ = s.repo.UpsertUserCategoryAffinity(accountID, *ev.CategoryID, delta)
			}
		}
	}

	return s.repo.BatchSaveInteractions(interactions)
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

