package models

import (
	"testing"
)

func TestAllPermissions_Completeness(t *testing.T) {
	if len(AllPermissions) != 18 {
		t.Fatalf("expected 18 permissions in AllPermissions, got %d", len(AllPermissions))
	}

	for _, perm := range AllPermissions {
		if label, ok := PermissionLabels[perm]; !ok || label == "" {
			t.Errorf("missing human-readable label for permission: %s", perm)
		}
	}
}

func TestAdminPermissions_AllTrue(t *testing.T) {
	allTrue := AdminPermissions{}.AllTrue()
	if len(allTrue) != len(AllPermissions) {
		t.Fatalf("expected %d permissions, got %d", len(AllPermissions), len(allTrue))
	}

	for _, perm := range AllPermissions {
		if !allTrue[perm] {
			t.Errorf("expected permission %s to be true in AllTrue()", perm)
		}
	}
}

func TestAdminPermissions_DefaultFalse(t *testing.T) {
	perms := AdminPermissions{}
	for _, perm := range AllPermissions {
		if perms[perm] {
			t.Errorf("expected unset permission %s to evaluate to false (default deny)", perm)
		}
	}
}
