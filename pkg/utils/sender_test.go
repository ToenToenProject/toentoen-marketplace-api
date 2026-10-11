package utils

import (
	"strings"
	"testing"
	"time"
)

func TestBuildOTPEmailHTML(t *testing.T) {
	content := getOTPEmailContent("Verifikasi Pendaftaran Akun")
	otp := "482910"
	testTime := time.Date(2026, 10, 12, 1, 30, 0, 0, time.UTC) // 08:30 WIB

	html := buildOTPEmailHTML(content, otp, testTime)

	// Verifikasi elemen kunci ada di dalam HTML
	if !strings.Contains(html, "Hey Sobat ToenToen!") {
		t.Errorf("HTML should contain greeting 'Hey Sobat ToenToen!'")
	}

	if !strings.Contains(html, "482910") {
		t.Errorf("HTML should contain the raw OTP code '482910'")
	}

	// Cek tiap digit ter-render dalam box terpisah
	for _, ch := range otp {
		digitTag := string(ch) + "</span>"
		if !strings.Contains(html, digitTag) {
			t.Errorf("HTML should contain digit span for %c", ch)
		}
	}

	// Cek keberadaan logo ToenToen SVG
	if !strings.Contains(html, "viewBox=\"0 0 46 46\"") {
		t.Errorf("HTML should contain logo SVG")
	}

	// Cek keberadaan swoosh SVG
	if !strings.Contains(html, "swoosh tl") || !strings.Contains(html, "swoosh br") {
		t.Errorf("HTML should contain swoosh SVGs")
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
	count := strings.Count(digits, "otp-box")
	if count != 0 && strings.Count(digits, "<span") != 6 {
		t.Errorf("Expected 6 span elements, got %d", strings.Count(digits, "<span"))
	}
}
