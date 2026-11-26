package middlewares

import (
	"crypto/subtle"
	"fmt"
	"log"
	"strings"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// extractToken extracts and sanitizes Bearer tokens from the Authorization header or auth_token cookie.
func extractToken(c *fiber.Ctx) string {
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	if authHeader != "" {
		for strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			authHeader = strings.TrimSpace(authHeader[7:])
		}
		authHeader = strings.Trim(authHeader, "\"'`")
		if authHeader != "" && authHeader != "undefined" && authHeader != "null" {
			return authHeader
		}
	}

	cookieToken := strings.TrimSpace(c.Cookies("auth_token"))
	cookieToken = strings.Trim(cookieToken, "\"'`")
	if cookieToken != "" && cookieToken != "undefined" && cookieToken != "null" {
		return cookieToken
	}

	return ""
}

func RequireAuth(c *fiber.Ctx) error {
	tokenString := extractToken(c)
	if tokenString == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Token pengguna tidak ditemukan, silakan login kembali", nil)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Metode enkripsi tidak valid")
		}
		return []byte(config.Envs.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		log.Printf("[AUTH-ERROR] Validasi JWT token gagal (IP: %s): %v", c.IP(), err)
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau telah kedaluwarsa", nil)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Gagal membaca data sesi pengguna", nil)
	}

	accountID, ok := claims["account_id"].(string)
	if ok {
		if config.RDB != nil {
			if revoked, _ := config.RDB.Exists(config.Ctx, "revoked_account:"+accountID).Result(); revoked > 0 {
				return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi Anda telah dicabut atau akun sedang diblokir", nil)
			}
		}

		var account models.Account
		if err := database.DB.Select("last_login_at, email_verified_at, account_status").Where("account_id = ?", accountID).First(&account).Error; err != nil {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akun tidak ditemukan atau telah dihapus, silakan login kembali", nil)
		}
		if account.AccountStatus == "banned" || account.AccountStatus == "suspended" {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Akun Anda sedang ditangguhkan atau diblokir", nil)
		}
		if account.LastLoginAt != nil {
			if iatFloat, ok := claims["iat"].(float64); ok {
				if int64(iatFloat) < account.LastLoginAt.Unix() {
					return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi Anda sudah tidak valid, silakan login kembali", nil)
				}
			}
		}
		c.Locals("email_verified", account.EmailVerifiedAt != nil)
	}

	c.Locals("account_id", claims["account_id"])
	c.Locals("role", claims["role"])

	return c.Next()
}

// RequireInternalSecret validates X-Internal-Secret for service-to-service endpoints across all environments.
func RequireInternalSecret(c *fiber.Ctx) error {
	expected := ""
	if config.Envs != nil {
		expected = config.Envs.InternalSecret
		if expected == "" {
			expected = config.Envs.JWTSecret
		}
	}
	secretHeader := c.Get("X-Internal-Secret")
	if expected == "" || len(secretHeader) != len(expected) || subtle.ConstantTimeCompare([]byte(secretHeader), []byte(expected)) != 1 {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Akses ditolak: otorisasi internal diperlukan", nil)
	}
	return c.Next()
}

func OptionalAuth(c *fiber.Ctx) error {
	tokenString := extractToken(c)
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
			isValid := false

			var account models.Account
			if err := database.DB.Select("last_login_at").Where("account_id = ?", accountID).First(&account).Error; err == nil {
				isValid = true
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
