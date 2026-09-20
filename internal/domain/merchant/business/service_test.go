package business

import (
	"errors"
	"strings"
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

type mockBusinessRepo struct {
	BusinessRepository
	emailVerified bool
	kycStatus     string
	getKYCErr     error
	createdBiz    *models.Business
}

func (m *mockBusinessRepo) GetOwnerKYCStatus(ownerID string) (bool, string, error) {
	if m.getKYCErr != nil {
		return false, "", m.getKYCErr
	}
	return m.emailVerified, m.kycStatus, nil
}

func (m *mockBusinessRepo) CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour) error {
	m.createdBiz = business
	return nil
}

func (m *mockBusinessRepo) HasNIBTag(tagIDs []int64) (bool, error) {
	return false, nil
}

func (m *mockBusinessRepo) RemoveNIBTagFromBusiness(businessID string) error {
	return nil
}

func TestCreateBusiness_OwnerRestrictions(t *testing.T) {
	input := models.CreateBusinessInput{
		BusinessName: "Toko Berkah",
		Email:        "owner@example.com",
		CategoryIDs:  []int64{1},
	}

	t.Run("fails when owner account is not found", func(t *testing.T) {
		repo := &mockBusinessRepo{
			getKYCErr: errors.New("record not found"),
		}
		service := NewBusinessService(repo, nil)

		_, err := service.CreateBusiness("OWNER_1", input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "tidak ditemukan") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails when email is not verified", func(t *testing.T) {
		repo := &mockBusinessRepo{
			emailVerified: false,
			kycStatus:     "pending",
		}
		service := NewBusinessService(repo, nil)

		_, err := service.CreateBusiness("OWNER_1", input)
		if err == nil {
			t.Fatal("expected error when email not verified, got nil")
		}
		if !strings.Contains(err.Error(), "Email akun Anda belum terverifikasi") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails when KYC is unverified", func(t *testing.T) {
		repo := &mockBusinessRepo{
			emailVerified: true,
			kycStatus:     "unverified",
		}
		service := NewBusinessService(repo, nil)

		_, err := service.CreateBusiness("OWNER_1", input)
		if err == nil {
			t.Fatal("expected error when KYC is unverified, got nil")
		}
		if !strings.Contains(err.Error(), "belum dikirimkan") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fails when KYC is rejected", func(t *testing.T) {
		repo := &mockBusinessRepo{
			emailVerified: true,
			kycStatus:     "rejected",
		}
		service := NewBusinessService(repo, nil)

		_, err := service.CreateBusiness("OWNER_1", input)
		if err == nil {
			t.Fatal("expected error when KYC is rejected, got nil")
		}
		if !strings.Contains(err.Error(), "ditolak") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("succeeds when email is verified and KYC is pending", func(t *testing.T) {
		repo := &mockBusinessRepo{
			emailVerified: true,
			kycStatus:     "pending",
		}
		service := NewBusinessService(repo, nil)

		bizID, err := service.CreateBusiness("OWNER_1", input)
		if err != nil {
			t.Fatalf("expected success when KYC is pending, got error: %v", err)
		}
		if bizID == "" {
			t.Errorf("expected non-empty business ID, got empty")
		}
		if repo.createdBiz == nil || *repo.createdBiz.BusinessName != "Toko Berkah" {
			t.Errorf("expected business to be created, got: %+v", repo.createdBiz)
		}
	})

	t.Run("succeeds when email is verified and KYC is verified", func(t *testing.T) {
		repo := &mockBusinessRepo{
			emailVerified: true,
			kycStatus:     "verified",
		}
		service := NewBusinessService(repo, nil)

		bizID, err := service.CreateBusiness("OWNER_1", input)
		if err != nil {
			t.Fatalf("expected success when KYC is verified, got error: %v", err)
		}
		if bizID == "" {
			t.Errorf("expected non-empty business ID, got empty")
		}
	})
}
