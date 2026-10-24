package interaction

import (
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

type mockPostInteractionRepo struct {
	PostInteractionRepository
	getCommentsByPostIDFunc func(postID string) ([]models.BusinessPostComment, error)
}

func (m *mockPostInteractionRepo) GetCommentsByPostID(postID string) ([]models.BusinessPostComment, error) {
	if m.getCommentsByPostIDFunc != nil {
		return m.getCommentsByPostIDFunc(postID)
	}
	return []models.BusinessPostComment{}, nil
}

func TestGetPostComments_MockFallback(t *testing.T) {
	service := &postInteractionService{
		repo: &mockPostInteractionRepo{
			getCommentsByPostIDFunc: func(postID string) ([]models.BusinessPostComment, error) {
				t.Fatalf("repo should not be called for mock postID %s", postID)
				return nil, nil
			},
		},
	}

	testCases := []string{"PST-1", "pst-1", "PST-2", "pst-2", "  PST-1  ", ""}
	for _, tc := range testCases {
		res, err := service.GetPostComments(tc, "ACC-123")
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tc, err)
		}
		if res == nil {
			t.Fatalf("result should not be nil for %s", tc)
		}
		if len(res) != 0 {
			t.Fatalf("result should be empty for %s, got len: %d", tc, len(res))
		}
	}
}

func TestGetPostComments_EmptyWhenNoComments(t *testing.T) {
	service := &postInteractionService{
		repo: &mockPostInteractionRepo{
			getCommentsByPostIDFunc: func(postID string) ([]models.BusinessPostComment, error) {
				return []models.BusinessPostComment{}, nil
			},
		},
	}

	res, err := service.GetPostComments("PST-20261024-REAL", "ACC-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("result should not be nil")
	}
	if len(res) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(res))
	}
}
