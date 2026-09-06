package account

import (
	"errors"
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

type mockProfileRepo struct {
	account       *models.Account
	owner         *models.Owner
	referralCodes map[string]*models.ReferralUser
}

func (m *mockProfileRepo) GetAccountByID(accountID string) (*models.Account, error) {
	return m.account, nil
}
func (m *mockProfileRepo) GetAccountByIdentifier(identifier string) (*models.Account, error) {
	return m.account, nil
}
func (m *mockProfileRepo) GetCustomerByAccountID(accountID string) (*models.Customer, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetOwnerByAccountID(accountID string) (*models.Owner, error) {
	return m.owner, nil
}
func (m *mockProfileRepo) GetBusinessAdminByAccountID(accountID string) (*models.BusinessAdmin, error) {
	return nil, nil
}
func (m *mockProfileRepo) SaveCustomerProfile(account *models.Account, customer *models.Customer) error {
	return nil
}
func (m *mockProfileRepo) SaveOwnerProfile(account *models.Account, owner *models.Owner) error {
	return nil
}
func (m *mockProfileRepo) SaveBusinessAdminProfile(account *models.Account, businessAdmin *models.BusinessAdmin) error {
	return nil
}
func (m *mockProfileRepo) DeleteAccountTx(accountID string, role string) ([]string, []string, error) {
	return nil, nil, nil
}
func (m *mockProfileRepo) CheckUsernameExists(username string, excludeAccountID string) (bool, error) {
	return false, nil
}
func (m *mockProfileRepo) CheckEmailExists(email string, excludeAccountID string) (bool, error) {
	return false, nil
}
func (m *mockProfileRepo) CheckPhoneNumberExists(phone string, excludeAccountID string) (bool, error) {
	return false, nil
}
func (m *mockProfileRepo) CheckKtpNumberExists(ktp string, excludeAccountID string) (bool, error) {
	return false, nil
}
func (m *mockProfileRepo) FindReferralCode(code string) (*models.ReferralUser, error) {
	ref, exists := m.referralCodes[code]
	if !exists {
		return nil, errors.New("record not found")
	}
	return ref, nil
}

func TestUpdateOwnerProfileReferralCode(t *testing.T) {
	codePromo := "PROMO2026"
	referralCodes := map[string]*models.ReferralUser{
		"PROMO2026": {ReferralCode: &codePromo, IsActive: true},
	}

	t.Run("owner already has referral code - reject change", func(t *testing.T) {
		existingCode := "PROMO2026"
		newCode := "NEWCODE"
		repo := &mockProfileRepo{
			account:       &models.Account{AccountID: "ACC_1", Role: "owner"},
			owner:         &models.Owner{AccountID: "ACC_1", ReferralCode: &existingCode},
			referralCodes: referralCodes,
		}
		service := NewProfileService(repo)

		_, err := service.UpdateOwnerProfile("ACC_1", models.UpdateProfileOwnerInput{
			ReferralCode: &newCode,
		})
		if err == nil {
			t.Fatal("expected error when modifying existing referral code, got nil")
		}
		if err.Error() != "Kode referral sudah terisi dan tidak dapat diubah kembali" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("owner already has referral code - allow identical code", func(t *testing.T) {
		existingCode := "PROMO2026"
		sameCode := "promo2026"
		repo := &mockProfileRepo{
			account:       &models.Account{AccountID: "ACC_1", Role: "owner"},
			owner:         &models.Owner{AccountID: "ACC_1", ReferralCode: &existingCode},
			referralCodes: referralCodes,
		}
		service := NewProfileService(repo)

		_, err := service.UpdateOwnerProfile("ACC_1", models.UpdateProfileOwnerInput{
			ReferralCode: &sameCode,
		})
		if err != nil {
			t.Fatalf("unexpected error when sending identical code: %v", err)
		}
	})

	t.Run("owner has no referral code - reject invalid code", func(t *testing.T) {
		invalidCode := "UNKNOWN"
		repo := &mockProfileRepo{
			account:       &models.Account{AccountID: "ACC_1", Role: "owner"},
			owner:         &models.Owner{AccountID: "ACC_1", ReferralCode: nil},
			referralCodes: referralCodes,
		}
		service := NewProfileService(repo)

		_, err := service.UpdateOwnerProfile("ACC_1", models.UpdateProfileOwnerInput{
			ReferralCode: &invalidCode,
		})
		if err == nil {
			t.Fatal("expected error for unknown referral code, got nil")
		}
		if err.Error() != "Kode referral tidak ditemukan" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("owner has no referral code - successfully set valid code", func(t *testing.T) {
		validCode := "PROMO2026"
		repo := &mockProfileRepo{
			account:       &models.Account{AccountID: "ACC_1", Role: "owner"},
			owner:         &models.Owner{AccountID: "ACC_1", ReferralCode: nil},
			referralCodes: referralCodes,
		}
		service := NewProfileService(repo)

		_, err := service.UpdateOwnerProfile("ACC_1", models.UpdateProfileOwnerInput{
			ReferralCode: &validCode,
		})
		if err != nil {
			t.Fatalf("unexpected error setting valid code: %v", err)
		}
		if repo.owner.ReferralCode == nil || *repo.owner.ReferralCode != "PROMO2026" {
			t.Errorf("expected referral code PROMO2026, got: %v", repo.owner.ReferralCode)
		}
	})
}
