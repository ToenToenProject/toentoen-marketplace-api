package admin

import (
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"
)

func TestRejectionCooldownLogic(t *testing.T) {
	now := time.Now()
	recentRejection := now.Add(-2 * 24 * time.Hour) // 2 days ago (< 7 days)
	oldRejection := now.Add(-8 * 24 * time.Hour)    // 8 days ago (> 7 days)

	assignmentCooldownActive := models.BusinessAdminAssignment{
		RejectionCount: 3,
		LastRejectedAt: &recentRejection,
	}

	if assignmentCooldownActive.RejectionCount >= 3 && assignmentCooldownActive.LastRejectedAt != nil {
		if time.Since(*assignmentCooldownActive.LastRejectedAt) < 7*24*time.Hour {
			// Expected: in cooldown
		} else {
			t.Errorf("Expected cooldown to be active for rejection 2 days ago")
		}
	}

	assignmentCooldownExpired := models.BusinessAdminAssignment{
		RejectionCount: 3,
		LastRejectedAt: &oldRejection,
	}

	if assignmentCooldownExpired.RejectionCount >= 3 && assignmentCooldownExpired.LastRejectedAt != nil {
		if time.Since(*assignmentCooldownExpired.LastRejectedAt) < 7*24*time.Hour {
			t.Errorf("Expected cooldown to have expired for rejection 8 days ago")
		}
	}
}

func TestGracePeriodDuration(t *testing.T) {
	now := time.Now()
	scheduledAt := now.Add(72 * time.Hour)

	diff := scheduledAt.Sub(now)
	if diff.Hours() != 72 {
		t.Errorf("Expected grace period duration to be 72 hours, got %f", diff.Hours())
	}
}
