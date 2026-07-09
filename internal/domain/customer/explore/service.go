package explore

type ExploreService interface {
	GetExploreBusinesses(req SearchRequest) (ExploreResponse, error)
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
