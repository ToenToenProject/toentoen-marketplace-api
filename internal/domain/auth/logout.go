package auth

import (
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var authCookies = []string{
	"auth_token",
	"admin_token",
	"owner_token",
	"biz_admin_token",
	"user_token",
}

func Logout(c *fiber.Ctx) error {
	// Revoke bearer/cookie JWT server-side via jti denylist in Redis.
	if config.RDB != nil && config.Envs != nil {
		tokenStr := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization"), "Bearer "))
		if tokenStr == "" {
			for _, cookieName := range authCookies {
				if v := strings.TrimSpace(c.Cookies(cookieName)); v != "" {
					tokenStr = v
					break
				}
			}
		}
		if tokenStr != "" {
			if tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("invalid alg")
				}
				return []byte(config.Envs.JWTSecret), nil
			}); err == nil && tok.Valid {
				if claims, ok := tok.Claims.(jwt.MapClaims); ok {
					if jti, ok := claims["jti"].(string); ok && jti != "" {
						ttl := 24 * time.Hour
						if exp, ok := claims["exp"].(float64); ok {
							if rem := time.Until(time.Unix(int64(exp), 0)); rem > 0 {
								ttl = rem
							}
						}
						config.RDB.Set(config.Ctx, "revoked_jti:"+jti, "1", ttl)
					}
				}
			}
		}
	}

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

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil logout. Semua sesi telah dibersihkan.", nil)
}