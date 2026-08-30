package crypto

import (
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	plaintexts := []string{
		"3201012345670001",
		"Jl. Sudirman No. 123, RT 01 / RW 02, Kelurahan Menteng",
		"https://s3.aws.com/toentoen/ktp/photo_123.jpg",
		"Pesan rahasia antara customer dan merchant 123!@#$%",
		"",
	}

	for _, pt := range plaintexts {
		encrypted, err := Encrypt(pt)
		if err != nil {
			t.Fatalf("Gagal enkripsi plaintext '%s': %v", pt, err)
		}

		if pt != "" && !strings.HasPrefix(encrypted, CiphertextPrefix) {
			t.Fatalf("Ciphertext harus diawali prefix '%s', didapat: %s", CiphertextPrefix, encrypted)
		}

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Gagal dekripsi ciphertext '%s': %v", encrypted, err)
		}

		if decrypted != pt {
			t.Fatalf("Hasil dekripsi '%s' tidak cocok dengan plaintext asli '%s'", decrypted, pt)
		}
	}
}

func TestGracefulFallbackLegacyPlaintext(t *testing.T) {
	legacyPlaintext := "3201012345670001" // Tidak ada prefix "enc::"
	decrypted, err := Decrypt(legacyPlaintext)
	if err != nil {
		t.Fatalf("Harusnya tidak error pada data legacy: %v", err)
	}

	if decrypted != legacyPlaintext {
		t.Fatalf("Data legacy harus dikembalikan apa adanya, didapat: %s", decrypted)
	}
}

func TestBlindIndexDeterministic(t *testing.T) {
	input1 := "3201012345670001"
	input2 := " 3201012345670001 "
	input3 := "3201012345670002"

	hash1 := GenerateBlindIndex(input1)
	hash2 := GenerateBlindIndex(input2)
	hash3 := GenerateBlindIndex(input3)

	if hash1 == "" {
		t.Fatal("Blind index tidak boleh kosong")
	}
	if hash1 != hash2 {
		t.Fatalf("Hash deterministik harus sama untuk input yang sama (meskipun ada whitespace): %s != %s", hash1, hash2)
	}
	if hash1 == hash3 {
		t.Fatal("Hash untuk input berbeda tidak boleh sama")
	}
}
