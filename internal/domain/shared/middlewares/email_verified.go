package middlewares

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// RequireEmailVerified ensures the authenticated user has verified their email address before accessing protected routes.
func RequireEmailVerified(c *fiber.Ctx) error {
	accountIDVal := c.Locals("account_id")
	if accountIDVal == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Autentikasi diperlukan sebelum mengakses fitur ini", nil)
	}

	accountID, ok := accountIDVal.(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi autentikasi tidak valid", nil)
	}

	if verified, hasCached := c.Locals("email_verified").(bool); hasCached {
		if !verified {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "akun belum terverifikasi, silahkan verifikasi terlebih dahulu", nil)
		}
		return c.Next()
	}

	var account models.Account
	if err := database.DB.Select("email_verified_at").Where("account_id = ?", accountID).First(&account).Error; err != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Akun tidak ditemukan", nil)
	}

	if account.EmailVerifiedAt == nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "akun belum terverifikasi, silahkan verifikasi terlebih dahulu", nil)
	}

	return c.Next()
}
