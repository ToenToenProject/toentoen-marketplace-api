package business_profile

import (
	"testing"
	"time"
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
	getSavedAndWishlistedBizIDsFunc func(accountID string) (SavedShopData, error)
	getBusinessesByIDsFunc          func(businessIDs []string) ([]models.Business, error)
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

func (m *mockBusinessProfileRepo) GetSavedAndWishlistedBizIDs(accountID string) (SavedShopData, error) {
	if m.getSavedAndWishlistedBizIDsFunc != nil {
		return m.getSavedAndWishlistedBizIDsFunc(accountID)
	}
	return SavedShopData{
		SavedBizMap:      make(map[string]time.Time),
		WishlistedBizMap: make(map[string]int),
		WishlistLatestAt: make(map[string]time.Time),
	}, nil
}

func (m *mockBusinessProfileRepo) GetBusinessesByIDs(businessIDs []string) ([]models.Business, error) {
	if m.getBusinessesByIDsFunc != nil {
		return m.getBusinessesByIDsFunc(businessIDs)
	}
	return []models.Business{}, nil
}

func (m *mockBusinessProfileRepo) GetActivePromotions(businessID string) ([]models.Promotion, error) {
	return []models.Promotion{}, nil
}

func (m *mockBusinessProfileRepo) GetActivePromotionPosts(businessID string) ([]models.BusinessPost, error) {
	return []models.BusinessPost{}, nil
}

func (m *mockBusinessProfileRepo) GetAllProducts(businessID string, groupID string) ([]models.Product, error) {
	return []models.Product{}, nil
}

func (m *mockBusinessProfileRepo) GetActiveBundlingPromotions(businessID string) ([]models.Promotion, error) {
	return []models.Promotion{}, nil
}

func (m *mockBusinessProfileRepo) HasActivePromotion(businessID string) bool {
	return false
}

func (m *mockBusinessProfileRepo) GetAllProductGroups(businessID string) ([]models.ProductGroup, error) {
	return []models.ProductGroup{}, nil
}

func (m *mockBusinessProfileRepo) GetAllPosts(businessID string) ([]models.BusinessPost, error) {
	return []models.BusinessPost{}, nil
}

func (m *mockBusinessProfileRepo) GetPostLikesByAccount(accountID string, postIDs []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (m *mockBusinessProfileRepo) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	return &models.ReviewSummary{}, nil
}

func (m *mockBusinessProfileRepo) GetBookmarkedBusinesses(accountID string) ([]models.Business, error) {
	return []models.Business{}, nil
}

func (m *mockBusinessProfileRepo) ToggleBookmark(accountID, businessID string) (bool, error) {
	return true, nil
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

func TestGetSavedAndWishlistedShops(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Hour)

	name1 := "Shop One (Bookmark Only)"
	name2 := "Shop Two (Wishlist Only)"
	name3 := "Shop Three (Both)"

	mockRepo := &mockBusinessProfileRepo{
		getSavedAndWishlistedBizIDsFunc: func(accountID string) (SavedShopData, error) {
			if accountID == "ACC-USER-1" {
				return SavedShopData{
					SavedBizMap: map[string]time.Time{
						"BUS-1": older,
						"BUS-3": now,
					},
					WishlistedBizMap: map[string]int{
						"BUS-2": 3,
						"BUS-3": 1,
					},
					WishlistLatestAt: map[string]time.Time{
						"BUS-2": now.Add(-1 * time.Hour),
						"BUS-3": now,
					},
				}, nil
			}
			return SavedShopData{
				SavedBizMap:      make(map[string]time.Time),
				WishlistedBizMap: make(map[string]int),
				WishlistLatestAt: make(map[string]time.Time),
			}, nil
		},
		getBusinessesByIDsFunc: func(businessIDs []string) ([]models.Business, error) {
			bizMap := map[string]models.Business{
				"BUS-1": {BusinessID: "BUS-1", BusinessName: &name1, LikeCount: 10, ReviewCount: 5, AverageRating: 4.5},
				"BUS-2": {BusinessID: "BUS-2", BusinessName: &name2, LikeCount: 20, ReviewCount: 8, AverageRating: 4.8},
				"BUS-3": {BusinessID: "BUS-3", BusinessName: &name3, LikeCount: 30, ReviewCount: 12, AverageRating: 5.0},
			}
			var result []models.Business
			for _, id := range businessIDs {
				if b, ok := bizMap[id]; ok {
					result = append(result, b)
				}
			}
			return result, nil
		},
	}

	service := NewService(mockRepo, nil)

	// Case 1: Anonymous / Guest user
	resAnon, err := service.GetSavedAndWishlistedShops("", "all")
	if err != nil {
		t.Fatalf("unexpected error for anon: %v", err)
	}
	if len(resAnon.Shops) != 0 || resAnon.TotalShops != 0 {
		t.Errorf("expected 0 shops for anonymous user, got %d", resAnon.TotalShops)
	}

	// Case 2: User-1 with filter "all" (Union of bookmark and wishlist)
	resAll, err := service.GetSavedAndWishlistedShops("ACC-USER-1", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resAll.TotalSaved != 2 {
		t.Errorf("expected TotalSaved=2, got %d", resAll.TotalSaved)
	}
	if resAll.TotalWishlisted != 2 {
		t.Errorf("expected TotalWishlisted=2, got %d", resAll.TotalWishlisted)
	}
	if resAll.TotalShops != 3 {
		t.Errorf("expected TotalShops=3, got %d", resAll.TotalShops)
	}

	// Verify sources and aliases
	shopMap := make(map[string]SavedShopItem)
	for _, s := range resAll.Shops {
		shopMap[s.BusinessID] = s
	}

	s1, ok1 := shopMap["BUS-1"]
	if !ok1 || s1.Source != "bookmark" || !s1.IsSaved || s1.IsWishlisted {
		t.Errorf("BUS-1 mismatch: %+v", s1)
	}
	if s1.ShopID != "BUS-1" || s1.ID != "BUS-1" || s1.Name != name1 {
		t.Errorf("BUS-1 alias mismatch: %+v", s1)
	}

	s2, ok2 := shopMap["BUS-2"]
	if !ok2 || s2.Source != "wishlist" || s2.IsSaved || !s2.IsWishlisted || s2.WishlistedProductsCount != 3 {
		t.Errorf("BUS-2 mismatch: %+v", s2)
	}

	s3, ok3 := shopMap["BUS-3"]
	if !ok3 || s3.Source != "both" || !s3.IsSaved || !s3.IsWishlisted || s3.WishlistedProductsCount != 1 {
		t.Errorf("BUS-3 mismatch: %+v", s3)
	}

	// Case 3: Filter "saved"
	resSaved, err := service.GetSavedAndWishlistedShops("ACC-USER-1", "saved")
	if err != nil {
		t.Fatalf("unexpected error for filter saved: %v", err)
	}
	if resSaved.TotalShops != 2 {
		t.Errorf("expected 2 shops for filter saved, got %d", resSaved.TotalShops)
	}

	// Case 4: Filter "wishlisted"
	resWish, err := service.GetSavedAndWishlistedShops("ACC-USER-1", "wishlisted")
	if err != nil {
		t.Fatalf("unexpected error for filter wishlisted: %v", err)
	}
	if resWish.TotalShops != 2 {
		t.Errorf("expected 2 shops for filter wishlisted, got %d", resWish.TotalShops)
	}
}

