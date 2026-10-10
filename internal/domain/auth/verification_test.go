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

func TestIsEmailStrictValidation(t *testing.T) {
	testCases := []struct {
		email    string
		expected bool
	}{
		{"user@gmail.com", true},
		{"john.doe@toentoen.com", true},
		{"customer_123@yahoo.co.id", true},
		{"va.ricky.joooooooooooo@gmail.com", false}, // Karakter berulang 5+ huruf 'o'
		{"fake.user.aaaaaaa@gmail.com", false},       // Karakter berulang 5+ huruf 'a'
		{"test@gmaill.com", false},                   // Typo domain gmaill.com
		{"test@gamil.com", false},                    // Typo domain gamil.com
		{"test@gmai.com", false},                     // Typo domain gmai.com
		{"test@yaho.com", false},                     // Typo domain yaho.com
		{"test@mailinator.com", false},               // Disposable email domain
		{"test@tempmail.com", false},                 // Disposable email domain
		{"invalid-format", false},
		{"@no-user.com", false},
		{"user@.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			got := isEmail(tc.email)
			if got != tc.expected {
				t.Errorf("isEmail(%q) = %v, expected %v", tc.email, got, tc.expected)
			}
		})
	}
}

func TestSendEmailOTP_AccountNotFound(t *testing.T) {
	mockRepo := &mockAuthRepo{
		accounts: make(map[string]*models.Account),
	}
	service := NewAuthService(mockRepo)

	err := service.SendEmailOTP("nonexistent@gmail.com", "customer")
	if err == nil {
		t.Fatal("expected error for non-existent account, got nil")
	}
	if err.Error() != "Akun dengan email ini tidak ditemukan" {
		t.Errorf("expected 'Akun dengan email ini tidak ditemukan', got '%s'", err.Error())
	}
}

