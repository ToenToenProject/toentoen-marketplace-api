package middlewares

import (
	"slices"

	"github.com/gofiber/fiber/v2"
)

func AllowRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleValue := c.Locals("role")
		if roleValue == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Role pengguna tidak ditemukan, pastikan rute ini melewati proses otentikasi",
			})
		}

		userRole, ok := roleValue.(string)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Format role tidak valid",
			})
		}

		isAllowed := slices.Contains(allowedRoles, userRole)

		if !isAllowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "fail",
				"error":  "Role Anda (" + userRole + ") tidak memiliki izin untuk mengakses halaman ini",
			})
		}

		return c.Next()
	}
}