package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"toentoen-backend/internal/config"
)

// getBlindIndexKey resolves the blind index HMAC secret key from configuration.
func getBlindIndexKey() []byte {
	keyStr := ""
	if config.Envs != nil {
		keyStr = config.Envs.BlindIndexKey
	}
	if keyStr == "" {
		keyStr = "toentoen-blind-index-key-32bytes!" // fallback default dev key
	}
	return []byte(keyStr)
}

// GenerateBlindIndex computes a deterministic HMAC-SHA256 hash for fast exact-match lookups without decrypting ciphertext.
func GenerateBlindIndex(input string) string {
	cleaned := strings.TrimSpace(input)
	if cleaned == "" {
		return ""
	}

	key := getBlindIndexKey()
	h := hmac.New(sha256.New, key)
	h.Write([]byte(cleaned))

	return hex.EncodeToString(h.Sum(nil))
}
