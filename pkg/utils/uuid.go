package utils

import (
	"crypto/rand"
	"fmt"
	"time"
)

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