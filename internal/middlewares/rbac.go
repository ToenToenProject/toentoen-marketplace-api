package middlewares

import (
	"github.com/gofiber/fiber/v2"
)

func AllowRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleValue := c.Locals("role")
		if roleValue == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Role tidak ditemukan. Pastikan rute ini melewati RequireAuth lebih dulu.",
			})
		}

		userRole, ok := roleValue.(string)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Format role tidak valid",
			})
		}

		isAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "fail",
				"error":  "Access Forbidden: Role Anda (" + userRole + ") tidak diizinkan untuk mengakses halaman ini",
			})
		}

		return c.Next()
	}
}