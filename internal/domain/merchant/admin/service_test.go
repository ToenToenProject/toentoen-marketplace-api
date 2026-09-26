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

func TestSupervisorPermissions_DefaultAndCustom(t *testing.T) {
	// Case 1: Supervisor without custom permissions (nil) defaults to all true
	unconfiguredSupervisor := models.BusinessAdminAssignment{
		Role:        "supervisor",
		Permissions: nil,
	}

	spvPerms := unconfiguredSupervisor.Permissions
	if spvPerms == nil || len(spvPerms) == 0 {
		if unconfiguredSupervisor.Role == "supervisor" {
			spvPerms = models.AdminPermissions{}.AllTrue()
		}
	}
	if len(spvPerms) != len(models.AllPermissions) {
		t.Fatalf("expected unconfigured supervisor to have %d permissions, got %d", len(models.AllPermissions), len(spvPerms))
	}
	for _, k := range models.AllPermissions {
		if !spvPerms[k] {
			t.Errorf("expected permission %s to be true by default for supervisor", k)
		}
	}

	// Case 2: Supervisor with customized permissions preserves false values (not overridden by all-true)
	customizedSupervisor := models.BusinessAdminAssignment{
		Role: "supervisor",
		Permissions: models.AdminPermissions{
			models.PermDeleteProduct: false,
			models.PermAddProduct:    true,
		},
	}

	perms := customizedSupervisor.Permissions
	if perms == nil || len(perms) == 0 {
		if customizedSupervisor.Role == "supervisor" {
			perms = models.AdminPermissions{}.AllTrue()
		}
	}

	normalized := make(models.AdminPermissions, len(models.AllPermissions))
	for _, k := range models.AllPermissions {
		normalized[k] = perms[k]
	}

	if normalized[models.PermDeleteProduct] != false {
		t.Errorf("expected PermDeleteProduct to remain false for customized supervisor, got %v", normalized[models.PermDeleteProduct])
	}
	if normalized[models.PermAddProduct] != true {
		t.Errorf("expected PermAddProduct to be true, got %v", normalized[models.PermAddProduct])
	}
}
