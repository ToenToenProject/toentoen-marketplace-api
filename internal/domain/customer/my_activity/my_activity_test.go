package my_activity

import (
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"
)

type mockMyActivityRepo struct {
	customer        *models.Account
	businessReviews []models.Review
	productReviews  []models.ProductReview
	bookmarks       []models.BusinessBookmark
	wishlists       []models.ProductWishlist
	comments        []models.BusinessPostComment
}

func (m *mockMyActivityRepo) GetCustomerInfo(accountID string) (*models.Account, error) {
	if m.customer != nil {
		return m.customer, nil
	}
	return &models.Account{
		AccountID: accountID,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
	}, nil
}

func (m *mockMyActivityRepo) GetTotalReviews(accountID string) (int64, error) {
	return int64(len(m.businessReviews) + len(m.productReviews)), nil
}

func (m *mockMyActivityRepo) GetTotalComments(accountID string) (int64, error) {
	return int64(len(m.comments)), nil
}

func (m *mockMyActivityRepo) GetMyComments(accountID string, limit, offset int) ([]models.BusinessPostComment, error) {
	return m.comments, nil
}

func (m *mockMyActivityRepo) GetMyBusinessReviews(accountID string, limit, offset int, reviewType ...string) ([]models.MyActivityReviewResponse, error) {
	rType := "all"
	if len(reviewType) > 0 && reviewType[0] != "" {
		rType = reviewType[0]
	}

	var unified []models.MyActivityReviewResponse

	if rType == "all" || rType == "business" || rType == "store" {
		for _, br := range m.businessReviews {
			unified = append(unified, models.MyActivityReviewResponse{
				ReviewID:     br.ReviewID,
				ReviewType:   "business",
				BusinessID:   br.BusinessID,
				AccountID:    br.AccountID,
				Rating:       float64(br.Rating),
				Comment:      br.Comment,
				ReviewImages: br.ReviewImages,
				CreatedAt:    br.CreatedAt,
				Business:     br.Business,
				Store:        br.Business,
			})
		}
	}

	if rType == "all" || rType == "product" {
		for _, pr := range m.productReviews {
			prodID := pr.ProductID
			unified = append(unified, models.MyActivityReviewResponse{
				ReviewID:     pr.ReviewID,
				ReviewType:   "product",
				BusinessID:   pr.BusinessID,
				ProductID:    &prodID,
				AccountID:    pr.AccountID,
				Rating:       pr.Rating,
				Comment:      pr.Comment,
				ReviewImages: pr.ReviewImages,
				CreatedAt:    pr.CreatedAt,
				Business:     pr.Business,
				Store:        pr.Business,
				Product:      pr.Product,
			})
		}
	}

	if offset >= len(unified) {
		return []models.MyActivityReviewResponse{}, nil
	}
	end := offset + limit
	if end > len(unified) {
		end = len(unified)
	}
	return unified[offset:end], nil
}

func (m *mockMyActivityRepo) GetMyBookmarks(accountID string, limit, offset int) ([]models.BusinessBookmark, error) {
	return m.bookmarks, nil
}

func (m *mockMyActivityRepo) GetMyWishlists(accountID string, limit, offset int) ([]models.ProductWishlist, error) {
	return m.wishlists, nil
}

func (m *mockMyActivityRepo) ToggleWishlist(accountID, productID string) (bool, error) {
	return true, nil
}

func TestGetMyProfileSummary(t *testing.T) {
	now := time.Now()
	mockRepo := &mockMyActivityRepo{
		businessReviews: []models.Review{
			{ReviewID: "REV-1", CreatedAt: now.Add(-2 * time.Hour)},
		},
		productReviews: []models.ProductReview{
			{ReviewID: "PREV-1", CreatedAt: now.Add(-1 * time.Hour)},
			{ReviewID: "PREV-2", CreatedAt: now},
		},
		comments: []models.BusinessPostComment{
			{CommentID: "CMT-1"},
		},
	}
	service := NewMyActivityService(mockRepo)

	summary, err := service.GetMyProfileSummary("ACC-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// 1 business review + 2 product reviews = 3
	if summary.TotalReviewedPosts != 3 {
		t.Fatalf("expected 3 total reviews, got %d", summary.TotalReviewedPosts)
	}
	if summary.TotalCommentedPosts != 1 {
		t.Fatalf("expected 1 total comments, got %d", summary.TotalCommentedPosts)
	}
}

func TestGetMyBusinessReviews_Unified(t *testing.T) {
	now := time.Now()
	prodID := "PROD-1"
	busiID := "BUSI-1"
	bizName := "Toko Berkah"

	mockRepo := &mockMyActivityRepo{
		businessReviews: []models.Review{
			{
				ReviewID:   "REV-BIZ-1",
				BusinessID: busiID,
				AccountID:  "ACC-1",
				Rating:     5,
				Comment:    "Toko sangat ramah!",
				CreatedAt:  now.Add(-2 * time.Hour),
				Business: &models.Business{
					BusinessID:   busiID,
					BusinessName: &bizName,
				},
			},
		},
		productReviews: []models.ProductReview{
			{
				ReviewID:   "REV-PRD-1",
				ProductID:  prodID,
				BusinessID: busiID,
				AccountID:  "ACC-1",
				Rating:     4.5,
				Comment:    "Kualitas produk mantap!",
				CreatedAt:  now.Add(-1 * time.Hour),
				Business: &models.Business{
					BusinessID:   busiID,
					BusinessName: &bizName,
				},
				Product: &models.Product{
					ProductID:   prodID,
					ProductName: "Kemeja Flanel",
				},
			},
		},
	}
	service := NewMyActivityService(mockRepo)

	t.Run("returns unified business and product reviews", func(t *testing.T) {
		reviews, err := service.GetMyBusinessReviews("ACC-1", 1, 10, "all")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(reviews) != 2 {
			t.Fatalf("expected 2 reviews, got %d", len(reviews))
		}

		// Check business review
		foundBiz := false
		foundPrd := false
		for _, r := range reviews {
			if r.ReviewType == "business" {
				foundBiz = true
				if r.ReviewID != "REV-BIZ-1" || r.Rating != 5 {
					t.Errorf("unexpected business review data: %+v", r)
				}
				if r.Business == nil || r.Business.BusinessName == nil || *r.Business.BusinessName != "Toko Berkah" {
					t.Errorf("business relation not set on business review: %+v", r)
				}
			} else if r.ReviewType == "product" {
				foundPrd = true
				if r.ReviewID != "REV-PRD-1" || r.Rating != 4.5 {
					t.Errorf("unexpected product review data: %+v", r)
				}
				if r.Product == nil || r.Product.ProductName != "Kemeja Flanel" {
					t.Errorf("product relation not set on product review: %+v", r)
				}
				if r.Business == nil || r.Business.BusinessName == nil || *r.Business.BusinessName != "Toko Berkah" {
					t.Errorf("business relation not set on product review: %+v", r)
				}
			}
		}

		if !foundBiz {
			t.Error("expected business review in unified results, not found")
		}
		if !foundPrd {
			t.Error("expected product review in unified results, not found")
		}
	})

	t.Run("filters only product reviews when type=product", func(t *testing.T) {
		reviews, err := service.GetMyBusinessReviews("ACC-1", 1, 10, "product")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(reviews) != 1 {
			t.Fatalf("expected 1 product review, got %d", len(reviews))
		}
		if reviews[0].ReviewType != "product" {
			t.Fatalf("expected product review_type, got %s", reviews[0].ReviewType)
		}
	})

	t.Run("filters only business reviews when type=business", func(t *testing.T) {
		reviews, err := service.GetMyBusinessReviews("ACC-1", 1, 10, "business")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(reviews) != 1 {
			t.Fatalf("expected 1 business review, got %d", len(reviews))
		}
		if reviews[0].ReviewType != "business" {
			t.Fatalf("expected business review_type, got %s", reviews[0].ReviewType)
		}
	})

	t.Run("returns empty non-nil slice when user has no reviews", func(t *testing.T) {
		emptyRepo := &mockMyActivityRepo{}
		emptyService := NewMyActivityService(emptyRepo)

		reviews, err := emptyService.GetMyBusinessReviews("ACC-EMPTY", 1, 10)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if reviews == nil {
			t.Fatal("expected empty slice, got nil")
		}
		if len(reviews) != 0 {
			t.Fatalf("expected 0 reviews, got %d", len(reviews))
		}
	})
}
