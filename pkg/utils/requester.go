package utils

import "github.com/gofiber/fiber/v2"

// GetRequesterInfo extracts account ID and role credentials from Fiber request context.
func GetRequesterInfo(c *fiber.Ctx) (string, string, error) {
	accountIDValue := c.Locals("account_id")
	roleValue := c.Locals("role")

	if accountIDValue == nil || roleValue == nil {
		return "", "", fiber.ErrUnauthorized
	}

	requesterID, ok1 := accountIDValue.(string)
	requesterRole, ok2 := roleValue.(string)
	if !ok1 || !ok2 {
		return "", "", fiber.ErrInternalServerError
	}

	return requesterID, requesterRole, nil
}
