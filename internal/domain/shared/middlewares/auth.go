package middlewares

import (
	"fmt"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(c *fiber.Ctx) error {
	var tokenString string

	tokenString = c.Cookies("auth_token")

	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token pengguna tidak ditemukan, silakan login kembali"})
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Metode enkripsi tidak valid")
		}
		return []byte(config.Envs.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesi tidak valid atau telah kedaluwarsa"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Gagal membaca data sesi pengguna"})
	}

	accountID, ok := claims["account_id"].(string)
	if ok {
		var account models.Account
		if err := database.DB.Select("last_login_at").Where("account_id = ?", accountID).First(&account).Error; err == nil {
			if account.LastLoginAt != nil {
				if iatFloat, ok := claims["iat"].(float64); ok {
					if int64(iatFloat) < account.LastLoginAt.Unix() {
						return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesi Anda sudah tidak valid, silakan login kembali"})
					}
				}
			}
		}
	}

	c.Locals("account_id", claims["account_id"])
	c.Locals("role", claims["role"])

	return c.Next()
}

func OptionalAuth(c *fiber.Ctx) error {
	tokenString := c.Cookies("auth_token")

	if tokenString == "" {
		return c.Next()
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Metode enkripsi tidak valid")
		}
		return []byte(config.Envs.JWTSecret), nil
	})

	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			accountID, _ := claims["account_id"].(string)
			isValid := true

			var account models.Account
			if err := database.DB.Select("last_login_at").Where("account_id = ?", accountID).First(&account).Error; err == nil {
				if account.LastLoginAt != nil {
					if iatFloat, ok := claims["iat"].(float64); ok {
						if int64(iatFloat) < account.LastLoginAt.Unix() {
							isValid = false
						}
					}
				}
			}

			if isValid {
				c.Locals("account_id", claims["account_id"])
				c.Locals("role", claims["role"])
			}
		}
	}

	return c.Next()
}
