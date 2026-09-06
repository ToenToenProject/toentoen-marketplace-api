package account

import (
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

func TestKYCService(t *testing.T) {
	t.Run("GetKYCStatus for customer returns verified automatically", func(t *testing.T) {
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_CUST", Role: "customer"},
		}
		service := NewProfileService(repo)

		status, err := service.GetKYCStatus("ACC_CUST")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.KycStatus != "verified" || !status.IsVerified {
			t.Errorf("expected customer to be verified, got %+v", status)
		}
	})

	t.Run("GetKYCStatus for unverified owner", func(t *testing.T) {
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_OWNER", Role: "owner"},
			owner:   &models.Owner{AccountID: "ACC_OWNER", KycStatus: "unverified"},
		}
		service := NewProfileService(repo)

		status, err := service.GetKYCStatus("ACC_OWNER")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.KycStatus != "unverified" || status.IsVerified {
			t.Errorf("expected unverified owner, got %+v", status)
		}
	})

	t.Run("SubmitKYC rejects non-merchant role", func(t *testing.T) {
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_CUST", Role: "customer"},
		}
		service := NewProfileService(repo)

		err := service.SubmitKYC("ACC_CUST", models.SubmitKYCInput{
			KtpNumber:   "1234567890123456",
			KtpPhotoUrl: "https://example.com/ktp.jpg",
		})
		if err == nil {
			t.Fatal("expected error for customer submitting KYC, got nil")
		}
	})

	t.Run("SubmitKYC validates 16-digit NIK", func(t *testing.T) {
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_OWNER", Role: "owner"},
			owner:   &models.Owner{AccountID: "ACC_OWNER"},
		}
		service := NewProfileService(repo)

		err := service.SubmitKYC("ACC_OWNER", models.SubmitKYCInput{
			KtpNumber:   "12345",
			KtpPhotoUrl: "https://example.com/ktp.jpg",
		})
		if err == nil {
			t.Fatal("expected error for invalid NIK length, got nil")
		}
	})

	t.Run("SubmitKYC successfully updates status to pending", func(t *testing.T) {
		owner := &models.Owner{AccountID: "ACC_OWNER", KycStatus: "unverified"}
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_OWNER", Role: "owner"},
			owner:   owner,
		}
		service := NewProfileService(repo)

		err := service.SubmitKYC("ACC_OWNER", models.SubmitKYCInput{
			KtpNumber:   "3171012345670001",
			KtpPhotoUrl: "https://s3.amazonaws.com/toentoen/ktp/owner.jpg",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
