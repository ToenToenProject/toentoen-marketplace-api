package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"toentoen-backend/internal/config"
)

// CiphertextPrefix adalah penanda opsional untuk mendeteksi data terenkripsi secara deterministik
const CiphertextPrefix = "enc::"

const defaultDevKey = "toentoen-secret-key-32bytes-long!"

// getCandidateKeys menghasilkan daftar kandidat kunci untuk dekripsi (termasuk backward compatibility)
func getCandidateKeys() [][]byte {
	var keys [][]byte
	keyStr := ""
	if config.Envs != nil {
		keyStr = config.Envs.DataEncryptionKey
	}

	if keyStr != "" {
		// 1. Jika hex string 64-karakter (32 byte binary)
		if len(keyStr) == 64 {
			if decoded, err := hex.DecodeString(keyStr); err == nil && len(decoded) == 32 {
				keys = append(keys, decoded)
			}
		}
		// 2. Jika persis 32 byte
		if len(keyStr) == 32 {
			keys = append(keys, []byte(keyStr))
		}
		// 3. Normalisasi slice 32 byte
		norm := make([]byte, 32)
		copy(norm, []byte(keyStr))
		keys = append(keys, norm)
	}

	// 4. Default fallback key untuk backward-compatibility data lokal
	defaultNorm := make([]byte, 32)
	copy(defaultNorm, []byte(defaultDevKey))
	keys = append(keys, defaultNorm)

	return keys
}

// getPrimaryAESKey mengambil kunci utama untuk proses enkripsi baru
func getPrimaryAESKey() ([]byte, error) {
	candidates := getCandidateKeys()
	if len(candidates) > 0 {
		return candidates[0], nil
	}
	return nil, errors.New("no encryption key configured")
}

// Encrypt mengenkripsi plaintext string menjadi string base64 berformat "enc::<base64_payload>"
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := getPrimaryAESKey()
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

func decryptPayload(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext terlalu pendek")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
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

	candidateKeys := getCandidateKeys()
	for _, key := range candidateKeys {
		if decrypted, err := decryptPayload(data, key); err == nil {
			return decrypted, nil
		}
	}

	// Jika semua kandidat kunci gagal decrypt (misal data rusak), fallback ke teks asli
	return encryptedText, nil
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
