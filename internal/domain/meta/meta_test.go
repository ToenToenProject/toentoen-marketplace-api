package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
