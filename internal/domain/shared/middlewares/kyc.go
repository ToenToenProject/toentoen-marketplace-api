package middlewares

import (
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
			message := "Fitur ini memerlukan verifikasi identitas merchant (KYC). Silakan lengkapi berkas KTP Anda di menu Profil."
			if owner.KycStatus == "pending" {
				message = "Dokumen identitas (KYC) Anda sedang dalam proses peninjauan oleh tim kami. Harap menunggu hingga proses verifikasi selesai."
			} else if owner.KycStatus == "rejected" {
				message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan periksa alasan penolakan dan unggah ulang berkas KTP yang valid."
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
			message := "Fitur ini memerlukan verifikasi identitas admin (KYC). Silakan lengkapi berkas KTP Anda."
			if admin.KycStatus == "pending" {
				message = "Dokumen identitas (KYC) Anda sedang dalam proses peninjauan oleh tim kami."
			} else if admin.KycStatus == "rejected" {
				message = "Pengajuan verifikasi identitas (KYC) Anda ditolak. Silakan unggah ulang berkas yang valid."
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
