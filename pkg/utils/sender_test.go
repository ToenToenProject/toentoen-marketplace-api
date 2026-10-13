package utils

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"gopkg.in/gomail.v2"
)

func TestBuildOTPEmailHTML(t *testing.T) {
	content := getOTPEmailContent("Verifikasi Pendaftaran Akun")
	otp := "482910"
	testTime := time.Date(2026, 10, 12, 1, 30, 0, 0, time.UTC) // 08:30 WIB

	html := buildOTPEmailHTML(content, otp, testTime)
	_ = os.MkdirAll("C:/Users/ADMIN/.gemini/antigravity/brain/c91a8afd-aa22-40f8-94dd-962b4148c3ce/scratch", 0755)
	_ = os.WriteFile("C:/Users/ADMIN/.gemini/antigravity/brain/c91a8afd-aa22-40f8-94dd-962b4148c3ce/scratch/preview_email.html", []byte(html), 0644)

	// Verifikasi greeting dan teks utama
	if !strings.Contains(html, "Hey Sobat ToenToen!") {
		t.Errorf("HTML should contain greeting 'Hey Sobat ToenToen!'")
	}

	// Verifikasi kode OTP ada sebagai teks utuh
	if !strings.Contains(html, "482910") {
		t.Errorf("HTML should contain the raw OTP code '482910'")
	}

	// Pastikan background hitam (#1e1e1e) sudah dihapus dan menggunakan background bersih (#f8fafc) serta card putih (#ffffff)
	if strings.Contains(html, "#1e1e1e") {
		t.Errorf("HTML should not contain dark background #1e1e1e")
	}
	if !strings.Contains(html, "#f8fafc") {
		t.Errorf("HTML should contain clean light background #f8fafc")
	}
	if !strings.Contains(html, "#ffffff") {
		t.Errorf("HTML should contain white card #ffffff")
	}

	// Pastikan tombol Copy OTP ada sesuai desain referensi
	if !strings.Contains(html, "Copy OTP") {
		t.Errorf("HTML should contain 'Copy OTP' button")
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
	if strings.Count(digits, "otp-digit-box") != 6 {
		t.Errorf("Expected 6 otp-digit-box elements, got count %d", strings.Count(digits, "otp-digit-box"))
	}
	for _, ch := range "123456" {
		if !strings.Contains(digits, string(ch)) {
			t.Errorf("Expected digit %c in rendered HTML", ch)
		}
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

func TestGetLogoBytes(t *testing.T) {
	bytes := getLogoBytes()
	if len(bytes) == 0 {
		t.Fatalf("Expected getLogoBytes to return non-empty embedded bytes")
	}
}

func TestBuildOTPEmailHTML_FallbackBadge(t *testing.T) {
	content := getOTPEmailContent("Verifikasi Akun")
	html := buildOTPEmailHTML(content, "123456", time.Now(), false)
	if !strings.Contains(html, "TOEN") {
		t.Errorf("Expected fallback badge 'TOEN' when hasLogo is false")
	}
	if strings.Contains(html, "cid:toentoen-logo.png") {
		t.Errorf("Should not contain cid:toentoen-logo.png when hasLogo is false")
	}
}

func TestGomailEmbedOutput(t *testing.T) {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", "support@toentoen.com")
	mailer.SetHeader("To", "user@example.com")
	mailer.SetHeader("Subject", "Test Subject")
	mailer.SetBody("text/plain", "Hello world")
	mailer.Embed("toentoen-logo.png", gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write([]byte("fake png content"))
		return err
	}), gomail.SetHeader(map[string][]string{
		"Content-Type": {"image/png"},
	}))
	content := getOTPEmailContent("Verifikasi Akun")
	mailer.AddAlternative("text/html", buildOTPEmailHTML(content, "123456", time.Now()))

	var buf strings.Builder
	_, err := mailer.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo error: %v", err)
	}
	fullMIME := buf.String()
	lines := strings.Split(fullMIME, "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "Content-") || strings.HasPrefix(l, "--") {
			t.Logf("HEADER: %s", l)
		}
	}
}

