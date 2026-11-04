package auth

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

const (
	// DefaultReviewerEmail adalah alamat email resmi untuk pengujian Google Play Reviewer
	DefaultReviewerEmail = "google.tester@toentoen.com"

	// DefaultReviewerPassword adalah password untuk akun pengujian Google Play
	DefaultReviewerPassword = "ToenToen2026!"

	// DefaultReviewerOTP adalah kode OTP statis khusus akun pengujian Google Play
	DefaultReviewerOTP = "123456"
)

// isReviewerIdentifier memeriksa apakah identifier (email/no HP) adalah akun pengujian Google Play
func isReviewerIdentifier(identifier string) bool {
	clean := strings.ToLower(strings.TrimSpace(cleanIdentifier(identifier)))
	if clean == "" {
		return false
	}
	envTester := strings.ToLower(strings.TrimSpace(os.Getenv("GOOGLE_REVIEWER_EMAIL")))
	if envTester != "" && clean == envTester {
		return true
	}
	return clean == "google.tester@toentoen.com" ||
		clean == "reviewer@toentoen.com" ||
		clean == "demo.reviewer@toentoen.com" ||
		clean == "081299998888" ||
		clean == "6281299998888"
}

// isReviewerPassword memvalidasi kata sandi untuk akun pengujian Google Play
func isReviewerPassword(password string) bool {
	p := strings.TrimSpace(password)
	return p == DefaultReviewerPassword || p == "Reviewer2026!" || p == "Password123!"
}

// ensureReviewerAccountExists memastikan akun pengujian Google Play selalu tersedia di basis data
func (s *authService) ensureReviewerAccountExists(identifier string) (*models.Account, error) {
	acc, err := s.repo.FindByIdentifier(identifier, models.RoleGroupCustomer)
	if err == nil && acc != nil {
		return acc, nil
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(DefaultReviewerPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hashBytes)

	now := time.Now()
	termsVer := "v1.0"
	privacyVer := "v1.0"
	emailVal := DefaultReviewerEmail
	phoneVal := "081299998888"

	if isEmail(identifier) {
		emailVal = strings.ToLower(strings.TrimSpace(identifier))
	} else if cleanIdentifier(identifier) != "" {
		phoneVal = cleanIdentifier(identifier)
	}

	newAcc := &models.Account{
		AccountID:         uuid.New().String(),
		AuthProvider:      "local",
		Email:             &emailVal,
		PhoneNumber:       &phoneVal,
		FirstName:         "Google",
		LastName:          "Reviewer",
		Username:          "googletester_" + utils.GenerateNumericOTP(4),
		PasswordHash:      &hashStr,
		Role:              "customer",
		AccountStatus:     "active",
		EmailVerifiedAt:   &now,
		PhoneVerifiedAt:   &now,
		AcceptedTermsAt:   &now,
		TermsVersion:      &termsVer,
		AcceptedPrivacyAt: &now,
		PrivacyVersion:    &privacyVer,
	}

	if err := s.repo.CreateAccountWithProfile(newAcc, "customer"); err != nil {
		log.Printf("[REVIEWER-SETUP] Gagal auto-provision akun reviewer: %v", err)
		return nil, err
	}
	log.Printf("ℹ️ [REVIEWER-SETUP] Akun Google Play Reviewer berhasil dibuat otomatis: %s (Role: customer)", emailVal)
	return newAcc, nil
}
