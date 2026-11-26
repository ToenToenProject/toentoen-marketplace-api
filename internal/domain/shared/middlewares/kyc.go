package middlewares

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/kychelper"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// RequireKYC verifies that the merchant account has approved KYC status before accessing commercial features.
func RequireKYC(c *fiber.Ctx) error {
	roleVal := c.Locals("role")
	accountIDVal := c.Locals("account_id")

	if roleVal == nil || accountIDVal == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Autentikasi diperlukan sebelum mengakses fitur ini", nil)
	}

	role, _ := roleVal.(string)
	accountID, _ := accountIDVal.(string)

	switch role {
	case "owner":
		var owner models.Owner
		if err := database.DB.Select("kyc_status").Where("account_id = ?", accountID).First(&owner).Error; err != nil {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Data merchant tidak ditemukan. Silakan lengkapi profil akun Anda.", nil)
		}
		if owner.KycStatus != "verified" {
			// Check if sibling admin account is verified.
			if kychelper.SyncVerifiedKYCIfSiblingVerified(database.DB, accountID, "owner") {
				return c.Next()
			}
			message := "Fitur ini memerlukan verifikasi identitas merchant (KYC). Silakan lengkapi berkas identitas Anda di menu Profil."
			if owner.KycStatus == "pending" {
				message = "akun sedang dalam verifikasi, mohon tunggu"
			} else if owner.KycStatus == "rejected" {
				message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan periksa alasan penolakan dan ajukan ulang berkas yang valid."
			}
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, message, fiber.Map{
				"kycStatus": owner.KycStatus,
			})
		}

	case "business_admin":
		var admin models.BusinessAdmin
		if err := database.DB.Select("kyc_status").Where("account_id = ?", accountID).First(&admin).Error; err != nil {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Data admin bisnis tidak ditemukan. Silakan lengkapi profil akun Anda.", nil)
		}
		if admin.KycStatus != "verified" {
			// Check if sibling owner account is verified.
			if kychelper.SyncVerifiedKYCIfSiblingVerified(database.DB, accountID, "business_admin") {
				return c.Next()
			}
			message := "Fitur ini memerlukan verifikasi identitas admin (KYC). Silakan lengkapi berkas identitas Anda."
			if admin.KycStatus == "pending" {
				message = "akun sedang dalam verifikasi, mohon tunggu"
			} else if admin.KycStatus == "rejected" {
				message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan ajukan ulang berkas yang valid."
			}
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, message, fiber.Map{
				"kycStatus": admin.KycStatus,
			})
		}
	}

	return c.Next()
}

// RequireOwnerKYCSubmitted ensures an owner has verified email and at least submitted KYC before creating stores.
func RequireOwnerKYCSubmitted(c *fiber.Ctx) error {
	roleVal := c.Locals("role")
	accountIDVal := c.Locals("account_id")

	if roleVal == nil || accountIDVal == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Autentikasi diperlukan sebelum mengakses fitur ini", nil)
	}

	role, _ := roleVal.(string)
	accountID, _ := accountIDVal.(string)

	if role != "owner" {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Hanya akun pemilik (owner) yang dapat membuat bisnis", nil)
	}

	// 1. Check email verification.
	if verified, hasCached := c.Locals("email_verified").(bool); hasCached {
		if !verified {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Email akun Anda belum terverifikasi. Silakan verifikasi email Anda terlebih dahulu sebelum membuat bisnis.", nil)
		}
	} else {
		var account models.Account
		if err := database.DB.Select("email_verified_at").Where("account_id = ?", accountID).First(&account).Error; err != nil {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Akun tidak ditemukan", nil)
		}
		if account.EmailVerifiedAt == nil {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Email akun Anda belum terverifikasi. Silakan verifikasi email Anda terlebih dahulu sebelum membuat bisnis.", nil)
		}
	}

	// 2. Check submitted KYC status (pending or verified).
	var owner models.Owner
	if err := database.DB.Select("kyc_status").Where("account_id = ?", accountID).First(&owner).Error; err != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Data merchant tidak ditemukan. Silakan lengkapi profil akun Anda.", nil)
	}

	if owner.KycStatus != "pending" && owner.KycStatus != "verified" {
		// Check if sibling admin account is verified.
		if kychelper.SyncVerifiedKYCIfSiblingVerified(database.DB, accountID, "owner") {
			return c.Next()
		}
		message := "Anda belum dapat membuat bisnis karena pengajuan verifikasi identitas (KYC) Anda belum dikirimkan. Silakan lengkapi dan ajukan KYC terlebih dahulu."
		if owner.KycStatus == "rejected" {
			message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan periksa alasan penolakan dan ajukan ulang berkas KYC Anda sebelum membuat bisnis."
		}
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, message, fiber.Map{
			"kycStatus": owner.KycStatus,
		})
	}

	return c.Next()
}
