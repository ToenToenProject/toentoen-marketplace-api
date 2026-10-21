package kychelper

import (
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"
)

func strPtr(s string) *string {
	return &s
}

func TestKYCCopy(t *testing.T) {
	verifiedTime := time.Now().Add(-24 * time.Hour)

	t.Run("CopyKYCOwnerToAdmin copies all KTP fields and sets verified status", func(t *testing.T) {
		owner := &models.Owner{
			AccountID:      "ACC_OWNER",
			KtpNumber:      strPtr("3171012345670001"),
			KtpPhotoUrl:    strPtr("https://s3.amazonaws.com/toentoen/ktp/owner.jpg"),
			SelfiePhotoUrl: strPtr("https://s3.amazonaws.com/toentoen/selfie/owner.jpg"),
			KtpName:        strPtr("Budi Santoso"),
			KtpBirthPlace:  strPtr("Jakarta"),
			KtpBirthDate:   strPtr("1990-01-01"),
			KtpGender:      strPtr("Laki-Laki"),
			KtpAddress:     strPtr("Jl. Sudirman No. 1"),
			KtpRtRw:        strPtr("001/002"),
			KtpVillage:     strPtr("Menteng"),
			KtpDistrict:    strPtr("Menteng"),
			KtpReligion:    strPtr("Islam"),
			KtpOccupation:  strPtr("Wiraswasta"),
			KtpCitizenship: strPtr("WNI"),
			KycStatus:      "verified",
			KycSubmittedAt: &verifiedTime,
			KycVerifiedAt:  &verifiedTime,
		}

		admin := &models.BusinessAdmin{
			AccountID: "ACC_ADMIN",
			KycStatus: "unverified",
		}

		CopyKYCOwnerToAdmin(owner, admin)

		if admin.KycStatus != "verified" {
			t.Fatalf("expected admin KycStatus to be 'verified', got: %s", admin.KycStatus)
		}
		if admin.KtpNumber == nil || *admin.KtpNumber != "3171012345670001" {
			t.Errorf("expected KtpNumber 3171012345670001, got: %v", admin.KtpNumber)
		}
		if admin.KtpName == nil || *admin.KtpName != "Budi Santoso" {
			t.Errorf("expected KtpName 'Budi Santoso', got: %v", admin.KtpName)
		}
		if admin.KtpAddress == nil || *admin.KtpAddress != "Jl. Sudirman No. 1" {
			t.Errorf("expected KtpAddress 'Jl. Sudirman No. 1', got: %v", admin.KtpAddress)
		}
		if admin.KycVerifiedAt == nil || !admin.KycVerifiedAt.Equal(verifiedTime) {
			t.Errorf("expected KycVerifiedAt to match verifiedTime, got: %v", admin.KycVerifiedAt)
		}
	})

	t.Run("CopyKYCAdminToOwner copies all KTP fields and sets verified status", func(t *testing.T) {
		admin := &models.BusinessAdmin{
			AccountID:      "ACC_ADMIN",
			KtpNumber:      strPtr("3271019876540002"),
			KtpPhotoUrl:    strPtr("https://s3.amazonaws.com/toentoen/ktp/admin.jpg"),
			SelfiePhotoUrl: strPtr("https://s3.amazonaws.com/toentoen/selfie/admin.jpg"),
			KtpName:        strPtr("Siti Rahma"),
			KtpBirthPlace:  strPtr("Bandung"),
			KtpBirthDate:   strPtr("1995-05-05"),
			KtpGender:      strPtr("Perempuan"),
			KtpAddress:     strPtr("Jl. Asia Afrika No. 10"),
			KtpRtRw:        strPtr("003/004"),
			KtpVillage:     strPtr("Braga"),
			KtpDistrict:    strPtr("Sumur Bandung"),
			KtpReligion:    strPtr("Islam"),
			KtpOccupation:  strPtr("Karyawan Swasta"),
			KtpCitizenship: strPtr("WNI"),
			KycStatus:      "verified",
			KycSubmittedAt: &verifiedTime,
			KycVerifiedAt:  &verifiedTime,
		}

		owner := &models.Owner{
			AccountID: "ACC_OWNER",
			KycStatus: "unverified",
		}

		CopyKYCAdminToOwner(admin, owner)

		if owner.KycStatus != "verified" {
			t.Fatalf("expected owner KycStatus to be 'verified', got: %s", owner.KycStatus)
		}
		if owner.KtpNumber == nil || *owner.KtpNumber != "3271019876540002" {
			t.Errorf("expected KtpNumber 3271019876540002, got: %v", owner.KtpNumber)
		}
		if owner.KtpName == nil || *owner.KtpName != "Siti Rahma" {
			t.Errorf("expected KtpName 'Siti Rahma', got: %v", owner.KtpName)
		}
		if owner.KtpAddress == nil || *owner.KtpAddress != "Jl. Asia Afrika No. 10" {
			t.Errorf("expected KtpAddress 'Jl. Asia Afrika No. 10', got: %v", owner.KtpAddress)
		}
		if owner.KycVerifiedAt == nil || !owner.KycVerifiedAt.Equal(verifiedTime) {
			t.Errorf("expected KycVerifiedAt to match verifiedTime, got: %v", owner.KycVerifiedAt)
		}
	})

	t.Run("nil src or dst handles safely without panic", func(t *testing.T) {
		owner := &models.Owner{}
		admin := &models.BusinessAdmin{}

		CopyKYCOwnerToAdmin(nil, admin)
		CopyKYCOwnerToAdmin(owner, nil)
		CopyKYCAdminToOwner(nil, owner)
		CopyKYCAdminToOwner(admin, nil)

		InheritVerifiedKYCOnCreate(nil, nil, "owner", owner, nil)
	})
}
