package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var nonAlphaNumUserRegex = regexp.MustCompile(`[^a-z0-9_]`)

// GenerateRandomUsername membuat username unik dari email/nomor telepon dan peran opsional
func GenerateRandomUsername(identifier string, rolePrefix ...string) string {
	identifier = strings.TrimSpace(strings.ToLower(identifier))
	base := identifier

	// Jika berupa email, ambil bagian sebelum @
	if atIdx := strings.Index(base, "@"); atIdx != -1 {
		base = base[:atIdx]
	}

	// Buang semua karakter non-alfanumerik
	base = nonAlphaNumUserRegex.ReplaceAllString(base, "")
	if len(base) > 12 {
		base = base[:12]
	}
	if base == "" {
		base = "user"
	}

	role := ""
	if len(rolePrefix) > 0 && strings.TrimSpace(rolePrefix[0]) != "" {
		r := strings.ToLower(strings.TrimSpace(rolePrefix[0]))
		switch r {
		case "customer":
			role = "c"
		case "owner":
			role = "o"
		case "business_admin":
			role = "adm"
		default:
			if len(r) > 3 {
				role = r[:3]
			} else {
				role = r
			}
		}
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	randNum := 100000 + n.Int64()

	if role != "" {
		return fmt.Sprintf("%s_%s%d", base, role, randNum)
	}
	return fmt.Sprintf("%s_%d", base, randNum)
}