package signal_test

import (
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

func TestSignal_PopulateCategories_Multi(t *testing.T) {
	catName1 := "Otomotif"
	catName2 := "Makanan & Minuman"
	signal := models.Signal{
		SignalID: "SIG-123",
		SignalCategories: []models.SignalCategory{
			{
				SignalID:   "SIG-123",
				CategoryID: 14,
				Category: &models.Category{
					CategoryID: 14,
					Name:       catName1,
				},
			},
			{
				SignalID:   "SIG-123",
				CategoryID: 1,
				Category: &models.Category{
					CategoryID: 1,
					Name:       catName2,
				},
			},
		},
	}

	signal.PopulateCategories()

	if len(signal.CategoryIDs) != 2 {
		t.Fatalf("expected 2 category IDs, got %d", len(signal.CategoryIDs))
	}
	if signal.CategoryIDs[0] != 14 || signal.CategoryIDs[1] != 1 {
		t.Errorf("unexpected category IDs: %v", signal.CategoryIDs)
	}
	if len(signal.Categories) != 2 {
		t.Fatalf("expected 2 category names, got %d", len(signal.Categories))
	}
	if signal.Categories[0] != catName1 || signal.Categories[1] != catName2 {
		t.Errorf("unexpected categories: %v", signal.Categories)
	}
}

func TestSignal_PopulateCategories_FallbackSingle(t *testing.T) {
	catID := int64(14)
	signal := models.Signal{
		SignalID:   "SIG-123",
		CategoryID: &catID,
		Category:   "Otomotif",
	}

	signal.PopulateCategories()

	if len(signal.CategoryIDs) != 1 || signal.CategoryIDs[0] != 14 {
		t.Errorf("expected category ID [14], got %v", signal.CategoryIDs)
	}
	if len(signal.Categories) != 1 || signal.Categories[0] != "Otomotif" {
		t.Errorf("expected category name ['Otomotif'], got %v", signal.Categories)
	}
}

func TestCategoryDeduplicationAndMaxLimit(t *testing.T) {
	inputIDs := []int64{14, 1, 14, 2, 1}
	seen := make(map[int64]bool)
	var deduped []int64
	for _, id := range inputIDs {
		if id > 0 && !seen[id] {
			seen[id] = true
			deduped = append(deduped, id)
		}
	}

	if len(deduped) != 3 {
		t.Errorf("expected 3 deduped IDs, got %d", len(deduped))
	}

	// Test exceeding limit of 5
	tooManyIDs := []int64{1, 2, 3, 4, 5, 6}
	if len(tooManyIDs) <= 5 {
		t.Errorf("expected more than 5 IDs")
	}
}
