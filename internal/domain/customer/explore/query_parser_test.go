package explore

import (
	"testing"

	myMeili "toentoen-backend/pkg/meilisearch"
)

func TestParseSearchQuery_NaturalLanguageExtraction(t *testing.T) {
	t.Run("Bengkel mobil dekat sini ada wifi buka sekarang", func(t *testing.T) {
		intent := ParseSearchQuery("bengkel mobil dekat sini ada wifi buka sekarang")
		if intent.CleanQuery != "bengkel mobil" {
			t.Errorf("expected CleanQuery 'bengkel mobil', got %q", intent.CleanQuery)
		}
		if !intent.DetectedNearby {
			t.Errorf("expected DetectedNearby true")
		}
		if !intent.DetectedOpenNow {
			t.Errorf("expected DetectedOpenNow true")
		}
		if len(intent.DetectedFacilities) != 1 || intent.DetectedFacilities[0] != "Wi-Fi" {
			t.Errorf("expected DetectedFacilities ['Wi-Fi'], got %v", intent.DetectedFacilities)
		}
	})

	t.Run("Ayam bakar yang ada promo parkir luas", func(t *testing.T) {
		intent := ParseSearchQuery("ayam bakar yang ada promo parkir luas")
		if intent.CleanQuery != "ayam bakar" {
			t.Errorf("expected CleanQuery 'ayam bakar', got %q", intent.CleanQuery)
		}
		if !intent.DetectedPromo {
			t.Errorf("expected DetectedPromo true")
		}
		if len(intent.DetectedFacilities) != 1 || intent.DetectedFacilities[0] != "Parkir" {
			t.Errorf("expected DetectedFacilities ['Parkir'], got %v", intent.DetectedFacilities)
		}
	})

	t.Run("Pure facility query", func(t *testing.T) {
		intent := ParseSearchQuery("bebas asap rokok")
		if intent.CleanQuery != "" {
			t.Errorf("expected empty CleanQuery, got %q", intent.CleanQuery)
		}
		if intent.EntityType != "FACILITY" {
			t.Errorf("expected EntityType 'FACILITY', got %q", intent.EntityType)
		}
		if len(intent.DetectedFacilities) != 1 || intent.DetectedFacilities[0] != "Bebas Asap Rokok" {
			t.Errorf("expected ['Bebas Asap Rokok'], got %v", intent.DetectedFacilities)
		}
	})
}

func TestComputeMatchExplanation(t *testing.T) {
	t.Run("Exact Offering Match", func(t *testing.T) {
		doc := MeiliBusinessDocument{
			Name:          "Warung Pak Budi",
			CategoryNames: []string{"Rumah Makan"},
			MatchedProducts: []MatchedProductPreview{
				{ProductName: "Ayam Bakar"},
			},
		}
		ComputeMatchExplanation(&doc, "ayam bakar", myMeili.ExpandKeywordWithSynonyms("ayam bakar"))
		if doc.MatchType != "EXACT_OFFERING" {
			t.Errorf("expected EXACT_OFFERING, got %q", doc.MatchType)
		}
		if doc.MatchedOffering != "Ayam Bakar" {
			t.Errorf("expected MatchedOffering 'Ayam Bakar', got %q", doc.MatchedOffering)
		}
		if doc.MatchConfidence != 1.0 {
			t.Errorf("expected confidence 1.0, got %f", doc.MatchConfidence)
		}
	})

	t.Run("Related Offering via Synonym", func(t *testing.T) {
		doc := MeiliBusinessDocument{
			Name:          "Rumah Makan Nusantara",
			CategoryNames: []string{"Restoran"},
			MatchedProducts: []MatchedProductPreview{
				{ProductName: "Ayam Bakar Madu"},
			},
		}
		ComputeMatchExplanation(&doc, "ayam panggang", myMeili.ExpandKeywordWithSynonyms("ayam panggang"))
		if doc.MatchType != "RELATED_OFFERING" {
			t.Errorf("expected RELATED_OFFERING, got %q", doc.MatchType)
		}
		if doc.MatchedOffering != "Ayam Bakar Madu" {
			t.Errorf("expected MatchedOffering 'Ayam Bakar Madu', got %q", doc.MatchedOffering)
		}
	})

	t.Run("Service Match", func(t *testing.T) {
		doc := MeiliBusinessDocument{
			Name:          "Auto Care Jaya",
			CategoryNames: []string{"Bengkel Mobil"},
			Services:      []string{"Ganti Oli", "Servis AC"},
		}
		ComputeMatchExplanation(&doc, "ganti oli", myMeili.ExpandKeywordWithSynonyms("ganti oli"))
		if doc.MatchType != "SERVICE_MATCH" {
			t.Errorf("expected SERVICE_MATCH, got %q", doc.MatchType)
		}
		if doc.MatchedOffering != "Ganti Oli" {
			t.Errorf("expected MatchedOffering 'Ganti Oli', got %q", doc.MatchedOffering)
		}
	})

	t.Run("Facility Match", func(t *testing.T) {
		doc := MeiliBusinessDocument{
			Name:          "Kopi Santai",
			CategoryNames: []string{"Kafe"},
			Facilities:    []string{"Wi-Fi", "Parkir"},
		}
		ComputeMatchExplanation(&doc, "wifi", myMeili.ExpandKeywordWithSynonyms("wifi"))
		if doc.MatchType != "FACILITY_MATCH" {
			t.Errorf("expected FACILITY_MATCH, got %q", doc.MatchType)
		}
		if doc.MatchedOffering != "Wi-Fi" {
			t.Errorf("expected MatchedOffering 'Wi-Fi', got %q", doc.MatchedOffering)
		}
	})
}
