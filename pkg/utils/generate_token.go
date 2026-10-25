package utils

import (
	"time"
	"toentoen-backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(id string, role string, email ...string) (string, error) {
	claims := jwt.MapClaims{
		"account_id": id,
		"role":       role,
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	if len(email) > 0 && email[0] != "" {
		claims["email"] = email[0]
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(config.Envs.JWTSecret))
}