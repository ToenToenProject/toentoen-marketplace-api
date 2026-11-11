package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/config"
)

// NormalizePhoneNumber menstandarkan format nomor telepon Indonesia / internasional:
// Menghilangkan spasi, strip, tanda kurung, dan simbol '+'.
// Jika berawalan '08', diubah menjadi '628'.
func NormalizePhoneNumber(phone string) string {
	clean := strings.TrimSpace(phone)
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, "(", "")
	clean = strings.ReplaceAll(clean, ")", "")
	clean = strings.TrimPrefix(clean, "+")
	if strings.HasPrefix(clean, "08") {
		clean = "62" + clean[1:]
	}
	return clean
}

// MatchPhoneNumbers memeriksa apakah dua representasi nomor telepon merujuk ke nomor yang sama
// (contoh: '08123456789' cocok dengan '+628123456789' atau '628123456789')
func MatchPhoneNumbers(phone1, phone2 string) bool {
	p1 := NormalizePhoneNumber(phone1)
	p2 := NormalizePhoneNumber(phone2)
	return p1 != "" && p1 == p2
}

// VerifyFirebasePhoneToken memverifikasi Firebase ID Token dari Flutter dan mengembalikan nomor telepon yang sah.
func VerifyFirebasePhoneToken(idToken string) (string, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return "", errors.New("Firebase ID Token tidak boleh kosong")
	}

	if config.FirebaseAuthClient == nil {
		return "", errors.New("Layanan Firebase Auth belum diinisialisasi di server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	decodedToken, err := config.FirebaseAuthClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", fmt.Errorf("Token verifikasi Firebase tidak valid atau telah kedaluwarsa: %w", err)
	}

	phoneClaim, ok := decodedToken.Claims["phone_number"]
	if !ok || phoneClaim == nil {
		return "", errors.New("Token Firebase ini tidak mengandung data nomor telepon yang diverifikasi")
	}

	phoneStr, ok := phoneClaim.(string)
	if !ok || strings.TrimSpace(phoneStr) == "" {
		return "", errors.New("Format nomor telepon pada klaim Firebase tidak valid")
	}

	return strings.TrimSpace(phoneStr), nil
}
