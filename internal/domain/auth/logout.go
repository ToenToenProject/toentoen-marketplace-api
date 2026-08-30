package auth

import (
	"time"
	"toentoen-backend/internal/config"

	"github.com/gofiber/fiber/v2"
)

var authCookies = []string{
	"auth_token",
	"admin_token",
	"owner_token",
	"biz_admin_token",
	"user_token",
}

func Logout(c *fiber.Ctx) error {
	isProduction := config.Envs != nil && config.Envs.AppEnv == "production"
	for _, cookieName := range authCookies {
		c.Cookie(&fiber.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-1 * time.Hour),
			HTTPOnly: true,
			Secure:   isProduction,
			SameSite: "Lax",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Berhasil logout. Semua sesi telah dibersihkan.",
	})
}