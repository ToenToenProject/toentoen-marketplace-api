package auth

import (
	"testing"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
)

func TestEvaluateAccountStatus(t *testing.T) {
	mockRepo := &mockAuthRepo{}
	service := NewAuthService(mockRepo)
	now := time.Now()

	// Pastikan Envs diinisialisasi
	if config.Envs == nil {
		config.Envs = &config.AppConfig{}
	}

	t.Run("nil account returns inactive", func(t *testing.T) {
		status := service.EvaluateAccountStatus(nil)
		if status != "inactive" {
			t.Errorf("expected 'inactive', got '%s'", status)
		}
	})

	t.Run("banned or suspended account preserves status", func(t *testing.T) {
		accBanned := &models.Account{
			AccountStatus:   "banned",
			EmailVerifiedAt: &now,
		}
		if status := service.EvaluateAccountStatus(accBanned); status != "banned" {
			t.Errorf("expected 'banned', got '%s'", status)
		}

		accSuspended := &models.Account{
			AccountStatus:   "suspended",
			EmailVerifiedAt: &now,
		}
		if status := service.EvaluateAccountStatus(accSuspended); status != "suspended" {
			t.Errorf("expected 'suspended', got '%s'", status)
		}
	})

	t.Run("when RequirePhoneVerification is false (current transition state)", func(t *testing.T) {
		config.Envs.RequirePhoneVerification = false

		accNoEmail := &models.Account{
			EmailVerifiedAt: nil,
			PhoneVerifiedAt: nil,
		}
		if status := service.EvaluateAccountStatus(accNoEmail); status != "inactive" {
			t.Errorf("expected 'inactive' when email is not verified, got '%s'", status)
		}

		accWithEmail := &models.Account{
			EmailVerifiedAt: &now,
			PhoneVerifiedAt: nil,
		}
		if status := service.EvaluateAccountStatus(accWithEmail); status != "active" {
			t.Errorf("expected 'active' when email is verified and phone is not required, got '%s'", status)
		}
	})

	t.Run("when RequirePhoneVerification is true (future full gate state)", func(t *testing.T) {
		config.Envs.RequirePhoneVerification = true
		defer func() {
			config.Envs.RequirePhoneVerification = false
		}()

		accOnlyEmail := &models.Account{
			EmailVerifiedAt: &now,
			PhoneVerifiedAt: nil,
		}
		if status := service.EvaluateAccountStatus(accOnlyEmail); status != "inactive" {
			t.Errorf("expected 'inactive' when phone is required but missing, got '%s'", status)
		}

		accBoth := &models.Account{
			EmailVerifiedAt: &now,
			PhoneVerifiedAt: &now,
		}
		if status := service.EvaluateAccountStatus(accBoth); status != "active" {
			t.Errorf("expected 'active' when both email and phone are verified, got '%s'", status)
		}
	})
}
