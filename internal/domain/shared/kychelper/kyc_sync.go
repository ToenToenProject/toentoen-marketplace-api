package kychelper

import (
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// InheritVerifiedKYCOnCreate memeriksa apakah akun dengan identitas yang sama (email / nomor telepon)
// sudah memiliki profil terverifikasi KYC di peran merchant lainnya (Owner <-> Business Admin).
// Jika ditemukan, profil baru akan langsung mewarisi data KTP dan berstatus 'verified'.
func InheritVerifiedKYCOnCreate(db *gorm.DB, account *models.Account, role string, ownerRecord *models.Owner, adminRecord *models.BusinessAdmin) {
	if account == nil || db == nil {
		return
	}

	cleanEmail := ""
	if account.Email != nil {
		cleanEmail = strings.ToLower(strings.TrimSpace(*account.Email))
	}
	cleanPhone := ""
	if account.PhoneNumber != nil {
		cleanPhone = strings.TrimSpace(*account.PhoneNumber)
	}

	if cleanEmail == "" && cleanPhone == "" {
		return
	}

	switch role {
	case "owner":
		if ownerRecord == nil {
			return
		}
		// Cari akun business_admin yang sudah terverifikasi KYC
		var siblingAdmin models.BusinessAdmin
		q := db.Model(&models.BusinessAdmin{}).
			Joins("JOIN accounts ON accounts.account_id = business_admins.account_id").
			Where("business_admins.kyc_status = ?", "verified")

		if cleanEmail != "" && cleanPhone != "" {
			q = q.Where("LOWER(accounts.email) = ? OR accounts.phone_number = ?", cleanEmail, cleanPhone)
		} else if cleanEmail != "" {
			q = q.Where("LOWER(accounts.email) = ?", cleanEmail)
		} else {
			q = q.Where("accounts.phone_number = ?", cleanPhone)
		}

		if err := q.First(&siblingAdmin).Error; err == nil {
			CopyKYCAdminToOwner(&siblingAdmin, ownerRecord)
		}

	case "business_admin":
		if adminRecord == nil {
			return
		}
		// Cari akun owner yang sudah terverifikasi KYC
		var siblingOwner models.Owner
		q := db.Model(&models.Owner{}).
			Joins("JOIN accounts ON accounts.account_id = owners.account_id").
			Where("owners.kyc_status = ?", "verified")

		if cleanEmail != "" && cleanPhone != "" {
			q = q.Where("LOWER(accounts.email) = ? OR accounts.phone_number = ?", cleanEmail, cleanPhone)
		} else if cleanEmail != "" {
			q = q.Where("LOWER(accounts.email) = ?", cleanEmail)
		} else {
			q = q.Where("accounts.phone_number = ?", cleanPhone)
		}

		if err := q.First(&siblingOwner).Error; err == nil {
			CopyKYCOwnerToAdmin(&siblingOwner, adminRecord)
		}
	}
}

// CopyKYCAdminToOwner menyalin seluruh data KTP dan status verifikasi dari BusinessAdmin ke Owner.
func CopyKYCAdminToOwner(src *models.BusinessAdmin, dst *models.Owner) {
	if src == nil || dst == nil {
		return
	}
	dst.KtpNumber = src.KtpNumber
	dst.KtpPhotoUrl = src.KtpPhotoUrl
	dst.SelfiePhotoUrl = src.SelfiePhotoUrl
	dst.KtpName = src.KtpName
	dst.KtpBirthPlace = src.KtpBirthPlace
	dst.KtpBirthDate = src.KtpBirthDate
	dst.KtpGender = src.KtpGender
	dst.KtpAddress = src.KtpAddress
	dst.KtpRtRw = src.KtpRtRw
	dst.KtpVillage = src.KtpVillage
	dst.KtpDistrict = src.KtpDistrict
	dst.KtpReligion = src.KtpReligion
	dst.KtpOccupation = src.KtpOccupation
	dst.KtpCitizenship = src.KtpCitizenship
	dst.KycStatus = "verified"
	dst.KycSubmittedAt = src.KycSubmittedAt
	if src.KycVerifiedAt != nil {
		dst.KycVerifiedAt = src.KycVerifiedAt
	} else {
		now := time.Now()
		dst.KycVerifiedAt = &now
	}
	dst.KycRejectionReason = nil
}

// CopyKYCOwnerToAdmin menyalin seluruh data KTP dan status verifikasi dari Owner ke BusinessAdmin.
func CopyKYCOwnerToAdmin(src *models.Owner, dst *models.BusinessAdmin) {
	if src == nil || dst == nil {
		return
	}
	dst.KtpNumber = src.KtpNumber
	dst.KtpPhotoUrl = src.KtpPhotoUrl
	dst.SelfiePhotoUrl = src.SelfiePhotoUrl
	dst.KtpName = src.KtpName
	dst.KtpBirthPlace = src.KtpBirthPlace
	dst.KtpBirthDate = src.KtpBirthDate
	dst.KtpGender = src.KtpGender
	dst.KtpAddress = src.KtpAddress
	dst.KtpRtRw = src.KtpRtRw
	dst.KtpVillage = src.KtpVillage
	dst.KtpDistrict = src.KtpDistrict
	dst.KtpReligion = src.KtpReligion
	dst.KtpOccupation = src.KtpOccupation
	dst.KtpCitizenship = src.KtpCitizenship
	dst.KycStatus = "verified"
	dst.KycSubmittedAt = src.KycSubmittedAt
	if src.KycVerifiedAt != nil {
		dst.KycVerifiedAt = src.KycVerifiedAt
	} else {
		now := time.Now()
		dst.KycVerifiedAt = &now
	}
	dst.KycRejectionReason = nil
}

// SyncVerifiedKYCIfSiblingVerified memeriksa apakah akun dengan ID tertentu belum terverifikasi KYC,
// namun memiliki akun saudara (role owner <-> business_admin dengan email/telepon yang sama)
// yang sudah terverifikasi. Jika ditemukan, fungsi ini akan otomatis menyinkronkan data dan statusnya
// ke database, serta mengembalikan nilai true.
func SyncVerifiedKYCIfSiblingVerified(db *gorm.DB, accountID string, role string) bool {
	if db == nil || accountID == "" {
		return false
	}

	var currentAcc models.Account
	if err := db.Select("account_id, email, phone_number, role").Where("account_id = ?", accountID).First(&currentAcc).Error; err != nil {
		return false
	}

	cleanEmail := ""
	if currentAcc.Email != nil {
		cleanEmail = strings.ToLower(strings.TrimSpace(*currentAcc.Email))
	}
	cleanPhone := ""
	if currentAcc.PhoneNumber != nil {
		cleanPhone = strings.TrimSpace(*currentAcc.PhoneNumber)
	}

	if cleanEmail == "" && cleanPhone == "" {
		return false
	}

	switch role {
	case "owner":
		var currentOwner models.Owner
		if err := db.Where("account_id = ?", accountID).First(&currentOwner).Error; err == nil {
			if currentOwner.KycStatus == "verified" {
				return true
			}
		}

		// Cari akun business_admin yang sudah verified milik user ini
		var siblingAdmin models.BusinessAdmin
		q := db.Model(&models.BusinessAdmin{}).
			Joins("JOIN accounts ON accounts.account_id = business_admins.account_id").
			Where("accounts.account_id != ? AND business_admins.kyc_status = ?", accountID, "verified")

		if cleanEmail != "" && cleanPhone != "" {
			q = q.Where("LOWER(accounts.email) = ? OR accounts.phone_number = ?", cleanEmail, cleanPhone)
		} else if cleanEmail != "" {
			q = q.Where("LOWER(accounts.email) = ?", cleanEmail)
		} else {
			q = q.Where("accounts.phone_number = ?", cleanPhone)
		}

		if err := q.First(&siblingAdmin).Error; err == nil {
			var updatedOwner models.Owner
			updatedOwner.AccountID = accountID
			CopyKYCAdminToOwner(&siblingAdmin, &updatedOwner)

			_ = db.Model(&models.Owner{}).Where("account_id = ?", accountID).Updates(map[string]interface{}{
				"ktp_number":           updatedOwner.KtpNumber,
				"ktp_photo_url":        updatedOwner.KtpPhotoUrl,
				"selfie_photo_url":     updatedOwner.SelfiePhotoUrl,
				"ktp_name":             updatedOwner.KtpName,
				"ktp_birth_place":      updatedOwner.KtpBirthPlace,
				"ktp_birth_date":       updatedOwner.KtpBirthDate,
				"ktp_gender":           updatedOwner.KtpGender,
				"ktp_address":          updatedOwner.KtpAddress,
				"ktp_rt_rw":            updatedOwner.KtpRtRw,
				"ktp_village":          updatedOwner.KtpVillage,
				"ktp_district":         updatedOwner.KtpDistrict,
				"ktp_religion":         updatedOwner.KtpReligion,
				"ktp_occupation":       updatedOwner.KtpOccupation,
				"ktp_citizenship":      updatedOwner.KtpCitizenship,
				"kyc_status":           "verified",
				"kyc_submitted_at":     updatedOwner.KycSubmittedAt,
				"kyc_verified_at":      updatedOwner.KycVerifiedAt,
				"kyc_rejection_reason": nil,
			}).Error
			return true
		}

	case "business_admin":
		var currentAdmin models.BusinessAdmin
		if err := db.Where("account_id = ?", accountID).First(&currentAdmin).Error; err == nil {
			if currentAdmin.KycStatus == "verified" {
				return true
			}
		}

		// Cari akun owner yang sudah verified milik user ini
		var siblingOwner models.Owner
		q := db.Model(&models.Owner{}).
			Joins("JOIN accounts ON accounts.account_id = owners.account_id").
			Where("accounts.account_id != ? AND owners.kyc_status = ?", accountID, "verified")

		if cleanEmail != "" && cleanPhone != "" {
			q = q.Where("LOWER(accounts.email) = ? OR accounts.phone_number = ?", cleanEmail, cleanPhone)
		} else if cleanEmail != "" {
			q = q.Where("LOWER(accounts.email) = ?", cleanEmail)
		} else {
			q = q.Where("accounts.phone_number = ?", cleanPhone)
		}

		if err := q.First(&siblingOwner).Error; err == nil {
			var updatedAdmin models.BusinessAdmin
			updatedAdmin.AccountID = accountID
			CopyKYCOwnerToAdmin(&siblingOwner, &updatedAdmin)

			_ = db.Model(&models.BusinessAdmin{}).Where("account_id = ?", accountID).Updates(map[string]interface{}{
				"ktp_number":           updatedAdmin.KtpNumber,
				"ktp_photo_url":        updatedAdmin.KtpPhotoUrl,
				"selfie_photo_url":     updatedAdmin.SelfiePhotoUrl,
				"ktp_name":             updatedAdmin.KtpName,
				"ktp_birth_place":      updatedAdmin.KtpBirthPlace,
				"ktp_birth_date":       updatedAdmin.KtpBirthDate,
				"ktp_gender":           updatedAdmin.KtpGender,
				"ktp_address":          updatedAdmin.KtpAddress,
				"ktp_rt_rw":            updatedAdmin.KtpRtRw,
				"ktp_village":          updatedAdmin.KtpVillage,
				"ktp_district":         updatedAdmin.KtpDistrict,
				"ktp_religion":         updatedAdmin.KtpReligion,
				"ktp_occupation":       updatedAdmin.KtpOccupation,
				"ktp_citizenship":      updatedAdmin.KtpCitizenship,
				"kyc_status":           "verified",
				"kyc_submitted_at":     updatedAdmin.KycSubmittedAt,
				"kyc_verified_at":      updatedAdmin.KycVerifiedAt,
				"kyc_rejection_reason": nil,
			}).Error
			return true
		}
	}

	return false
}

// FindSiblingAccountIDs mengembalikan daftar account_id lain yang menggunakan email
// atau nomor telepon yang sama dengan accountID yang diberikan.
func FindSiblingAccountIDs(db *gorm.DB, accountID string) []string {
	if db == nil || accountID == "" {
		return nil
	}

	var acc models.Account
	if err := db.Select("email, phone_number").Where("account_id = ?", accountID).First(&acc).Error; err != nil {
		return nil
	}

	cleanEmail := ""
	if acc.Email != nil {
		cleanEmail = strings.ToLower(strings.TrimSpace(*acc.Email))
	}
	cleanPhone := ""
	if acc.PhoneNumber != nil {
		cleanPhone = strings.TrimSpace(*acc.PhoneNumber)
	}

	if cleanEmail == "" && cleanPhone == "" {
		return nil
	}

	var siblingIDs []string
	q := db.Model(&models.Account{}).Where("account_id != ?", accountID)
	if cleanEmail != "" && cleanPhone != "" {
		q = q.Where("LOWER(email) = ? OR phone_number = ?", cleanEmail, cleanPhone)
	} else if cleanEmail != "" {
		q = q.Where("LOWER(email) = ?", cleanEmail)
	} else {
		q = q.Where("phone_number = ?", cleanPhone)
	}

	_ = q.Pluck("account_id", &siblingIDs).Error
	return siblingIDs
}
