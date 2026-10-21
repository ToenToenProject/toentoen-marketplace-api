package middlewares

import (
	"toentoen-backend/internal/domain/shared/kychelper"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"github.com/gofiber/fiber/v2"
)

// RequireKYC memastikan bahwa merchant (owner atau business_admin) telah lolos verifikasi KYC (status 'verified')
// sebelum dapat mengakses fitur-fitur operasional komersial.
func RequireKYC(c *fiber.Ctx) error {
	roleVal := c.Locals("role")
	accountIDVal := c.Locals("account_id")

	if roleVal == nil || accountIDVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"responseCode": "401",
			"error":        "Autentikasi diperlukan sebelum mengakses fitur ini",
		})
	}

	role, _ := roleVal.(string)
	accountID, _ := accountIDVal.(string)

	switch role {
	case "owner":
		var owner models.Owner
		if err := database.DB.Select("kyc_status").Where("account_id = ?", accountID).First(&owner).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"responseCode": "403",
				"error":        "Data merchant tidak ditemukan. Silakan lengkapi profil akun Anda.",
			})
		}
		if owner.KycStatus != "verified" {
			// Periksa apakah akun admin saudara sudah verified
			if kychelper.SyncVerifiedKYCIfSiblingVerified(database.DB, accountID, "owner") {
				return c.Next()
			}
			message := "Fitur ini memerlukan verifikasi identitas merchant (KYC). Silakan lengkapi berkas identitas Anda di menu Profil."
			if owner.KycStatus == "pending" {
				message = "akun sedang dalam verifikasi, mohon tunggu"
			} else if owner.KycStatus == "rejected" {
				message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan periksa alasan penolakan dan ajukan ulang berkas yang valid."
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"responseCode": "403",
				"kycStatus":    owner.KycStatus,
				"error":        message,
			})
		}

	case "business_admin":
		var admin models.BusinessAdmin
		if err := database.DB.Select("kyc_status").Where("account_id = ?", accountID).First(&admin).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"responseCode": "403",
				"error":        "Data admin bisnis tidak ditemukan. Silakan lengkapi profil akun Anda.",
			})
		}
		if admin.KycStatus != "verified" {
			// Periksa apakah akun owner saudara sudah verified
			if kychelper.SyncVerifiedKYCIfSiblingVerified(database.DB, accountID, "business_admin") {
				return c.Next()
			}
			message := "Fitur ini memerlukan verifikasi identitas admin (KYC). Silakan lengkapi berkas identitas Anda."
			if admin.KycStatus == "pending" {
				message = "akun sedang dalam verifikasi, mohon tunggu"
			} else if admin.KycStatus == "rejected" {
				message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan ajukan ulang berkas yang valid."
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"responseCode": "403",
				"kycStatus":    admin.KycStatus,
				"error":        message,
			})
		}
	}

	return c.Next()
}

// RequireOwnerKYCSubmitted memastikan akun owner telah memverifikasi email dan status KYC-nya minimal 'pending' atau 'verified'
// sebelum dapat membuat toko/bisnis baru.
func RequireOwnerKYCSubmitted(c *fiber.Ctx) error {
	roleVal := c.Locals("role")
	accountIDVal := c.Locals("account_id")

	if roleVal == nil || accountIDVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"responseCode": "401",
			"error":        "Autentikasi diperlukan sebelum mengakses fitur ini",
		})
	}

	role, _ := roleVal.(string)
	accountID, _ := accountIDVal.(string)

	if role != "owner" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"error":        "Hanya akun pemilik (owner) yang dapat membuat bisnis",
			"message":      "Hanya akun pemilik (owner) yang dapat membuat bisnis",
		})
	}

	// 1. Cek Email Verified
	var account models.Account
	if err := database.DB.Select("email_verified_at").Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"error":        "Akun tidak ditemukan",
			"message":      "Akun tidak ditemukan",
		})
	}
	if account.EmailVerifiedAt == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"error":        "Email akun Anda belum terverifikasi. Silakan verifikasi email Anda terlebih dahulu sebelum membuat bisnis.",
			"message":      "Email akun Anda belum terverifikasi. Silakan verifikasi email Anda terlebih dahulu sebelum membuat bisnis.",
		})
	}

	// 2. Cek status KYC (harus sudah diajukan: pending atau verified)
	var owner models.Owner
	if err := database.DB.Select("kyc_status").Where("account_id = ?", accountID).First(&owner).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"error":        "Data merchant tidak ditemukan. Silakan lengkapi profil akun Anda.",
			"message":      "Data merchant tidak ditemukan. Silakan lengkapi profil akun Anda.",
		})
	}

	if owner.KycStatus != "pending" && owner.KycStatus != "verified" {
		// Periksa apakah akun admin saudara sudah verified
		if kychelper.SyncVerifiedKYCIfSiblingVerified(database.DB, accountID, "owner") {
			return c.Next()
		}
		message := "Anda belum dapat membuat bisnis karena pengajuan verifikasi identitas (KYC) Anda belum dikirimkan. Silakan lengkapi dan ajukan KYC terlebih dahulu."
		if owner.KycStatus == "rejected" {
			message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan periksa alasan penolakan dan ajukan ulang berkas KYC Anda sebelum membuat bisnis."
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"kycStatus":    owner.KycStatus,
			"error":        message,
			"message":      message,
		})
	}

	return c.Next()
}
