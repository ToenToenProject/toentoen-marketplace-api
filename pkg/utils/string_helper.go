package utils

import (
	"strings"
)

func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}

// MaskEmail memotong tampilan email (contoh: b***o@gmail.com)
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	username := parts[0]
	domain := parts[1]

	if len(username) <= 2 {
		return username[:1] + "***@" + domain
	}
	return string(username[0]) + "***" + string(username[len(username)-1]) + "@" + domain
}

// MaskPhoneNumber memotong tampilan nomor telepon (contoh: 0812****789)
func MaskPhoneNumber(phone string) string {
	if len(phone) <= 6 {
		return phone
	}
	prefix := phone[:4]
	suffix := phone[len(phone)-3:]
	return prefix + "****" + suffix
}
