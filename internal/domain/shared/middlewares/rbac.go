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
				"error": "This role is not found, ensure this route have passed RequireAuth method ",
			})
		}

		userRole, ok := roleValue.(string)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Role format is not valid",
			})
		}

		isAllowed := slices.Contains(allowedRoles, userRole)

		if !isAllowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "fail",
				"error":  "Your role (" + userRole + ") is not allowed to access this page",
			})
		}

		return c.Next()
	}
}