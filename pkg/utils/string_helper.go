package utils

import (
	"fmt"
	"regexp"
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

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9_]`)

// NormalizeHashtag membersihkan string hashtag (menghapus #, menghapus spasi, lowercase, dan membuang simbol non-alfanumerik)
func NormalizeHashtag(tag string) string {
	clean := strings.TrimSpace(tag)
	for strings.HasPrefix(clean, "#") {
		clean = strings.TrimPrefix(clean, "#")
	}
	clean = strings.ToLower(clean)
	clean = strings.ReplaceAll(clean, " ", "")
	clean = nonAlphanumericRegex.ReplaceAllString(clean, "")
	return clean
}

// NormalizeHashtags memproses slice of hashtags, membuang string kosong, dan mendeduplikasi hasilnya
func NormalizeHashtags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := make(map[string]bool)
	var result []string
	for _, t := range tags {
		clean := NormalizeHashtag(t)
		if clean != "" && !seen[clean] {
			seen[clean] = true
			result = append(result, clean)
		}
	}
	if result == nil {
		return []string{}
	}
	return result
}

// ValidateHashtags memvalidasi jumlah dan panjang maksimal tiap kata pada hashtags
func ValidateHashtags(hashtags []string, maxCount, maxLen int) error {
	if len(hashtags) > maxCount {
		return fmt.Errorf("jumlah hashtag maksimal %d kata (diberikan: %d)", maxCount, len(hashtags))
	}
	for _, h := range hashtags {
		if len(h) > maxLen {
			return fmt.Errorf("panjang tiap hashtag maksimal %d karakter (hashtag '%s' memiliki %d karakter)", maxLen, h, len(h))
		}
	}
	return nil
}

