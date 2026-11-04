package auth

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
)

func TestReviewerHelpers(t *testing.T) {
	t.Run("isReviewerIdentifier matches expected tester accounts", func(t *testing.T) {
		testers := []string{
			"google.tester@toentoen.com",
			"GOOGLE.TESTER@TOENTOEN.COM",
			"reviewer@toentoen.com",
			"demo.reviewer@toentoen.com",
			"081299998888",
		}
		for _, email := range testers {
			if !isReviewerIdentifier(email) {
				t.Errorf("expected %s to be recognized as reviewer", email)
			}
		}

		normals := []string{
			"user@gmail.com",
			"customer@toentoen.com",
			"john@example.com",
			"081234567890",
		}
		for _, email := range normals {
			if isReviewerIdentifier(email) {
				t.Errorf("expected %s NOT to be recognized as reviewer", email)
			}
		}
	})

	t.Run("isReviewerPassword matches allowed passwords", func(t *testing.T) {
		validPasswords := []string{
			DefaultReviewerPassword,
			"Reviewer2026!",
			"Password123!",
		}
		for _, p := range validPasswords {
			if !isReviewerPassword(p) {
				t.Errorf("expected password %s to be valid for reviewer", p)
			}
		}

		if isReviewerPassword("wrongpassword") {
			t.Errorf("expected 'wrongpassword' to be rejected")
		}
	})
}

func TestReviewerLoginFlow(t *testing.T) {
	now := time.Now()
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte(DefaultReviewerPassword), bcrypt.DefaultCost)
	hashStr := string(hashBytes)
	emailStr := DefaultReviewerEmail

	reviewerAcc := &models.Account{
		AccountID:       "acc-google-reviewer-test",
		Email:           &emailStr,
		Username:        "googletester",
		PasswordHash:    &hashStr,
		Role:            "customer",
		AccountStatus:   "active",
		EmailVerifiedAt: &now,
	}

	mockRepo := &mockAuthRepo{
		accounts: map[string]*models.Account{
			DefaultReviewerEmail: reviewerAcc,
		},
	}
	service := NewAuthService(mockRepo)

	t.Run("Login reviewer returns nil error and triggers static OTP", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier: DefaultReviewerEmail,
			Password:   DefaultReviewerPassword,
		})
		if err != nil {
			t.Fatalf("expected reviewer login to succeed, got: %v", err)
		}
	})

	t.Run("VerifyLoginOTP accepts DefaultReviewerOTP 123456", func(t *testing.T) {
		if config.Envs == nil {
			config.Envs = &config.AppConfig{
				JWTSecret: "test-secret-key-12345678901234567890",
			}
		}

		token, acc, err := service.VerifyLoginOTP(models.VerifyLoginOTPInput{
			Identifier: DefaultReviewerEmail,
			OTP:        DefaultReviewerOTP,
		})
		if err != nil {
			t.Fatalf("expected reviewer OTP verification to succeed, got: %v", err)
		}
		if token == "" {
			t.Errorf("expected non-empty JWT token")
		}
		if acc == nil || acc.Role != "customer" {
			t.Errorf("expected valid customer account, got %+v", acc)
		}
	})
}
