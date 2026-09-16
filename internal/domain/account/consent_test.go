package account

import (
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"
)

func TestConsentService(t *testing.T) {
	t.Run("UpdateConsent rejects when accepted_terms is false or missing", func(t *testing.T) {
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_TEST", Role: "owner"},
		}
		service := NewProfileService(repo)

		disagreed := false
		agreed := true

		// 1. accepted_terms is false
		_, err := service.UpdateConsent("ACC_TEST", models.ConsentInput{
			AcceptedTerms:   &disagreed,
			AcceptedPrivacy: &agreed,
		})
		if err == nil || err.Error() != "Anda wajib menyetujui Syarat & Ketentuan untuk melanjutkan" {
			t.Fatalf("expected error for rejected terms, got: %v", err)
		}

		// 2. accepted_terms is nil
		_, errNil := service.UpdateConsent("ACC_TEST", models.ConsentInput{
			AcceptedTerms:   nil,
			AcceptedPrivacy: &agreed,
		})
		if errNil == nil || errNil.Error() != "Anda wajib menyetujui Syarat & Ketentuan untuk melanjutkan" {
			t.Fatalf("expected error for nil terms, got: %v", errNil)
		}
	})

	t.Run("UpdateConsent rejects when accepted_privacy is false or missing", func(t *testing.T) {
		repo := &mockProfileRepo{
			account: &models.Account{AccountID: "ACC_TEST", Role: "owner"},
		}
		service := NewProfileService(repo)

		disagreed := false
		agreed := true

		// 1. accepted_privacy is false
		_, err := service.UpdateConsent("ACC_TEST", models.ConsentInput{
			AcceptedTerms:   &agreed,
			AcceptedPrivacy: &disagreed,
		})
		if err == nil || err.Error() != "Anda wajib menyetujui Kebijakan Privasi untuk melanjutkan" {
			t.Fatalf("expected error for rejected privacy, got: %v", err)
		}

		// 2. accepted_privacy is nil
		_, errNil := service.UpdateConsent("ACC_TEST", models.ConsentInput{
			AcceptedTerms:   &agreed,
			AcceptedPrivacy: nil,
		})
		if errNil == nil || errNil.Error() != "Anda wajib menyetujui Kebijakan Privasi untuk melanjutkan" {
			t.Fatalf("expected error for nil privacy, got: %v", errNil)
		}
	})

	t.Run("UpdateConsent successfully saves consent with custom versions", func(t *testing.T) {
		acc := &models.Account{AccountID: "ACC_TEST", Role: "owner"}
		repo := &mockProfileRepo{
			account: acc,
		}
		service := NewProfileService(repo)

		agreed := true
		tVer := "v2.0"
		pVer := "v1.5"

		updated, err := service.UpdateConsent("ACC_TEST", models.ConsentInput{
			AcceptedTerms:   &agreed,
			TermsVersion:    &tVer,
			AcceptedPrivacy: &agreed,
			PrivacyVersion:  &pVer,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updated.AcceptedTermsAt == nil || updated.AcceptedPrivacyAt == nil {
			t.Fatalf("expected consent timestamps to be populated, got %+v", updated)
		}
		if updated.TermsVersion == nil || *updated.TermsVersion != "v2.0" {
			t.Errorf("expected terms version 'v2.0', got %v", updated.TermsVersion)
		}
		if updated.PrivacyVersion == nil || *updated.PrivacyVersion != "v1.5" {
			t.Errorf("expected privacy version 'v1.5', got %v", updated.PrivacyVersion)
		}
	})

	t.Run("GetKYCStatus reports consent status accurately", func(t *testing.T) {
		now := time.Now()
		ver := "v1.0"
		repo := &mockProfileRepo{
			account: &models.Account{
				AccountID:         "ACC_CONSENTED",
				Role:              "owner",
				AcceptedTermsAt:   &now,
				TermsVersion:      &ver,
				AcceptedPrivacyAt: &now,
				PrivacyVersion:    &ver,
			},
			owner: &models.Owner{AccountID: "ACC_CONSENTED", KycStatus: "pending"},
		}
		service := NewProfileService(repo)

		status, err := service.GetKYCStatus("ACC_CONSENTED")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !status.HasAcceptedTerms || !status.HasAcceptedPrivacy {
			t.Errorf("expected HasAcceptedTerms and HasAcceptedPrivacy to be true, got %+v", status)
		}
		if status.TermsVersion == nil || *status.TermsVersion != "v1.0" {
			t.Errorf("expected termsVersion 'v1.0', got %v", status.TermsVersion)
		}
	})
}
