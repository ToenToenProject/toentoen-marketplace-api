package middlewares

import (
	"fmt"

	"toentoen-backend/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(c *fiber.Ctx) error {
	var tokenString string

	tokenString = c.Cookies("auth_token")

	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User token is not found, please retry login!"})
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Encription methode is not valid")
		}
		return []byte(config.Envs.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session is not valid or expired"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Failed to read session data"})
	}

	c.Locals("account_id", claims["account_id"])
	c.Locals("role", claims["role"])

	return c.Next()
}
