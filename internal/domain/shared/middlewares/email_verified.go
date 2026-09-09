package middlewares

import (
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"github.com/gofiber/fiber/v2"
)

// RequireEmailVerified memastikan bahwa pengguna (customer, owner, business_admin)
// telah melakukan verifikasi email (email_verified_at IS NOT NULL) sebelum dapat
// mengakses endpoint berproteksi.
func RequireEmailVerified(c *fiber.Ctx) error {
	accountIDVal := c.Locals("account_id")
	if accountIDVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"responseCode": "401",
			"error":        "Autentikasi diperlukan sebelum mengakses fitur ini",
		})
	}

	accountID, ok := accountIDVal.(string)
	if !ok || accountID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"responseCode": "401",
			"error":        "Sesi autentikasi tidak valid",
		})
	}

	var account models.Account
	if err := database.DB.Select("email_verified_at").Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"error":        "Akun tidak ditemukan",
		})
	}

	if account.EmailVerifiedAt == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"responseCode": "403",
			"error":        "akun belum terverifikasi, silahkan verifikasi terlebih dahulu",
		})
	}

	return c.Next()
}
