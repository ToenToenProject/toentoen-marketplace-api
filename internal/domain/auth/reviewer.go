package auth

import (
	"fmt"
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

// ensureReviewerAccounts memastikan akun pengujian Google Play selalu tersedia untuk seluruh role:
// customer, owner, dan business_admin.
func (s *authService) ensureReviewerAccounts(identifier string) error {
	roles := []string{"customer", "owner", "business_admin"}
	var lastErr error

	for _, role := range roles {
		group := models.RoleGroupOf(role)
		acc, err := s.repo.FindByIdentifier(identifier, group)
		if err == nil && acc != nil {
			continue
		}

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(DefaultReviewerPassword), bcrypt.DefaultCost)
		if err != nil {
			lastErr = err
			continue
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

		var roleSuffix string
		switch role {
		case "owner":
			roleSuffix = "own"
		case "business_admin":
			roleSuffix = "adm"
		default:
			roleSuffix = "cust"
		}

		newAcc := &models.Account{
			AccountID:         uuid.New().String(),
			AuthProvider:      "local",
			Email:             &emailVal,
			PhoneNumber:       &phoneVal,
			FirstName:         "Google",
			LastName:          "Reviewer",
			Username:          fmt.Sprintf("googletester_%s_%s", roleSuffix, utils.GenerateNumericOTP(4)),
			PasswordHash:      &hashStr,
			Role:              role,
			AccountStatus:     "active",
			EmailVerifiedAt:   &now,
			PhoneVerifiedAt:   &now,
			AcceptedTermsAt:   &now,
			TermsVersion:      &termsVer,
			AcceptedPrivacyAt: &now,
			PrivacyVersion:    &privacyVer,
		}

		if role == "owner" {
			newAcc.Owner = &models.Owner{
				AccountID:     newAcc.AccountID,
				KycStatus:     "verified",
				KycVerifiedAt: &now,
				KtpName:       &newAcc.FirstName,
			}
		} else if role == "business_admin" {
			newAcc.BusinessAdmin = &models.BusinessAdmin{
				AccountID:     newAcc.AccountID,
				KycStatus:     "verified",
				KycVerifiedAt: &now,
				KtpName:       &newAcc.FirstName,
			}
		}

		if err := s.repo.CreateAccountWithProfile(newAcc, role); err != nil {
			log.Printf("[REVIEWER-SETUP] Gagal auto-provision akun reviewer role %s: %v", role, err)
			lastErr = err
			continue
		}
		log.Printf("ℹ️ [REVIEWER-SETUP] Akun Google Play Reviewer berhasil dibuat otomatis: %s (Role: %s)", emailVal, role)
	}

	return lastErr
}

// ensureReviewerAccountExists mempertahankan kompatibilitas lama
func (s *authService) ensureReviewerAccountExists(identifier string) (*models.Account, error) {
	_ = s.ensureReviewerAccounts(identifier)
	return s.repo.FindByIdentifier(identifier, models.RoleGroupCustomer)
}
