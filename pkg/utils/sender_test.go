package utils

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestBuildOTPEmailHTML(t *testing.T) {
	content := getOTPEmailContent("Verifikasi Pendaftaran Akun")
	otp := "482910"
	testTime := time.Date(2026, 10, 12, 1, 30, 0, 0, time.UTC) // 08:30 WIB

	html := buildOTPEmailHTML(content, otp, testTime)
	_ = os.WriteFile("C:/Users/ADMIN/.gemini/antigravity-ide/brain/a81798aa-7e37-4b12-bc8c-01bc76ac888a/scratch/preview_email.html", []byte(html), 0644)

	// Verifikasi greeting dan teks utama
	if !strings.Contains(html, "Hey Sobat ToenToen!") {
		t.Errorf("HTML should contain greeting 'Hey Sobat ToenToen!'")
	}

	// Verifikasi kode OTP ada sebagai teks utuh
	if !strings.Contains(html, "482910") {
		t.Errorf("HTML should contain the raw OTP code '482910'")
	}

	// Pastikan background hitam (#1e1e1e) sudah dihapus dan diganti tema terang (#f1f5f9)
	if strings.Contains(html, "#1e1e1e") {
		t.Errorf("HTML should not contain dark background #1e1e1e")
	}
	if !strings.Contains(html, "#f1f5f9") {
		t.Errorf("HTML should contain modern light background #f1f5f9")
	}

	// Pastikan logo resmi ToenToen dari asset digunakan via CID
	if !strings.Contains(html, "cid:toentoen-logo.png") {
		t.Errorf("HTML should contain official logo cid:toentoen-logo.png")
	}

	// Pastikan tidak ada pemisahan table per digit yang menyebabkan \n saat disalin
	if strings.Contains(html, "4</td>") {
		t.Errorf("HTML should not contain separate table cell per digit")
	}

	// Cek footer waktu WIB
	if !strings.Contains(html, "WIB") {
		t.Errorf("HTML footer should contain WIB timezone")
	}

	// Cek warning 5 menit
	if !strings.Contains(html, "5 menit") {
		t.Errorf("HTML should mention 5 minutes validity")
	}
}

func TestFormatWIBDate(t *testing.T) {
	testTime := time.Date(2026, time.October, 14, 12, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	res := formatWIBDate(testTime)

	if !strings.Contains(res, "Rabu") || !strings.Contains(res, "14 Oktober 2026") || !strings.Contains(res, "12:00 WIB") {
		t.Errorf("Unexpected formatted date: %s", res)
	}
}

func TestRenderOTPDigitsHTML(t *testing.T) {
	digits := renderOTPDigitsHTML("123456")
	if digits != "123456" {
		t.Errorf("Expected '123456', got '%s'", digits)
	}
}

func TestFindLogoPath(t *testing.T) {
	path := findLogoPath()
	if path == "" {
		t.Errorf("Expected logo to be found, got empty")
	}
	if !strings.HasSuffix(path, "toentoen-logo.png") {
		t.Errorf("Expected path to end with toentoen-logo.png, got %s", path)
	}
}
