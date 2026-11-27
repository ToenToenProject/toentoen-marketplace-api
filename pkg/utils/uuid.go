package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateSecureToken returns a cryptographically random hex token of numBytes*2 chars.
func GenerateSecureToken(numBytes int) (string, error) {
	if numBytes <= 0 {
		numBytes = 32
	}
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return "XXXXXX"
	}

	for i := 0; i < length; i++ {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func GenerateUUID(prefix string) string {
	dateStr := time.Now().Format("20060102150405")

	randomStr := GenerateRandomString(6)

	if prefix == "" {
		return fmt.Sprintf("%s-%s", dateStr, randomStr)
	}

	return fmt.Sprintf("%s-%s-%s", prefix, dateStr, randomStr)
}