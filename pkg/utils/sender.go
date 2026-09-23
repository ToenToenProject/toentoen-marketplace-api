package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
		if os.Getenv("APP_ENV") != "production" {
			log.Printf("[OTP-EMAIL-DEV] SMTP credentials belum diatur, OTP untuk %s adalah: %s", toEmail, otp)
			return nil
		}
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

// SendWAOTP mengirimkan kode OTP ke nomor WhatsApp tujuan.
// Jika FONNTE_TOKEN disetel di environment, mengirim via API Fonnte.
// Jika FONNTE_TOKEN kosong (development/testing), kode dicatat ke log console dan mengembalikan nil.
func SendWAOTP(phoneNumber string, otp string) error {
	phoneNumber = strings.TrimSpace(phoneNumber)
	if phoneNumber == "" {
		return errors.New("nomor telepon tujuan tidak boleh kosong")
	}

	fonnteToken := strings.TrimSpace(os.Getenv("FONNTE_TOKEN"))
	if fonnteToken != "" {
		message := fmt.Sprintf("Halo sobat ToenToen!\n\nKode verifikasi (OTP) Anda adalah: *%s*\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.", otp)
		payload := map[string]interface{}{
			"target":  phoneNumber,
			"message": message,
			"delay":   "1",
		}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("gagal memproses payload WhatsApp: %w", err)
		}

		req, err := http.NewRequest("POST", "https://api.fonnte.com/send", bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("gagal membuat request WhatsApp: %w", err)
		}
		req.Header.Set("Authorization", fonnteToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[OTP-WA-ERROR] Gagal mengirim WA OTP ke %s: %v", phoneNumber, err)
			return fmt.Errorf("gagal mengirim WA OTP: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[OTP-WA-ERROR] Fonnte gateway mengembalikan status: %d", resp.StatusCode)
			return fmt.Errorf("gagal mengirim OTP via WhatsApp, status gateway: %d", resp.StatusCode)
		}

		log.Printf("[OTP-WA-SUCCESS] WA OTP berhasil dikirim ke %s", phoneNumber)
		return nil
	}

	// Mode development / fallback jika belum ada token gateway WhatsApp
	log.Printf("[OTP-WA-DEV] Kode OTP verifikasi untuk nomor %s adalah: %s", phoneNumber, otp)
	return nil
}

// SendFeedbackEmail mengirimkan email masukan/feedback dari pengguna ke email resmi ToenToen
func SendFeedbackEmail(fromUserEmail, userName, role, subject, message string) error {
	fromUserEmail = strings.TrimSpace(fromUserEmail)
	if fromUserEmail == "" {
		fromUserEmail = "no-reply@toentoen.com"
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		userName = "Pengguna ToenToen"
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "Feedback Pengguna"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("isi pesan feedback tidak boleh kosong")
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
		log.Printf("[FEEDBACK-EMAIL-ERROR] SMTP credentials (SMTP_EMAIL / SMTP_PASSWORD) belum diatur di environment variable")
		return errors.New("konfigurasi server email belum lengkap")
	}

	receiverEmail := os.Getenv("FEEDBACK_RECEIVER_EMAIL")
	if receiverEmail == "" {
		receiverEmail = smtpEmail // default ke SMTP_EMAIL (email resmi toentoen)
	}

	mailer := gomail.NewMessage()
	mailer.SetAddressHeader("From", smtpEmail, fmt.Sprintf("%s via ToenToen Feedback", userName))
	mailer.SetHeader("Reply-To", fromUserEmail)
	mailer.SetHeader("To", receiverEmail)
	mailer.SetHeader("Subject", fmt.Sprintf("[Feedback ToenToen] %s - %s (%s)", subject, userName, role))

	plainBody := fmt.Sprintf("Feedback Baru dari Pengguna ToenToen:\n\nNama: %s\nEmail: %s\nPeran Akun: %s\nSubjek: %s\nWaktu: %s\n\nPesan:\n%s\n",
		userName, fromUserEmail, role, subject, time.Now().Format("02 Jan 2006 15:04 WIB"), message)
	mailer.SetBody("text/plain", plainBody)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f4f5f7; margin: 0; padding: 20px; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 32px; box-shadow: 0 4px 12px rgba(0,0,0,0.06); }
		.header { font-size: 20px; font-weight: 700; color: #0077b6; border-bottom: 2px solid #0077b6; padding-bottom: 12px; margin-bottom: 20px; }
		.message-box { background: #f8fafc; border-left: 4px solid #0077b6; border-radius: 4px; padding: 16px; margin-top: 20px; font-size: 14px; line-height: 1.6; white-space: pre-wrap; color: #222; }
		.footer { margin-top: 32px; border-top: 1px solid #eee; padding-top: 16px; font-size: 11px; color: #aaa; text-align: center; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">ToenToen Feedback</div>
		<p style="font-size: 14px; color: #666; margin-bottom: 16px;">Ada pesan masukan/saran baru dari pengguna aplikasi ToenToen:</p>
		<table style="width: 100%%; font-size: 14px; border-collapse: collapse;">
			<tr>
				<td style="padding: 6px 0; font-weight: 600; color: #555; width: 120px;">Pengirim:</td>
				<td style="padding: 6px 0; color: #111;">%s</td>
			</tr>
			<tr>
				<td style="padding: 6px 0; font-weight: 600; color: #555;">Email:</td>
				<td style="padding: 6px 0; color: #111;"><a href="mailto:%s">%s</a></td>
			</tr>
			<tr>
				<td style="padding: 6px 0; font-weight: 600; color: #555;">Peran Akun:</td>
				<td style="padding: 6px 0; color: #111;"><span style="background: #e0f2fe; color: #0369a1; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 600;">%s</span></td>
			</tr>
			<tr>
				<td style="padding: 6px 0; font-weight: 600; color: #555;">Subjek:</td>
				<td style="padding: 6px 0; color: #111; font-weight: 600;">%s</td>
			</tr>
			<tr>
				<td style="padding: 6px 0; font-weight: 600; color: #555;">Waktu:</td>
				<td style="padding: 6px 0; color: #888; font-size: 12px;">%s</td>
			</tr>
		</table>
		<div class="message-box">%s</div>
		<div class="footer">
			Anda dapat membalas pesan ini langsung untuk merespons pengguna melalui alamat email pengirim (<a href="mailto:%s">%s</a>).
		</div>
	</div>
</body>
</html>`, userName, fromUserEmail, fromUserEmail, role, subject, time.Now().Format("02 Jan 2006 15:04 WIB"), message, fromUserEmail, fromUserEmail)
	mailer.AddAlternative("text/html", htmlBody)

	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpEmail, smtpPassword)

	skipVerify := os.Getenv("SMTP_SKIP_VERIFY") == "true"
	dialer.TLSConfig = &tls.Config{
		ServerName:         smtpHost,
		InsecureSkipVerify: skipVerify,
	}

	if err := dialer.DialAndSend(mailer); err != nil {
		log.Printf("[FEEDBACK-EMAIL-ERROR] Gagal mengirim feedback ke %s: %v", receiverEmail, err)
		return fmt.Errorf("gagal mengirim email feedback: %w", err)
	}

	log.Printf("[FEEDBACK-EMAIL-SUCCESS] Email feedback dari %s (%s) berhasil dikirim ke %s", userName, fromUserEmail, receiverEmail)
	return nil
}

