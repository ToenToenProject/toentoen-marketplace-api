package profile

import (
	"encoding/json"
	"testing"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

type mockPublicProfileRepo struct {
	account              *models.Account
	totalCommentsReviews int64
	recentReviews        []models.RecentActivityResponse
}

func (m *mockPublicProfileRepo) GetAccountInfo(accountID string) (*models.Account, error) {
	return m.account, nil
}

func (m *mockPublicProfileRepo) GetTotalCommentsAndReviews(accountID string) (int64, error) {
	return m.totalCommentsReviews, nil
}

func (m *mockPublicProfileRepo) GetRecentReviews(accountID string, limit int) ([]models.RecentActivityResponse, error) {
	return m.recentReviews, nil
}

func TestPublicProfileSummary_PayloadFormat(t *testing.T) {
	gender := "male"
	avatar := "https://example.com/avatar.jpg"
	mockRepo := &mockPublicProfileRepo{
		account: &models.Account{
			AccountID:         "ACC-001",
			Username:          "budi_toeners",
			FirstName:         "Budi",
			LastName:          "Santoso",
			Gender:            &gender,
			ProfilePictureUrl: &avatar,
		},
		totalCommentsReviews: 4,
		recentReviews: []models.RecentActivityResponse{
			{
				ReviewID:     "REV-001",
				ActivityType: "business_review",
				Content:      "Pelayanan toko cepat dan ramah!",
				Rating:       5.0,
				CreatedAt:    time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC),
				BusinessName: "Nama Toko Anda",
			},
			{
				ReviewID:     "REV-002",
				ActivityType: "product_review",
				Content:      "Barang sesuai pesanan.",
				Rating:       4.5,
				CreatedAt:    time.Date(2026, 10, 18, 14, 30, 0, 0, time.UTC),
				ProductName:  "Nama Produk",
			},
		},
	}

	service := NewPublicProfileService(mockRepo)
	summary, err := service.GetPublicProfileSummary("ACC-001")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if summary.AccountID != "ACC-001" {
		t.Errorf("Expected AccountID 'ACC-001', got %s", summary.AccountID)
	}
	if summary.Username != "budi_toeners" {
		t.Errorf("Expected Username 'budi_toeners', got %s", summary.Username)
	}
	if summary.FullName != "Budi Santoso" {
		t.Errorf("Expected FullName 'Budi Santoso', got %s", summary.FullName)
	}
	if summary.Gender != "male" {
		t.Errorf("Expected Gender 'male', got %s", summary.Gender)
	}
	if summary.TotalCommentsReviews != 4 {
		t.Errorf("Expected TotalCommentsReviews 4, got %d", summary.TotalCommentsReviews)
	}
	if len(summary.RecentReviews) != 2 {
		t.Fatalf("Expected 2 recent reviews, got %d", len(summary.RecentReviews))
	}

	// Verify JSON marshaling format
	wrapped := map[string]interface{}{
		"status": "success",
		"data":   summary,
	}

	bytes, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	jsonStr := string(bytes)
	expectedSubstrings := []string{
		`"status":"success"`,
		`"account_id":"ACC-001"`,
		`"username":"budi_toeners"`,
		`"full_name":"Budi Santoso"`,
		`"gender":"male"`,
		`"profile_picture_url":"https://example.com/avatar.jpg"`,
		`"total_comments_reviews":4`,
		`"review_id":"REV-001"`,
		`"activity_type":"business_review"`,
		`"business_name":"Nama Toko Anda"`,
		`"review_id":"REV-002"`,
		`"activity_type":"product_review"`,
		`"product_name":"Nama Produk"`,
	}

	for _, sub := range expectedSubstrings {
		if !containsSub(jsonStr, sub) {
			t.Errorf("JSON output missing expected string %q\nGot: %s", sub, jsonStr)
		}
	}
}

func containsSub(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && findSub(s, substr)))
}

func findSub(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
