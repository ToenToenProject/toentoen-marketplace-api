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

	reviewerCust := &models.Account{
		AccountID:       "acc-google-reviewer-cust",
		Email:           &emailStr,
		Username:        "googletester_cust",
		PasswordHash:    &hashStr,
		Role:            "customer",
		AccountStatus:   "active",
		EmailVerifiedAt: &now,
	}

	reviewerOwner := &models.Account{
		AccountID:       "acc-google-reviewer-owner",
		Email:           &emailStr,
		Username:        "googletester_owner",
		PasswordHash:    &hashStr,
		Role:            "owner",
		AccountStatus:   "active",
		EmailVerifiedAt: &now,
	}

	reviewerAdmin := &models.Account{
		AccountID:       "acc-google-reviewer-admin",
		Email:           &emailStr,
		Username:        "googletester_admin",
		PasswordHash:    &hashStr,
		Role:            "business_admin",
		AccountStatus:   "active",
		EmailVerifiedAt: &now,
	}

	mockRepo := &mockAuthRepo{
		accounts: map[string]*models.Account{
			DefaultReviewerEmail + ":customer":       reviewerCust,
			DefaultReviewerEmail + ":owner":          reviewerOwner,
			DefaultReviewerEmail + ":business_admin": reviewerAdmin,
		},
	}
	service := NewAuthService(mockRepo)

	if config.Envs == nil {
		config.Envs = &config.AppConfig{
			JWTSecret: "test-secret-key-12345678901234567890",
		}
	}

	t.Run("Customer login and OTP verification", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier:   DefaultReviewerEmail,
			Password:     DefaultReviewerPassword,
			ExpectedRole: "customer",
		})
		if err != nil {
			t.Fatalf("expected customer login to succeed, got: %v", err)
		}

		token, acc, err := service.VerifyLoginOTP(models.VerifyLoginOTPInput{
			Identifier:   DefaultReviewerEmail,
			OTP:          DefaultReviewerOTP,
			ExpectedRole: "customer",
		})
		if err != nil {
			t.Fatalf("expected OTP verification to succeed, got: %v", err)
		}
		if token == "" || acc.Role != "customer" {
			t.Errorf("expected valid customer session, got role %v", acc.Role)
		}
	})

	t.Run("Owner login and OTP verification", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier:   DefaultReviewerEmail,
			Password:     DefaultReviewerPassword,
			ExpectedRole: "owner",
		})
		if err != nil {
			t.Fatalf("expected owner login to succeed, got: %v", err)
		}

		token, acc, err := service.VerifyLoginOTP(models.VerifyLoginOTPInput{
			Identifier:   DefaultReviewerEmail,
			OTP:          DefaultReviewerOTP,
			ExpectedRole: "owner",
		})
		if err != nil {
			t.Fatalf("expected OTP verification to succeed, got: %v", err)
		}
		if token == "" || acc.Role != "owner" {
			t.Errorf("expected valid owner session, got role %v", acc.Role)
		}
	})

	t.Run("Business Admin login and OTP verification", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier:   DefaultReviewerEmail,
			Password:     DefaultReviewerPassword,
			ExpectedRole: "business_admin",
		})
		if err != nil {
			t.Fatalf("expected business_admin login to succeed, got: %v", err)
		}

		token, acc, err := service.VerifyLoginOTP(models.VerifyLoginOTPInput{
			Identifier:   DefaultReviewerEmail,
			OTP:          DefaultReviewerOTP,
			ExpectedRole: "business_admin",
		})
		if err != nil {
			t.Fatalf("expected OTP verification to succeed, got: %v", err)
		}
		if token == "" || acc.Role != "business_admin" {
			t.Errorf("expected valid admin session, got role %v", acc.Role)
		}
	})
}
