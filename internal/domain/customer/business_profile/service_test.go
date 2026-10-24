package business_profile

import (
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

type mockBusinessProfileRepo struct {
	Repository
	getBusinessDetailFunc    func(businessID string) (models.Business, error)
	checkBusinessLikeFunc    func(accountID, businessID string) (bool, error)
	getBusinessLikeCountFunc func(businessID string) (int64, error)
	toggleBusinessLikeFunc   func(accountID, businessID string) (bool, int64, error)
	checkBookmarkFunc        func(accountID, businessID string) (bool, error)
	getRecentProductsFunc    func(businessID string, limit int) ([]models.Product, error)
	getRecentPostsFunc       func(businessID string, limit int) ([]models.BusinessPost, error)
	getSimilarBusinessesFunc func(category string, city string, currentBusinessID string, limit int) ([]models.Business, error)
	getRecentTopReviewsFunc  func(businessID string, limit int) ([]models.Review, error)
}

func (m *mockBusinessProfileRepo) GetBusinessDetail(businessID string) (models.Business, error) {
	if m.getBusinessDetailFunc != nil {
		return m.getBusinessDetailFunc(businessID)
	}
	name := "Test Business"
	return models.Business{BusinessID: businessID, BusinessName: &name}, nil
}

func (m *mockBusinessProfileRepo) CheckBusinessLike(accountID, businessID string) (bool, error) {
	if m.checkBusinessLikeFunc != nil {
		return m.checkBusinessLikeFunc(accountID, businessID)
	}
	return false, nil
}

func (m *mockBusinessProfileRepo) GetBusinessLikeCount(businessID string) (int64, error) {
	if m.getBusinessLikeCountFunc != nil {
		return m.getBusinessLikeCountFunc(businessID)
	}
	return 42, nil
}

func (m *mockBusinessProfileRepo) ToggleBusinessLike(accountID, businessID string) (bool, int64, error) {
	if m.toggleBusinessLikeFunc != nil {
		return m.toggleBusinessLikeFunc(accountID, businessID)
	}
	return true, 43, nil
}

func (m *mockBusinessProfileRepo) CheckBookmark(accountID, businessID string) (bool, error) {
	if m.checkBookmarkFunc != nil {
		return m.checkBookmarkFunc(accountID, businessID)
	}
	return false, nil
}

func (m *mockBusinessProfileRepo) GetRecentProducts(businessID string, limit int) ([]models.Product, error) {
	return []models.Product{}, nil
}

func (m *mockBusinessProfileRepo) GetRecentPosts(businessID string, limit int) ([]models.BusinessPost, error) {
	return []models.BusinessPost{}, nil
}

func (m *mockBusinessProfileRepo) GetSimilarBusinesses(category string, city string, currentBusinessID string, limit int) ([]models.Business, error) {
	return []models.Business{}, nil
}

func (m *mockBusinessProfileRepo) GetRecentTopReviews(businessID string, limit int) ([]models.Review, error) {
	return []models.Review{}, nil
}

func (m *mockBusinessProfileRepo) GetVerificationStatus(businessID string) (bool, bool, error) {
	return true, true, nil
}

func (m *mockBusinessProfileRepo) BatchGetVerificationStatus(businessIDs []string) (map[string]BusinessVerificationInfo, error) {
	res := make(map[string]BusinessVerificationInfo)
	for _, id := range businessIDs {
		res[id] = BusinessVerificationInfo{IsVerified: true, HasNIB: true}
	}
	return res, nil
}

func TestGetBusinessProfile_LikeFields(t *testing.T) {
	mockRepo := &mockBusinessProfileRepo{
		checkBusinessLikeFunc: func(accountID, businessID string) (bool, error) {
			if accountID == "ACC-LIKER" {
				return true, nil
			}
			return false, nil
		},
		getBusinessLikeCountFunc: func(businessID string) (int64, error) {
			return 128, nil
		},
	}

	service := NewService(mockRepo, nil)

	// Case 1: Logged in as ACC-LIKER
	resp, err := service.GetBusinessProfile("BUS-123", "ACC-LIKER", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsLikedByMe {
		t.Errorf("expected is_liked_by_me to be true, got false")
	}
	if resp.LikeCount != 128 {
		t.Errorf("expected like_count 128, got %d", resp.LikeCount)
	}
	if resp.TotalLikes != 128 {
		t.Errorf("expected total_likes 128, got %d", resp.TotalLikes)
	}

	// Case 2: Anonymous user (no account ID)
	respAnon, err := service.GetBusinessProfile("BUS-123", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respAnon.IsLikedByMe {
		t.Errorf("expected is_liked_by_me to be false for anon, got true")
	}
	if respAnon.LikeCount != 128 {
		t.Errorf("expected like_count 128, got %d", respAnon.LikeCount)
	}
}

func TestToggleLike(t *testing.T) {
	mockRepo := &mockBusinessProfileRepo{
		toggleBusinessLikeFunc: func(accountID, businessID string) (bool, int64, error) {
			return true, 129, nil
		},
	}

	service := NewService(mockRepo, nil)
	isLiked, totalLikes, err := service.ToggleLike("ACC-1", "BUS-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isLiked {
		t.Errorf("expected isLiked=true")
	}
	if totalLikes != 129 {
		t.Errorf("expected totalLikes=129, got %d", totalLikes)
	}
}
