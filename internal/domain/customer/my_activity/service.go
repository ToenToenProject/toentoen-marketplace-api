package my_activity

import (
	"toentoen-backend/internal/domain/shared/models"
)

type MyActivityService interface {
	GetMyProfileSummary(accountID string) (MyProfileSummaryResponse, error)
	GetMyComments(accountID string, page, limit int) ([]models.BusinessPostComment, error)
	GetMyBusinessReviews(accountID string, page, limit int) ([]models.Review, error)
	GetMyBookmarks(accountID string, page, limit int) ([]models.BusinessBookmark, error)
	GetMyWishlists(accountID string, page, limit int) ([]models.ProductWishlist, error)
	ToggleWishlist(accountID, productID string) (bool, error)
}

type myActivityService struct {
	repo MyActivityRepository
}

func NewMyActivityService(repo MyActivityRepository) MyActivityService {
	return &myActivityService{repo: repo}
}

type MyProfileSummaryResponse struct {
	FullName           string `json:"full_name"`
	Username           string `json:"username"`
	AvatarURL          string `json:"avatar_url"`
	TotalReviewedPosts int64  `json:"total_reviewed_posts"`
	TotalCommentedPosts int64 `json:"total_commented_posts"`
}

func (s *myActivityService) GetMyProfileSummary(accountID string) (MyProfileSummaryResponse, error) {
	account, err := s.repo.GetCustomerInfo(accountID)
	if err != nil {
		return MyProfileSummaryResponse{}, err
	}

	totalReviews, _ := s.repo.GetTotalReviews(accountID)
	totalComments, _ := s.repo.GetTotalComments(accountID)

	fullName := account.Fullname
	username := account.Username

	avatarUrl := ""
	if account.Customer != nil && account.Customer.ProfilePictureUrl != nil {
		avatarUrl = *account.Customer.ProfilePictureUrl
	}

	return MyProfileSummaryResponse{
		FullName:           fullName,
		Username:           username,
		AvatarURL:          avatarUrl,
		TotalReviewedPosts: totalReviews,
		TotalCommentedPosts: totalComments,
	}, nil
}

func (s *myActivityService) GetMyComments(accountID string, page, limit int) ([]models.BusinessPostComment, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	comments, err := s.repo.GetMyComments(accountID, limit, offset)
	if comments == nil {
		comments = []models.BusinessPostComment{}
	}
	return comments, err
}

func (s *myActivityService) GetMyBusinessReviews(accountID string, page, limit int) ([]models.Review, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	reviews, err := s.repo.GetMyBusinessReviews(accountID, limit, offset)
	if reviews == nil {
		reviews = []models.Review{}
	}
	return reviews, err
}

func (s *myActivityService) GetMyBookmarks(accountID string, page, limit int) ([]models.BusinessBookmark, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	bookmarks, err := s.repo.GetMyBookmarks(accountID, limit, offset)
	if bookmarks == nil {
		bookmarks = []models.BusinessBookmark{}
	}
	return bookmarks, err
}

func (s *myActivityService) GetMyWishlists(accountID string, page, limit int) ([]models.ProductWishlist, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	wishlists, err := s.repo.GetMyWishlists(accountID, limit, offset)
	if wishlists == nil {
		wishlists = []models.ProductWishlist{}
	}
	return wishlists, err
}

func (s *myActivityService) ToggleWishlist(accountID, productID string) (bool, error) {
	return s.repo.ToggleWishlist(accountID, productID)
}
