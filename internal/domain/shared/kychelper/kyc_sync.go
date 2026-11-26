package kychelper

import (
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// InheritVerifiedKYCOnCreate checks whether an account with the same email or phone
// already holds a verified KYC profile in a sibling merchant role (Owner <-> Business Admin).
// If found, the new profile inherits verified KYC status and identity records immediately.
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
		// Find verified sibling business_admin account.
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
		// Find verified sibling owner account.
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

// CopyKYCAdminToOwner copies KTP identity details and verification status from BusinessAdmin to Owner.
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

// CopyKYCOwnerToAdmin copies KTP identity details and verification status from Owner to BusinessAdmin.
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

// SyncVerifiedKYCIfSiblingVerified checks if an unverified merchant account has a verified sibling account
// sharing the same email or phone number. If verified, it synchronizes the profile and returns true.
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

		// Find verified sibling business_admin account.
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

		// Find verified sibling owner account.
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

// FindSiblingAccountIDs returns account IDs sharing the same email or phone number across roles.
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

// GetEffectiveOwnerKYCStatus resolves the effective KYC status for an owner, inspecting verified sibling accounts if needed.
func GetEffectiveOwnerKYCStatus(db *gorm.DB, ownerID string) string {
	if db == nil || strings.TrimSpace(ownerID) == "" {
		return "unverified"
	}
	var owner models.Owner
	if err := db.Select("kyc_status").Where("account_id = ?", ownerID).First(&owner).Error; err != nil {
		return "unverified"
	}
	status := strings.ToLower(strings.TrimSpace(owner.KycStatus))
	if status == "" {
		status = "unverified"
	}
	if status != "verified" {
		if SyncVerifiedKYCIfSiblingVerified(db, ownerID, "owner") {
			return "verified"
		}
	}
	return status
}

// IsOwnerKYCVerified checks whether the business owner has approved KYC verification.
func IsOwnerKYCVerified(db *gorm.DB, ownerID string) bool {
	return GetEffectiveOwnerKYCStatus(db, ownerID) == "verified"
}

// BuildBusinessVisibility builds the store visibility summary for merchant dashboards based on KYC verification.
func BuildBusinessVisibility(kycStatus string, isActive bool, bizStatus string) *models.BusinessVisibilityInfo {
	status := strings.ToLower(strings.TrimSpace(kycStatus))
	if status == "" {
		status = "unverified"
	}
	if bizStatus == "deleted" || !isActive {
		return &models.BusinessVisibilityInfo{
			IsPublic:  false,
			KycStatus: status,
			Reason:    "Toko sedang tidak aktif sehingga tidak tampil ke publik.",
		}
	}
	if status == "verified" {
		return &models.BusinessVisibilityInfo{
			IsPublic:  true,
			KycStatus: status,
			Reason:    "",
		}
	}

	reason := "Toko belum tampil ke publik karena verifikasi identitas (KYC) belum diajukan. Anda tetap dapat melengkapi profil dan produk toko."
	switch status {
	case "pending":
		reason = "Toko belum tampil ke publik karena verifikasi identitas (KYC) sedang dalam proses peninjauan. Anda tetap dapat mengelola toko dan produk."
	case "rejected":
		reason = "Toko belum tampil ke publik karena pengajuan KYC ditolak. Silakan ajukan ulang berkas KYC Anda."
	}
	return &models.BusinessVisibilityInfo{
		IsPublic:  false,
		KycStatus: status,
		Reason:    reason,
	}
}
