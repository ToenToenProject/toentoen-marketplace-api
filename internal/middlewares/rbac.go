package middlewares

import (
	"github.com/gofiber/fiber/v2"
)

// AllowRoles middleware to restrict access based on user roles
func AllowRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fetch role
		roleValue := c.Locals("role")
		if roleValue == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Role tidak ditemukan. Pastikan rute ini melewati RequireAuth lebih dulu.",
			})
		}

		// Convert to string
		userRole, ok := roleValue.(string)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Format role tidak valid",
			})
		}

		// Check whether the user role is allowed or not
		isAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		// If not allowes, return fail
		if !isAllowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "fail",
				"error":  "Access Forbidden: Role Anda (" + userRole + ") tidak diizinkan untuk mengakses halaman ini",
			})
		}

		// If allowed, continue to handler
		return c.Next()
	}
}