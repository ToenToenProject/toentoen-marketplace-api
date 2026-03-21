package middlewares

import (
	"fmt"

	"toentoen-backend/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(c *fiber.Ctx) error {
	var tokenString string

	// Fetch cookies
	tokenString = c.Cookies("auth_token")

	// If empty, return error
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token not found. Silakan login kembali."})
	}

	// JWT Validation & Unpacking
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode enkripsi tidak valid")
		}
		return []byte(config.Envs.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesi tidak valid atau telah kedaluwarsa"})
	}

	// Fetch data from token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Gagal membaca data sesi"})
	}

	// Store account_id to Locals (Fiber)
	if id, ok := claims["account_id"].(float64); ok {
		c.Locals("account_id", int64(id))
	} else {
		c.Locals("account_id", claims["account_id"])
	}
	
	c.Locals("role", claims["role"])

	// Continue to handler
	return c.Next()
}