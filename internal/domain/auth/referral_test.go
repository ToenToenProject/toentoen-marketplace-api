package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type mockAuthRepo struct {
	referralCodes map[string]*models.ReferralUser
	accounts      map[string]*models.Account
}

func (m *mockAuthRepo) CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error {
	return nil
}
func (m *mockAuthRepo) UpdateAccountStatus(identifier string, status string) error {
	return nil
}
func (m *mockAuthRepo) UpdateEmailVerified(identifier string, verifiedAt time.Time, newStatus string) error {
	return nil
}
func (m *mockAuthRepo) UpdatePhoneVerified(identifier string, verifiedAt time.Time, newStatus string) error {
	return nil
}
func (m *mockAuthRepo) FindByIdentifier(identifier string) (*models.Account, error) {
	if m.accounts != nil {
		if acc, ok := m.accounts[identifier]; ok {
			return acc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockAuthRepo) FindByID(id string) (*models.Account, error) {
	return nil, nil
}
func (m *mockAuthRepo) Update(account *models.Account) error {
	return nil
}
func (m *mockAuthRepo) UpdateEmail(accountID string, newEmail string) error {
	return nil
}
func (m *mockAuthRepo) UpdatePhoneNumber(accountID string, newPhone string) error {
	return nil
}
func (m *mockAuthRepo) LoadAssociations(account *models.Account) error {
	return nil
}
func (m *mockAuthRepo) FindReferralCode(code string) (*models.ReferralUser, error) {
	ref, exists := m.referralCodes[code]
	if !exists {
		return nil, errors.New("record not found")
	}
	return ref, nil
}

func TestCheckReferralCode(t *testing.T) {
	codePromo := "PROMO2026"
	codeExpired := "EXPIRED"
	mockRepo := &mockAuthRepo{
		referralCodes: map[string]*models.ReferralUser{
			"PROMO2026": {ReferralCode: &codePromo, IsActive: true},
			"EXPIRED":   {ReferralCode: &codeExpired, IsActive: false},
		},
	}
	service := NewAuthService(mockRepo)

	t.Run("empty code", func(t *testing.T) {
		res, err := service.CheckReferralCode("   ")
		if err == nil || res != nil {
			t.Fatalf("expected error for empty code, got res: %v", res)
		}
		if err.Error() != "Kode referral tidak boleh kosong" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("code not found", func(t *testing.T) {
		res, err := service.CheckReferralCode("UNKNOWN")
		if err == nil || res != nil {
			t.Fatalf("expected error for unknown code, got res: %v", res)
		}
		if err.Error() != "Kode referral tidak ditemukan" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("code inactive", func(t *testing.T) {
		res, err := service.CheckReferralCode("EXPIRED")
		if err == nil || res != nil {
			t.Fatalf("expected error for inactive code, got res: %v", res)
		}
		if err.Error() != "Kode referral tidak ditemukan" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("code valid and active", func(t *testing.T) {
		res, err := service.CheckReferralCode("PROMO2026")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Code != "PROMO2026" || !res.IsValid {
			t.Errorf("unexpected response: %+v", res)
		}
	})
}

func TestRegisterWithReferralCode(t *testing.T) {
	codePromo := "PROMO2026"
	mockRepo := &mockAuthRepo{
		referralCodes: map[string]*models.ReferralUser{
			"PROMO2026": {ReferralCode: &codePromo, IsActive: true},
		},
	}
	service := NewAuthService(mockRepo)

	t.Run("registration with phone number is supported", func(t *testing.T) {
		input := models.RegisterUnifiedInput{
			Identifier: "081234567890",
			Password:   "secret123",
			Role:       "customer",
		}
		err := service.Register(input)
		if err != nil {
			t.Fatalf("expected nil error for phone number registration, got: %v", err)
		}
	})

	t.Run("registration with already active account", func(t *testing.T) {
		activeEmail := "active@example.com"
		repoWithActive := &mockAuthRepo{
			accounts: map[string]*models.Account{
				activeEmail: {
					Email:         &activeEmail,
					AccountStatus: "active",
				},
			},
		}
		svc := NewAuthService(repoWithActive)
		err := svc.Register(models.RegisterUnifiedInput{
			Identifier: activeEmail,
			Password:   "secret123",
			Role:       "customer",
		})
		if err == nil || err.Error() != "Akun dengan email ini sudah terdaftar dan aktif. Silakan langsung login." {
			t.Fatalf("expected already active error, got: %v", err)
		}
	})

	t.Run("verify register OTP with expired or missing OTP", func(t *testing.T) {
		token, err := service.VerifyRegisterOTP(models.VerifyRegisterOTPInput{
			Identifier: "nonexistent@example.com",
			OTP:        "999999",
		})
		if err == nil {
			t.Fatal("expected error for invalid or missing OTP, got nil")
		}
		if token != "" {
			t.Errorf("expected empty token on error, got: %s", token)
		}
	})
}

func TestLogin(t *testing.T) {
	password := "secret123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	hashStr := string(hash)

	phone := "081234567890"
	email := "unverified@example.com"

	mockRepo := &mockAuthRepo{
		accounts: map[string]*models.Account{
			"081234567890": {
				AccountID:     "ACC-1",
				PhoneNumber:   &phone,
				PasswordHash:  &hashStr,
				Role:          "customer",
				AccountStatus: "inactive", // Status belum verified
			},
			"unverified@example.com": {
				AccountID:     "ACC-2",
				Email:         &email,
				PasswordHash:  &hashStr,
				Role:          "owner",
				AccountStatus: "inactive", // Status belum verified
			},
			"suspended@example.com": {
				AccountID:     "ACC-3",
				Email:         &email,
				PasswordHash:  &hashStr,
				Role:          "customer",
				AccountStatus: "suspended",
			},
			"banned@example.com": {
				AccountID:     "ACC-4",
				Email:         &email,
				PasswordHash:  &hashStr,
				Role:          "customer",
				AccountStatus: "banned",
			},
		},
	}
	service := NewAuthService(mockRepo)

	t.Run("inactive account with phone number (belum verified) is allowed to login", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier: "081234567890",
			Password:   "secret123",
		})
		if err != nil {
			t.Fatalf("expected login to succeed for unverified phone account, got: %v", err)
		}
	})

	t.Run("inactive account with email (belum verified) is allowed to login", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier: "unverified@example.com",
			Password:   "secret123",
		})
		if err != nil {
			t.Fatalf("expected login to succeed for unverified email account, got: %v", err)
		}
	})

	t.Run("suspended account is rejected from login", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier: "suspended@example.com",
			Password:   "secret123",
		})
		if err == nil || !strings.Contains(err.Error(), "ditangguhkan") {
			t.Fatalf("expected suspended error, got: %v", err)
		}
	})

	t.Run("banned account is rejected from login", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier: "banned@example.com",
			Password:   "secret123",
		})
		if err == nil || !strings.Contains(err.Error(), "diblokir") {
			t.Fatalf("expected banned error, got: %v", err)
		}
	})

	t.Run("wrong password returns error", func(t *testing.T) {
		err := service.Login(models.LoginUnifiedInput{
			Identifier: "081234567890",
			Password:   "wrongpassword",
		})
		if err == nil || err.Error() != "Email atau password salah" {
			t.Fatalf("expected invalid credentials error, got: %v", err)
		}
	})
}

