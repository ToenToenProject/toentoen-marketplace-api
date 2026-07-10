package explore
import (
	"toentoen-backend/internal/domain/shared/models"
)
type ExploreService interface {
	GetExploreBusinesses(req SearchRequest) (ExploreResponse, error)
	GetTimelinePosts(page, limit int, currentAccountID string) ([]models.PostResponse, error)
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
	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Radius <= 0 {
		req.Radius = 10000 // default 10km radius
	}

	// Calculate offset
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

func (s *exploreService) GetTimelinePosts(page, limit int, currentAccountID string) ([]models.PostResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	posts, err := s.repo.GetTimelinePosts(limit, offset)
	if err != nil {
		return nil, err
	}

	likedMap := make(map[string]bool)
	if currentAccountID != "" {
		var postIDs []string
		for _, p := range posts {
			postIDs = append(postIDs, p.PostID)
		}
		likedMap, _ = s.repo.GetPostLikesByAccount(currentAccountID, postIDs)
	}

	var response []models.PostResponse
	for _, p := range posts {
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
