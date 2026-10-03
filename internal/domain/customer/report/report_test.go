package report

import (
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

func TestGetReportCategories(t *testing.T) {
	t.Run("GetAllCategories", func(t *testing.T) {
		cats := models.GetReportCategories("")
		if len(cats) != 5 {
			t.Fatalf("Expected 5 categories, got %d", len(cats))
		}

		expectedTypes := []string{"business", "post", "comment", "product", "review"}
		for i, exp := range expectedTypes {
			if cats[i].TargetType != exp {
				t.Errorf("Expected target_type %s, got %s", exp, cats[i].TargetType)
			}
		}
	})

	t.Run("FilterByBusinessTargetType", func(t *testing.T) {
		cats := models.GetReportCategories("business")
		if len(cats) != 1 {
			t.Fatalf("Expected 1 category for business, got %d", len(cats))
		}
		if cats[0].ID != "cat_biz_01" {
			t.Errorf("Expected cat_biz_01, got %s", cats[0].ID)
		}
		if len(cats[0].SubCategories) != 5 {
			t.Errorf("Expected 5 sub_categories, got %d", len(cats[0].SubCategories))
		}
	})

	t.Run("FilterByPostTargetType", func(t *testing.T) {
		cats := models.GetReportCategories("post")
		if len(cats) != 1 {
			t.Fatalf("Expected 1 category for post, got %d", len(cats))
		}
		if cats[0].ID != "cat_post_02" {
			t.Errorf("Expected cat_post_02, got %s", cats[0].ID)
		}
	})
}
