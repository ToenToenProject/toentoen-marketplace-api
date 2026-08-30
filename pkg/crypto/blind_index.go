package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"toentoen-backend/internal/config"
)

// getBlindIndexKey mengambil secret key untuk HMAC dari config
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

// GenerateBlindIndex menghasilkan hash deterministik HMAC-SHA256 dari sebuah input (seperti NIK atau nomor identitas).
// Ini memungkinkan pencarian cepat / pengecekan duplikasi pada basis data tanpa perlu mendekripsi ciphertext.
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
