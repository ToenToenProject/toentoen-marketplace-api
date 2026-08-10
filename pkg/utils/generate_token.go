package utils

import (
	"time"
	"toentoen-backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(id string, role string) (string, error) {
	claims := jwt.MapClaims{
		"account_id": id,
		"role":       role,
		"iat":        time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(config.Envs.JWTSecret)) 
}