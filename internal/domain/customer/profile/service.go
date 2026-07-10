package profile

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
)

type PublicProfileService interface {
	GetPublicProfileSummary(accountID string) (*models.PublicProfileSummaryResponse, error)
}

type publicProfileService struct {
	repo PublicProfileRepository
}

func NewPublicProfileService(repo PublicProfileRepository) PublicProfileService {
	return &publicProfileService{repo}
}

func (s *publicProfileService) GetPublicProfileSummary(accountID string) (*models.PublicProfileSummaryResponse, error) {
	account, err := s.repo.GetAccountInfo(accountID)
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	totalCommentsReviews, err := s.repo.GetTotalCommentsAndReviews(accountID)
	if err != nil {
		totalCommentsReviews = 0
	}

	recentReviews, err := s.repo.GetRecentReviews(accountID, 2) // As per UI, show 2 recent reviews
	if err != nil {
		recentReviews = []models.RecentActivityResponse{}
	}
	if recentReviews == nil {
		recentReviews = []models.RecentActivityResponse{}
	}

	var avatar *string
	if account.Customer != nil {
		avatar = account.Customer.ProfilePictureUrl
	}
	
	// If the user's fullname is empty, use username or a default
	name := account.Fullname
	if name == "" {
		name = account.Username
	}
	if name == "" {
		name = "User Name"
	}

	response := &models.PublicProfileSummaryResponse{
		AccountID:            account.AccountID,
		Username:             name,
		ProfilePictureURL:    avatar,
		TotalCommentsReviews: totalCommentsReviews,
		RecentReviews:        recentReviews,
	}

	return response, nil
}
