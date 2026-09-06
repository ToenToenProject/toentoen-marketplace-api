package account

import (
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

func TestIsAccountDataComplete(t *testing.T) {
	email := "test@example.com"
	emptyEmail := "   "

	t.Run("nil account returns inactive", func(t *testing.T) {
		status := isAccountDataComplete(nil)
		if status != "inactive" {
			t.Errorf("expected 'inactive', got '%s'", status)
		}
	})

	t.Run("suspended account preserves suspended status", func(t *testing.T) {
		account := &models.Account{
			AccountStatus: "suspended",
			Email:         &email,
		}
		status := isAccountDataComplete(account)
		if status != "suspended" {
			t.Errorf("expected 'suspended', got '%s'", status)
		}
	})

	t.Run("banned account preserves banned status", func(t *testing.T) {
		account := &models.Account{
			AccountStatus: "banned",
			Email:         &email,
		}
		status := isAccountDataComplete(account)
		if status != "banned" {
			t.Errorf("expected 'banned', got '%s'", status)
		}
	})

	t.Run("account with valid email returns active even without phone and ktp", func(t *testing.T) {
		account := &models.Account{
			Role:        "owner",
			Email:       &email,
			PhoneNumber: nil,
			Owner:       &models.Owner{KtpNumber: nil, KtpPhotoUrl: nil},
		}
		status := isAccountDataComplete(account)
		if status != "active" {
			t.Errorf("expected 'active', got '%s'", status)
		}
	})

	t.Run("account with nil or empty email returns inactive", func(t *testing.T) {
		accNil := &models.Account{
			Role:  "owner",
			Email: nil,
		}
		if status := isAccountDataComplete(accNil); status != "inactive" {
			t.Errorf("expected 'inactive' for nil email, got '%s'", status)
		}

		accEmpty := &models.Account{
			Role:  "owner",
			Email: &emptyEmail,
		}
		if status := isAccountDataComplete(accEmpty); status != "inactive" {
			t.Errorf("expected 'inactive' for empty email, got '%s'", status)
		}
	})
}
