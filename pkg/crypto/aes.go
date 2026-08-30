package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"toentoen-backend/internal/config"
)

// CiphertextPrefix adalah penanda opsional untuk mendeteksi data terenkripsi secara deterministik
const CiphertextPrefix = "enc::"

// getAESKey mengambil 32-byte key dari config
func getAESKey() ([]byte, error) {
	keyStr := ""
	if config.Envs != nil {
		keyStr = config.Envs.DataEncryptionKey
	}
	if keyStr == "" {
		keyStr = "toentoen-secret-key-32bytes-long!" // fallback default dev key
	}

	key := []byte(keyStr)
	if len(key) != 32 {
		// Jika panjang kunci bukan 32 byte, normalisasi ke 32 byte
		normalized := make([]byte, 32)
		copy(normalized, key)
		return normalized, nil
	}
	return key, nil
}

// Encrypt mengenkripsi plaintext string menjadi string base64 berformat "enc::<base64_payload>"
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := getAESKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return CiphertextPrefix + encoded, nil
}

// Decrypt mendekripsi ciphertext berformat "enc::<base64_payload>".
// Jika data tidak memiliki prefix "enc::" (misal data lama/plaintext),
// fungsi ini akan mengembalikan data asli secara aman (graceful fallback).
func Decrypt(encryptedText string) (string, error) {
	if encryptedText == "" {
		return "", nil
	}

	// Graceful fallback jika data belum terenkripsi (data legacy/plaintext)
	if !strings.HasPrefix(encryptedText, CiphertextPrefix) {
		return encryptedText, nil
	}

	payload := strings.TrimPrefix(encryptedText, CiphertextPrefix)
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		// Jika decode gagal, kembalikan string asli agar tidak merusak data
		return encryptedText, nil
	}

	key, err := getAESKey()
	if err != nil {
		return encryptedText, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return encryptedText, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedText, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return encryptedText, errors.New("ciphertext terlalu pendek")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Jika gagal decrypt dengan key (misal corrupted), fallback ke teks asli
		return encryptedText, nil
	}

	return string(plaintext), nil
}

// EncryptPointer adalah helper untuk mengenkripsi pointer string (*string)
func EncryptPointer(plaintext *string) (*string, error) {
	if plaintext == nil || *plaintext == "" {
		return plaintext, nil
	}

	encrypted, err := Encrypt(*plaintext)
	if err != nil {
		return nil, err
	}
	return &encrypted, nil
}

// DecryptPointer adalah helper untuk mendekripsi pointer string (*string)
func DecryptPointer(ciphertext *string) (*string, error) {
	if ciphertext == nil || *ciphertext == "" {
		return ciphertext, nil
	}

	decrypted, err := Decrypt(*ciphertext)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}
