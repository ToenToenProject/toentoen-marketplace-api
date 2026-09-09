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

	t.Run("account with verified email is active", func(t *testing.T) {
		accNoEmail := &models.Account{
			EmailVerifiedAt: nil,
		}
		if status := service.EvaluateAccountStatus(accNoEmail); status != "inactive" {
			t.Errorf("expected 'inactive' when email is not verified, got '%s'", status)
		}

		accWithEmail := &models.Account{
			EmailVerifiedAt: &now,
		}
		if status := service.EvaluateAccountStatus(accWithEmail); status != "active" {
			t.Errorf("expected 'active' when email is verified, got '%s'", status)
		}
	})
}
