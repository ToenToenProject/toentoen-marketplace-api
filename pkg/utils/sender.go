package utils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

// SendEmailOTP mengirimkan kode OTP ke alamat email tujuan.
// Mengembalikan error jika pengiriman gagal agar caller dapat menangani kegagalan dengan tepat.
func SendEmailOTP(toEmail string, otp string, customSubject ...string) error {
	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return errors.New("alamat email tujuan tidak boleh kosong")
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}

	smtpPort := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			smtpPort = pInt
		}
	}

	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if smtpEmail == "" || smtpPassword == "" {
		log.Printf("[OTP-EMAIL-ERROR] SMTP credentials (SMTP_EMAIL / SMTP_PASSWORD) belum diatur di environment variable")
		return errors.New("konfigurasi server email belum lengkap")
	}

	subject := "Kode Verifikasi ToenToen"
	if len(customSubject) > 0 && strings.TrimSpace(customSubject[0]) != "" {
		subject = strings.TrimSpace(customSubject[0])
	}

	mailer := gomail.NewMessage()
	mailer.SetAddressHeader("From", smtpEmail, "ToenToen Support")
	mailer.SetHeader("To", toEmail)
	mailer.SetHeader("Subject", subject)

	// Plain text fallback
	plainBody := fmt.Sprintf("Halo!\n\nKode verifikasi (OTP) Anda adalah: %s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.\n\nSalam,\nTim ToenToen", otp)
	mailer.SetBody("text/plain", plainBody)

	// HTML body
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f4f5f7; margin: 0; padding: 20px; color: #333; }
		.container { max-width: 500px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 32px; box-shadow: 0 4px 12px rgba(0,0,0,0.06); }
		.logo { font-size: 24px; font-weight: 700; color: #0077b6; margin-bottom: 24px; text-align: center; }
		.greeting { font-size: 18px; font-weight: 600; margin-bottom: 12px; }
		.desc { font-size: 14px; line-height: 1.6; color: #555; margin-bottom: 24px; }
		.otp-box { background: #f0f7ff; border: 2px dashed #0077b6; border-radius: 8px; padding: 18px; text-align: center; margin-bottom: 24px; }
		.otp-code { font-size: 32px; font-weight: 800; letter-spacing: 6px; color: #0077b6; }
		.warning { font-size: 12px; color: #888; text-align: center; line-height: 1.5; }
		.footer { margin-top: 32px; border-top: 1px solid #eee; padding-top: 16px; font-size: 11px; color: #aaa; text-align: center; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">ToenToen</div>
		<div class="greeting">Halo!</div>
		<div class="desc">Gunakan kode verifikasi berikut untuk melanjutkan proses verifikasi di ToenToen:</div>
		<div class="otp-box">
			<div class="otp-code">%s</div>
		</div>
		<div class="warning">
			Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>
			Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun termasuk pihak ToenToen.
		</div>
		<div class="footer">
			Email ini dikirim otomatis oleh sistem ToenToen. Jika Anda tidak merasa meminta kode ini, silakan abaikan pesan ini.
		</div>
	</div>
</body>
</html>`, otp)
	mailer.AddAlternative("text/html", htmlBody)

	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpEmail, smtpPassword)

	skipVerify := os.Getenv("SMTP_SKIP_VERIFY") == "true"
	dialer.TLSConfig = &tls.Config{
		ServerName:         smtpHost,
		InsecureSkipVerify: skipVerify,
	}

	if err := dialer.DialAndSend(mailer); err != nil {
		log.Printf("[OTP-EMAIL-ERROR] Gagal mengirim email OTP ke %s via %s:%d: %v", toEmail, smtpHost, smtpPort, err)
		return fmt.Errorf("gagal mengirim email OTP: %w", err)
	}

	log.Printf("[OTP-EMAIL-SUCCESS] Email OTP berhasil dikirim ke %s", toEmail)
	return nil
}

// SendWAOTP dinonaktifkan sementara karena layanan OTP via WhatsApp/nomor telepon masih dalam tahap pengembangan.
func SendWAOTP(phoneNumber string, otp string) error {
	return errors.New("layanan OTP via nomor telepon/WhatsApp sedang dalam tahap pengembangan")
}
