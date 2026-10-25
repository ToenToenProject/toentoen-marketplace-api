package profile

import (
	"errors"
	"strings"
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

	recentReviews, err := s.repo.GetRecentReviews(accountID, 2) // Show 2 recent reviews
	if err != nil || recentReviews == nil {
		recentReviews = []models.RecentActivityResponse{}
	}

	var avatarUrl *string
	if account.ProfilePictureUrl != nil && strings.TrimSpace(*account.ProfilePictureUrl) != "" {
		cleaned := strings.TrimSpace(*account.ProfilePictureUrl)
		avatarUrl = &cleaned
	}

	fullName := strings.TrimSpace(account.FirstName + " " + account.LastName)
	if fullName == "" {
		if account.Username != "" {
			fullName = account.Username
		} else {
			fullName = "Pengguna ToenToen"
		}
	}

	username := strings.TrimSpace(account.Username)
	if username == "" {
		username = fullName
	}

	gender := ""
	if account.Gender != nil {
		gender = strings.ToLower(strings.TrimSpace(*account.Gender))
	}

	isEmailVerified := account.EmailVerifiedAt != nil
	isPhoneVerified := account.PhoneVerifiedAt != nil && account.PhoneNumber != nil && strings.TrimSpace(*account.PhoneNumber) != ""
	isVerified := isEmailVerified && isPhoneVerified

	response := &models.PublicProfileSummaryResponse{
		AccountID:            account.AccountID,
		Username:             username,
		FullName:             fullName,
		Gender:               gender,
		ProfilePictureURL:    avatarUrl,
		IsVerified:           isVerified,
		IsEmailVerified:      isEmailVerified,
		IsPhoneVerified:      isPhoneVerified,
		TotalCommentsReviews: totalCommentsReviews,
		RecentReviews:        recentReviews,
	}

	return response, nil
}
