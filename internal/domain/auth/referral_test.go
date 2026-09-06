package auth

import (
	"errors"
	"testing"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type mockAuthRepo struct {
	referralCodes map[string]*models.ReferralUser
}

func (m *mockAuthRepo) CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error {
	return nil
}
func (m *mockAuthRepo) UpdateAccountStatus(identifier string, status string) error {
	return nil
}
func (m *mockAuthRepo) FindByIdentifier(identifier string) (*models.Account, error) {
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

	t.Run("owner with invalid referral code", func(t *testing.T) {
		invalidCode := "INVALID"
		input := models.RegisterUnifiedInput{
			Identifier:   "merchant@example.com",
			Password:     "secret123",
			Role:         "owner",
			ReferralCode: &invalidCode,
		}
		err := service.Register(input)
		if err == nil {
			t.Fatal("expected error for invalid referral code, got nil")
		}
		if err.Error() != "Kode referral tidak ditemukan" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
