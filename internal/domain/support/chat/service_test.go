package chat

import (
	"strings"
	"testing"
	"toentoen-backend/internal/domain/shared/models"
)

func TestCleanGreetingContent(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Halo New! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
			expected: "Halo! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
		},
		{
			input:    "Halo Kak New! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
			expected: "Halo! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
		},
		{
			input:    "Halo Kak Budi! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
			expected: "Halo Kak Budi! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
		},
		{
			input:    "Halo! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
			expected: "Halo! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?",
		},
	}

	for _, c := range cases {
		actual := cleanGreetingContent(c.input)
		if actual != c.expected {
			t.Errorf("cleanGreetingContent(%q) = %q, expected %q", c.input, actual, c.expected)
		}
	}
}

func TestFormatGreetingMessage(t *testing.T) {
	// Case 1: Akun baru dengan default "New" "Account"
	accNew := &models.Account{
		FirstName: "New",
		LastName:  "Account",
		Username:  "new_user_123",
	}
	resNew := formatGreetingMessage(accNew)
	expectedNew := "Halo! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?"
	if resNew != expectedNew {
		t.Errorf("expected %q, got %q", expectedNew, resNew)
	}
	if strings.Contains(resNew, "New") {
		t.Errorf("resNew must not contain 'New': %s", resNew)
	}

	// Case 2: Akun hanya dengan FirstName "New" dan LastName kosong
	accNewOnly := &models.Account{
		FirstName: "New",
		LastName:  "",
	}
	resNewOnly := formatGreetingMessage(accNewOnly)
	if resNewOnly != expectedNew {
		t.Errorf("expected %q, got %q", expectedNew, resNewOnly)
	}

	// Case 3: Akun dengan nama asli pengguna
	accReal := &models.Account{
		FirstName: "Jonathan",
		LastName:  "Saputra",
	}
	resReal := formatGreetingMessage(accReal)
	expectedReal := "Halo Kak Jonathan! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?"
	if resReal != expectedReal {
		t.Errorf("expected %q, got %q", expectedReal, resReal)
	}

	// Case 4: Akun tanpa nama tetapi punya username valid
	accUser := &models.Account{
		FirstName: "",
		LastName:  "",
		Username:  "antigravity_dev",
	}
	resUser := formatGreetingMessage(accUser)
	expectedUser := "Halo Kak antigravity_dev! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?"
	if resUser != expectedUser {
		t.Errorf("expected %q, got %q", expectedUser, resUser)
	}

	// Case 5: Akun nil
	resNil := formatGreetingMessage(nil)
	if resNil != expectedNew {
		t.Errorf("expected %q, got %q", expectedNew, resNil)
	}
}
