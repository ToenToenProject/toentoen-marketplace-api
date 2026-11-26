package middlewares

import (
	"slices"

	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func AllowRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleValue := c.Locals("role")
		if roleValue == nil {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Role pengguna tidak ditemukan, pastikan rute ini melewati proses otentikasi", nil)
		}

		userRole, ok := roleValue.(string)
		if !ok {
			return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Format role tidak valid", nil)
		}

		isAllowed := slices.Contains(allowedRoles, userRole)

		if !isAllowed {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Role Anda ("+userRole+") tidak memiliki izin untuk mengakses halaman ini", nil)
		}

		return c.Next()
	}
}