package post

import (
	"testing"
	"time"

	"toentoen-backend/internal/domain/merchant/catalog"
	"toentoen-backend/internal/domain/shared/models"
)

type mockPostRepo struct {
	posts map[string]*models.BusinessPost
}

func (m *mockPostRepo) CreatePost(post *models.BusinessPost) error {
	if m.posts == nil {
		m.posts = make(map[string]*models.BusinessPost)
	}
	m.posts[post.PostID] = post
	return nil
}

func (m *mockPostRepo) GetPostsByBusinessID(businessID string, page, limit int) ([]models.BusinessPost, int64, error) {
	var list []models.BusinessPost
	for _, p := range m.posts {
		if p.BusinessID == businessID {
			list = append(list, *p)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockPostRepo) UpdatePost(post *models.BusinessPost) error {
	m.posts[post.PostID] = post
	return nil
}

func (m *mockPostRepo) DeletePost(postID, businessID string) error {
	delete(m.posts, postID)
	return nil
}

func (m *mockPostRepo) ToggleLike(postID, accountID string) (string, error) {
	return "liked", nil
}

func (m *mockPostRepo) GetPostByID(postID string) (*models.BusinessPost, error) {
	p, ok := m.posts[postID]
	if !ok {
		return nil, nil
	}
	return p, nil
}

type mockProductRepo struct {
	catalog.ProductRepository
}

func (m *mockProductRepo) HasAccessToBusiness(requesterID, role, businessID string) bool {
	return true
}

func (m *mockProductRepo) HasPermission(requesterID, businessID, perm string) bool {
	return true
}

func TestCreatePost_PromotionalDuration(t *testing.T) {
	repo := &mockPostRepo{posts: make(map[string]*models.BusinessPost)}
	prodRepo := &mockProductRepo{}
	svc := NewBusinessPostService(repo, prodRepo, nil, nil)

	isPromo := true
	days := 5
	input := models.CreatePostInput{
		Content:      "Diskon Akhir Pekan",
		IsPromotion:  &isPromo,
		DurationDays: &days,
	}

	post, err := svc.CreatePost("user-1", "owner", "biz-1", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !post.IsPromotion {
		t.Errorf("expected is_promotion=true, got false")
	}
	if post.PromotionEndAt == nil {
		t.Fatalf("expected promotion_end_at to be set, got nil")
	}

	expectedMinEnd := time.Now().AddDate(0, 0, 4)
	if post.PromotionEndAt.Before(expectedMinEnd) {
		t.Errorf("expected promotion_end_at to be around 5 days from now, got %v", post.PromotionEndAt)
	}
}

func TestStopPostPromotion(t *testing.T) {
	repo := &mockPostRepo{posts: make(map[string]*models.BusinessPost)}
	prodRepo := &mockProductRepo{}
	svc := NewBusinessPostService(repo, prodRepo, nil, nil)

	end := time.Now().AddDate(0, 0, 7)
	repo.posts["post-1"] = &models.BusinessPost{
		PostID:         "post-1",
		BusinessID:     "biz-1",
		Content:        "Promo",
		IsPromotion:    true,
		PromotionEndAt: &end,
	}

	err := svc.StopPostPromotion("user-1", "owner", "biz-1", "post-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	p := repo.posts["post-1"]
	if p.IsPromotion {
		t.Errorf("expected is_promotion=false, got true")
	}
}

func TestExtendPostPromotion(t *testing.T) {
	repo := &mockPostRepo{posts: make(map[string]*models.BusinessPost)}
	prodRepo := &mockProductRepo{}
	svc := NewBusinessPostService(repo, prodRepo, nil, nil)

	initialEnd := time.Now().AddDate(0, 0, 2)
	repo.posts["post-1"] = &models.BusinessPost{
		PostID:         "post-1",
		BusinessID:     "biz-1",
		Content:        "Promo",
		IsPromotion:    true,
		PromotionEndAt: &initialEnd,
	}

	addDays := 10
	updated, err := svc.ExtendPostPromotion("user-1", "owner", "biz-1", "post-1", models.ExtendPostPromotionInput{
		AddDays: &addDays,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !updated.IsPromotion {
		t.Errorf("expected is_promotion=true, got false")
	}

	expectedMin := time.Now().AddDate(0, 0, 11)
	if updated.PromotionEndAt.Before(expectedMin) {
		t.Errorf("expected promotion_end_at to be extended by ~10 days from initialEnd, got %v", updated.PromotionEndAt)
	}
}
