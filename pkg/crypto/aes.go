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

// CiphertextPrefix identifies values encrypted by this service.
const CiphertextPrefix = "enc::"

const defaultDevKey = "toentoen-secret-key-32bytes-long!"

// getCandidateKeys returns ordered decryption keys to support key rotations and backward compatibility.
func getCandidateKeys() [][]byte {
	var keys [][]byte
	keyStr := ""
	if config.Envs != nil {
		keyStr = config.Envs.DataEncryptionKey
	}

	if keyStr != "" {
		// 64-character hex-encoded 32-byte key.
		if len(keyStr) == 64 {
			if decoded, err := hex.DecodeString(keyStr); err == nil && len(decoded) == 32 {
				keys = append(keys, decoded)
			}
		}
		// Raw 32-byte string.
		if len(keyStr) == 32 {
			keys = append(keys, []byte(keyStr))
		}
		// Normalized 32-byte buffer.
		norm := make([]byte, 32)
		copy(norm, []byte(keyStr))
		keys = append(keys, norm)
	}

	// Local development fallback key.
	defaultNorm := make([]byte, 32)
	copy(defaultNorm, []byte(defaultDevKey))
	keys = append(keys, defaultNorm)

	return keys
}

// getPrimaryAESKey returns the primary active key used for new encryptions.
func getPrimaryAESKey() ([]byte, error) {
	candidates := getCandidateKeys()
	if len(candidates) > 0 {
		return candidates[0], nil
	}
	return nil, errors.New("no encryption key configured")
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a prefixed base64 string ("enc::<payload>").
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

// Decrypt decrypts AES-GCM ciphertext prefixed with "enc::".
// Unprefixed or unencrypted strings are returned as-is for backward compatibility.
func Decrypt(encryptedText string) (string, error) {
	if encryptedText == "" {
		return "", nil
	}

	// Return legacy plaintext directly.
	if !strings.HasPrefix(encryptedText, CiphertextPrefix) {
		return encryptedText, nil
	}

	payload := strings.TrimPrefix(encryptedText, CiphertextPrefix)
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return encryptedText, nil
	}

	candidateKeys := getCandidateKeys()
	for _, key := range candidateKeys {
		if decrypted, err := decryptPayload(data, key); err == nil {
			return decrypted, nil
		}
	}

	// Fall back to original text if decryption fails with all candidate keys.
	return encryptedText, nil
}

// EncryptPointer encrypts a nullable string pointer.
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

// DecryptPointer decrypts a nullable string pointer.
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
