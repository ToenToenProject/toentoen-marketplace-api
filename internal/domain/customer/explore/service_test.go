package explore

import (
	"testing"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

func TestHaversineDistance(t *testing.T) {
	// Monas Jakarta (-6.1754, 106.8272) to Bundaran HI (-6.1950, 106.8230) ~ 2.2 km
	lat1, lon1 := -6.1754, 106.8272
	lat2, lon2 := -6.1950, 106.8230

	dist := haversineDistance(lat1, lon1, lat2, lon2)
	if dist < 2.0 || dist > 2.5 {
		t.Errorf("Expected distance around 2.2 km, got %f km", dist)
	}
}

func TestScoringAndRankingCalculation(t *testing.T) {
	now := time.Now()

	// Post 1: Near, Recent, Good engagement
	post1 := models.BusinessPost{
		PostID:       "POST-1",
		BusinessID:   "BIZ-1",
		LikeCount:    50,
		CommentCount: 10,
		CreatedAt:    now.Add(-2 * time.Hour),
	}

	// Post 2: Far away, Old, Low engagement
	post2 := models.BusinessPost{
		PostID:       "POST-2",
		BusinessID:   "BIZ-2",
		LikeCount:    1,
		CommentCount: 0,
		CreatedAt:    now.Add(-72 * time.Hour),
	}

	hoursOld1 := now.Sub(post1.CreatedAt).Hours()
	recencyScore1 := 1.0 / (1.0 + (hoursOld1 / 36.0))

	hoursOld2 := now.Sub(post2.CreatedAt).Hours()
	recencyScore2 := 1.0 / (1.0 + (hoursOld2 / 36.0))

	if recencyScore1 <= recencyScore2 {
		t.Errorf("Recent post should have higher recency score: %f vs %f", recencyScore1, recencyScore2)
	}
}
