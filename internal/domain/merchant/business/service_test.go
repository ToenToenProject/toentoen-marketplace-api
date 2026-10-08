package business

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

type mockBusinessRepo struct {
	BusinessRepository
	emailVerified      bool
	kycStatus          string
	getKYCErr          error
	createdBiz         *models.Business
	updatedBiz         *models.Business
	findBiz            *models.Business
	confirmedOpenBizID string
	confirmedOpenDate  string
	confirmedOTBizID   string
	confirmedOTDate    string
	declinedBizID      string
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

func (m *mockBusinessRepo) FindBusinessByID(businessID string) (*models.Business, error) {
	if m.findBiz != nil {
		return m.findBiz, nil
	}
	return nil, errors.New("bisnis tidak ditemukan")
}

func (m *mockBusinessRepo) UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64) error {
	m.updatedBiz = business
	return nil
}

func (m *mockBusinessRepo) CheckHandleExists(handle, excludeBusinessID string) bool {
	return false
}

func (m *mockBusinessRepo) HasNIBTag(tagIDs []int64) (bool, error) {
	return false, nil
}

func (m *mockBusinessRepo) RemoveNIBTagFromBusiness(businessID string) error {
	return nil
}

func (m *mockBusinessRepo) ResetManualCloseStatus() error {
	return nil
}

func (m *mockBusinessRepo) ConfirmOpenToday(businessID, todayDate string) error {
	m.confirmedOpenBizID = businessID
	m.confirmedOpenDate = todayDate
	return nil
}

func (m *mockBusinessRepo) ConfirmOvertimeToday(businessID, todayDate string) error {
	m.confirmedOTBizID = businessID
	m.confirmedOTDate = todayDate
	return nil
}

func (m *mockBusinessRepo) DeclineSchedule(businessID string) error {
	m.declinedBizID = businessID
	return nil
}

func (m *mockBusinessRepo) GetBusinessesWithHours() ([]models.Business, error) {
	return nil, nil
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

func TestBusinessCertificate_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshals standard json format", func(t *testing.T) {
		raw := `{"type":"Halal","number":"ID123456","image_url":"https://storage.toentoen.com/cert.jpg"}`
		var cert models.BusinessCertificate
		if err := json.Unmarshal([]byte(raw), &cert); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if cert.Type != "Halal" || cert.Number != "ID123456" || cert.ImageURL != "https://storage.toentoen.com/cert.jpg" {
			t.Errorf("unexpected cert: %+v", cert)
		}
		if cert.Name != "Halal" || cert.URL != "https://storage.toentoen.com/cert.jpg" {
			t.Errorf("aliases not populated: %+v", cert)
		}
	})

	t.Run("unmarshals frontend alias keys (name, certificate_number, url)", func(t *testing.T) {
		raw := `{"name":"BPOM RI","certificate_number":"TR987654","url":"https://storage.toentoen.com/bpom.jpg"}`
		var cert models.BusinessCertificate
		if err := json.Unmarshal([]byte(raw), &cert); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if cert.Type != "BPOM RI" || cert.Number != "TR987654" || cert.ImageURL != "https://storage.toentoen.com/bpom.jpg" {
			t.Errorf("unexpected cert from aliases: %+v", cert)
		}
		if cert.CertificateNumber != "TR987654" || cert.CertificateName != "BPOM RI" {
			t.Errorf("aliases not populated: %+v", cert)
		}
	})

	t.Run("unmarshals plain string url", func(t *testing.T) {
		raw := `"https://storage.toentoen.com/simple_cert.png"`
		var cert models.BusinessCertificate
		if err := json.Unmarshal([]byte(raw), &cert); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if cert.ImageURL != "https://storage.toentoen.com/simple_cert.png" || cert.URL != "https://storage.toentoen.com/simple_cert.png" {
			t.Errorf("unexpected cert from string: %+v", cert)
		}
	})
}

func TestCreateBusiness_CertificatesPersistence(t *testing.T) {
	repo := &mockBusinessRepo{
		emailVerified: true,
		kycStatus:     "verified",
	}
	service := NewBusinessService(repo, nil)

	inputJSON := `{
		"business_name": "Toko Berkah Halal",
		"email": "owner@berkah.com",
		"certificate": [
			{
				"name": "Halal MUI",
				"certificate_number": "MUI-123456",
				"url": "https://storage.toentoen.com/mui.jpg"
			}
		]
	}`

	var input models.CreateBusinessInput
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		t.Fatalf("failed to unmarshal input: %v", err)
	}

	bizID, err := service.CreateBusiness("OWNER_1", input)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if bizID == "" {
		t.Errorf("expected non-empty business ID")
	}

	if repo.createdBiz == nil {
		t.Fatal("expected created business in repo")
	}
	if len(repo.createdBiz.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(repo.createdBiz.Certificates))
	}

	cert := repo.createdBiz.Certificates[0]
	if cert.Type != "Halal MUI" {
		t.Errorf("expected Type 'Halal MUI', got '%s'", cert.Type)
	}
	// Pastikan data sensitif dienkripsi sebelum disimpan ke DB
	if !strings.HasPrefix(cert.Number, "enc::") {
		t.Errorf("expected Number to be encrypted, got '%s'", cert.Number)
	}
	if !strings.HasPrefix(cert.ImageURL, "enc::") {
		t.Errorf("expected ImageURL to be encrypted, got '%s'", cert.ImageURL)
	}
}

func TestUpdateBusiness_CertificatesPersistence(t *testing.T) {
	existingBiz := &models.Business{
		BusinessID: "BIZ_123",
		AccountID:  "OWNER_1",
		Certificates: []models.BusinessCertificate{},
	}

	repo := &mockBusinessRepo{
		findBiz: existingBiz,
	}
	service := NewBusinessService(repo, nil)

	updateJSON := `{
		"certificates": [
			{
				"type": "HACCP",
				"number": "HACCP-999",
				"image_url": "https://storage.toentoen.com/haccp.png"
			}
		]
	}`

	var input models.UpdateBusinessInput
	if err := json.Unmarshal([]byte(updateJSON), &input); err != nil {
		t.Fatalf("failed to unmarshal update input: %v", err)
	}

	if err := service.UpdateBusiness("OWNER_1", "BIZ_123", input); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if repo.updatedBiz == nil {
		t.Fatal("expected updated business in repo")
	}
	if len(repo.updatedBiz.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(repo.updatedBiz.Certificates))
	}

	cert := repo.updatedBiz.Certificates[0]
	if cert.Type != "HACCP" {
		t.Errorf("expected Type 'HACCP', got '%s'", cert.Type)
	}
	if !strings.HasPrefix(cert.Number, "enc::") {
		t.Errorf("expected Number to be encrypted, got '%s'", cert.Number)
	}
}

func TestGetBusinessDetail_CertificatesDecryption(t *testing.T) {
	name := "Toko Kopi"
	existingBiz := &models.Business{
		BusinessID:   "BIZ_456",
		AccountID:    "OWNER_1",
		BusinessName: &name,
		Certificates: []models.BusinessCertificate{
			{
				Type:     "Sertifikat Halal",
				Number:   "enc::fakecipher", // atau plaintext fallback
				ImageURL: "https://storage.toentoen.com/cert.jpg",
			},
		},
	}

	repo := &mockBusinessRepo{
		findBiz: existingBiz,
	}
	service := NewBusinessService(repo, nil)

	detail, err := service.GetBusinessDetail("BIZ_456")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(detail.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(detail.Certificates))
	}
	if len(detail.CertificatesAlias) != 1 {
		t.Fatalf("expected CertificatesAlias to be populated")
	}
	if len(detail.BusinessCertificatesAlias) != 1 {
		t.Fatalf("expected BusinessCertificatesAlias to be populated")
	}

	cert := detail.Certificates[0]
	if cert.Type != "Sertifikat Halal" || cert.Name != "Sertifikat Halal" {
		t.Errorf("expected Name and Type to match 'Sertifikat Halal', got Type='%s' Name='%s'", cert.Type, cert.Name)
	}
	if cert.ImageURL != "https://storage.toentoen.com/cert.jpg" || cert.URL != "https://storage.toentoen.com/cert.jpg" {
		t.Errorf("expected ImageURL and URL to match, got ImageURL='%s' URL='%s'", cert.ImageURL, cert.URL)
	}
}

func TestConfirmScheduleAction(t *testing.T) {
	existingBiz := &models.Business{
		BusinessID: "BIZ_123",
		AccountID:  "OWNER_1",
	}

	t.Run("confirm_open calls ConfirmOpenToday", func(t *testing.T) {
		repo := &mockBusinessRepo{
			findBiz: existingBiz,
		}
		service := NewBusinessService(repo, nil)

		err := service.ConfirmScheduleAction("OWNER_1", "BIZ_123", models.ConfirmScheduleInput{
			Action: "confirm_open",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if repo.confirmedOpenBizID != "BIZ_123" {
			t.Errorf("expected confirmedOpenBizID BIZ_123, got %s", repo.confirmedOpenBizID)
		}
		if repo.confirmedOpenDate == "" {
			t.Errorf("expected confirmedOpenDate to be set, got empty")
		}
	})

	t.Run("confirm_overtime calls ConfirmOvertimeToday", func(t *testing.T) {
		repo := &mockBusinessRepo{
			findBiz: existingBiz,
		}
		service := NewBusinessService(repo, nil)

		err := service.ConfirmScheduleAction("OWNER_1", "BIZ_123", models.ConfirmScheduleInput{
			Action: "confirm_overtime",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if repo.confirmedOTBizID != "BIZ_123" {
			t.Errorf("expected confirmedOTBizID BIZ_123, got %s", repo.confirmedOTBizID)
		}
		if repo.confirmedOTDate == "" {
			t.Errorf("expected confirmedOTDate to be set, got empty")
		}
	})

	t.Run("decline calls DeclineSchedule", func(t *testing.T) {
		repo := &mockBusinessRepo{
			findBiz: existingBiz,
		}
		service := NewBusinessService(repo, nil)

		err := service.ConfirmScheduleAction("OWNER_1", "BIZ_123", models.ConfirmScheduleInput{
			Action: "decline",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if repo.declinedBizID != "BIZ_123" {
			t.Errorf("expected declinedBizID BIZ_123, got %s", repo.declinedBizID)
		}
	})

	t.Run("invalid action returns error", func(t *testing.T) {
		repo := &mockBusinessRepo{
			findBiz: existingBiz,
		}
		service := NewBusinessService(repo, nil)

		err := service.ConfirmScheduleAction("OWNER_1", "BIZ_123", models.ConfirmScheduleInput{
			Action: "invalid_action",
		})
		if err == nil {
			t.Fatal("expected error for invalid action, got nil")
		}
	})
}
