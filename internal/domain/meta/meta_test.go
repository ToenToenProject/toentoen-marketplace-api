package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"toentoen-backend/internal/domain/shared/models"

	"github.com/gofiber/fiber/v2"
)

func TestGetStoreTypes(t *testing.T) {
	service := NewMetaService(nil)
	handler := NewMetaHandler(service)

	app := fiber.New()
	app.Get("/meta/store-types", handler.GetStoreTypes)

	req := httptest.NewRequest(http.MethodGet, "/meta/store-types", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		ResponseCode string              `json:"responseCode"`
		Message      string              `json:"message"`
		Data         []map[string]string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) != 9 {
		t.Fatalf("expected 9 store types, got %d", len(result.Data))
	}

	expectedValues := map[string]string{
		"home":      "Rumah",
		"mobile":    "Keliling",
		"stand":     "Gerobak",
		"store":     "Toko",
		"shophouse": "Ruko",
		"market":    "Pasar",
		"plaza":     "Alun - alun",
		"mall":      "Pusat Belanja",
		"event":     "Acara/Musiman",
	}

	for _, item := range result.Data {
		val := item["value"]
		expectedLabel, exists := expectedValues[val]
		if !exists {
			t.Errorf("unexpected store type value: %s", val)
		}
		if item["label"] != expectedLabel {
			t.Errorf("expected label %s for %s, got %s", expectedLabel, val, item["label"])
		}
		if item["label_en"] == "" {
			t.Errorf("expected non-empty label_en for %s", val)
		}
	}
}

type mockMetaRepo struct {
	categories []models.Category
	tags       []models.Tag
	err        error
	lastCatID  *int64
	lastScope  string
}

func (m *mockMetaRepo) GetAllCategories() ([]models.Category, error) {
	return m.categories, m.err
}

func (m *mockMetaRepo) GetAllTags(categoryID *int64, scope string) ([]models.Tag, error) {
	m.lastCatID = categoryID
	m.lastScope = scope
	return m.tags, m.err
}

func (m *mockMetaRepo) GetAllOccupations() ([]models.Occupation, error) {
	return nil, nil
}

func TestGetTagsHandler(t *testing.T) {
	mockRepo := &mockMetaRepo{
		tags: []models.Tag{
			{TagID: 1, Name: "Experience & Environment", IsGlobal: true},
			{TagID: 2, Name: "Retail & Shopping", IsGlobal: false},
		},
	}
	service := NewMetaService(mockRepo)
	handler := NewMetaHandler(service)

	app := fiber.New()
	app.Get("/meta/tags", handler.GetTags)

	// Test 1: Without query params
	req := httptest.NewRequest(http.MethodGet, "/meta/tags", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mockRepo.lastCatID != nil {
		t.Errorf("expected nil categoryID, got %v", mockRepo.lastCatID)
	}
	if mockRepo.lastScope != "" {
		t.Errorf("expected empty scope, got %s", mockRepo.lastScope)
	}

	// Test 2: With scope=global and category_id=5
	req = httptest.NewRequest(http.MethodGet, "/meta/tags?scope=global&category_id=5", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mockRepo.lastCatID == nil || *mockRepo.lastCatID != 5 {
		t.Errorf("expected categoryID 5, got %v", mockRepo.lastCatID)
	}
	if mockRepo.lastScope != "global" {
		t.Errorf("expected scope global, got %s", mockRepo.lastScope)
	}
}
