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

	var customDesc []string
	if len(customSubject) > 1 && strings.TrimSpace(customSubject[1]) != "" {
		customDesc = append(customDesc, customSubject[1])
	}

	content := getOTPEmailContent(subject, customDesc...)

	mailer := gomail.NewMessage()
	mailer.SetAddressHeader("From", smtpEmail, "ToenToen Support")
	mailer.SetHeader("To", toEmail)
	mailer.SetHeader("Subject", content.Subject)

	// Plain text fallback
	plainBody := fmt.Sprintf("%s\n\n%s\n\nKode verifikasi (OTP) Anda adalah: %s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.\n\nSalam,\nTim ToenToen",
		content.Greeting, stripBasicHTML(content.Description), otp)
	mailer.SetBody("text/plain", plainBody)

	// HTML body
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f4f5f7; margin: 0; padding: 24px; color: #2d3748; }
		.container { max-width: 520px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; box-shadow: 0 4px 20px rgba(0,0,0,0.06); }
		.logo-row { text-align: center; margin-bottom: 24px; }
		.logo { font-size: 26px; font-weight: 800; color: #0077b6; letter-spacing: -0.5px; }
		.badge { display: inline-block; background: #e0f2fe; color: #0369a1; padding: 4px 14px; border-radius: 20px; font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; margin-top: 8px; }
		.greeting { font-size: 18px; font-weight: 700; margin-bottom: 12px; color: #1a202c; }
		.desc { font-size: 14px; line-height: 1.6; color: #4a5568; margin-bottom: 24px; }
		.otp-box { background: #f0f7ff; border: 2px dashed #0077b6; border-radius: 12px; padding: 20px; text-align: center; margin-bottom: 24px; }
		.otp-code { font-size: 34px; font-weight: 800; letter-spacing: 8px; color: #0077b6; font-family: 'Courier New', monospace; }
		.warning { font-size: 12px; color: #718096; text-align: center; line-height: 1.6; background: #f7fafc; padding: 12px; border-radius: 8px; }
		.footer { margin-top: 32px; border-top: 1px solid #edf2f7; padding-top: 18px; font-size: 11px; color: #a0aec0; text-align: center; line-height: 1.5; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo-row">
			<div class="logo">ToenToen</div>
			<div><span class="badge">%s</span></div>
		</div>
		<div class="greeting">%s</div>
		<div class="desc">%s</div>
		<div class="otp-box">
			<div class="otp-code">%s</div>
		</div>
		<div class="warning">
			%s
		</div>
		<div class="footer">
			Email ini dikirim otomatis oleh sistem ToenToen. Jika Anda tidak merasa melakukan permintaan ini, silakan abaikan pesan ini.
		</div>
	</div>
</body>
</html>`, content.Badge, content.Greeting, content.Description, otp, content.Warning)
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

type otpEmailContent struct {
	Subject     string
	Badge       string
	Greeting    string
	Description string
	Warning     string
}

func stripBasicHTML(s string) string {
	s = strings.ReplaceAll(s, "<strong>", "")
	s = strings.ReplaceAll(s, "</strong>", "")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	return s
}

func getOTPEmailContent(subject string, customDesc ...string) otpEmailContent {
	if len(customDesc) > 0 && strings.TrimSpace(customDesc[0]) != "" {
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Keamanan",
			Greeting:    "Halo Sobat ToenToen!",
			Description: strings.TrimSpace(customDesc[0]),
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun termasuk pihak ToenToen.",
		}
	}

	subjLower := strings.ToLower(subject)

	switch {
	case strings.Contains(subjLower, "aktivasi") && strings.Contains(subjLower, "admin"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Aktivasi Admin Toko",
			Greeting:    "Halo Rekan Admin!",
			Description: "Anda telah didaftarkan sebagai <strong>Admin Toko</strong> di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru dan mengaktifkan akun admin Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jangan pernah membagikan kode ini kepada siapapun demi keamanan akses toko Anda.",
		}
	case strings.Contains(subjLower, "admin") && strings.Contains(subjLower, "registrasi"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Registrasi Admin Toko",
			Greeting:    "Halo Rekan Admin!",
			Description: "Terima kasih telah mendaftar sebagai <strong>Admin Toko</strong> di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk menyelesaikan proses pendaftaran akun Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "admin") && (strings.Contains(subjLower, "reset") || strings.Contains(subjLower, "kata sandi")):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Reset Password Admin Toko",
			Greeting:    "Halo Rekan Admin!",
			Description: "Kami menerima permintaan untuk mengatur ulang kata sandi akun Admin Toko Anda di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun toko Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "admin") && (strings.Contains(subjLower, "login") || strings.Contains(subjLower, "masuk")):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Login Admin Toko",
			Greeting:    "Halo Rekan Admin!",
			Description: "Kami mendeteksi aktivitas login ke akun Admin Toko Anda di ToenToen. Masukkan kode verifikasi (OTP) berikut untuk melanjutkan:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jika Anda tidak merasa mencoba login, segera hubungi Owner toko Anda.",
		}
	case strings.Contains(subjLower, "admin") && strings.Contains(subjLower, "email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Email Admin",
			Greeting:    "Halo Rekan Admin!",
			Description: "Gunakan kode verifikasi (OTP) berikut untuk memverifikasi alamat email akun Admin Toko Anda di platform ToenToen:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun toko Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "registrasi"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Pendaftaran Akun Baru",
			Greeting:    "Selamat Datang di ToenToen!",
			Description: "Terima kasih telah bergabung di platform ToenToen. Gunakan kode verifikasi (OTP) berikut untuk menyelesaikan proses pendaftaran dan verifikasi akun Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "login") || strings.Contains(subjLower, "masuk"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Masuk (Login)",
			Greeting:    "Halo Sobat ToenToen!",
			Description: "Kami mendeteksi aktivitas login ke akun ToenToen Anda. Masukkan kode verifikasi (OTP) berikut untuk melanjutkan proses masuk:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jika Anda tidak merasa mencoba login, segera amankan akun Anda.",
		}
	case strings.Contains(subjLower, "reset password") || strings.Contains(subjLower, "lupa password") || strings.Contains(subjLower, "kata sandi"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Atur Ulang Kata Sandi",
			Greeting:    "Halo Sobat ToenToen!",
			Description: "Kami menerima permintaan untuk mengatur ulang kata sandi akun ToenToen Anda. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jika Anda tidak meminta perubahan kata sandi, abaikan pesan ini dan akun Anda tetap aman.",
		}
	case strings.Contains(subjLower, "penautan email") || strings.Contains(subjLower, "tautkan email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Penautan Email",
			Greeting:    "Halo Sobat ToenToen!",
			Description: "Anda sedang menautkan alamat email ini ke akun ToenToen Anda. Masukkan kode verifikasi (OTP) berikut untuk mengonfirmasi penautan email:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jangan berikan kode ini kepada orang lain.",
		}
	case strings.Contains(subjLower, "perubahan email") || strings.Contains(subjLower, "ganti email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Perubahan Alamat Email",
			Greeting:    "Halo Sobat ToenToen!",
			Description: "Anda mengajukan permohonan perubahan alamat email akun ToenToen. Masukkan kode verifikasi (OTP) berikut untuk mengonfirmasi bahwa email baru ini milik Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jangan berikan kode ini kepada orang lain.",
		}
	case strings.Contains(subjLower, "penghapusan akun") || strings.Contains(subjLower, "delete"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "⚠️ Konfirmasi Hapus Akun",
			Greeting:    "Pemberitahuan Keamanan Penting,",
			Description: "Kami menerima permintaan untuk menghapus akun ToenToen Anda secara permanen. Jika ini benar Anda, masukkan kode verifikasi (OTP) berikut untuk mengonfirmasi:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br><strong>PERHATIAN:</strong> Tindakan penghapusan akun bersifat permanen dan data Anda tidak dapat dipulihkan.",
		}
	case strings.Contains(subjLower, "pengalihan bisnis") || strings.Contains(subjLower, "transfer"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Otorisasi Pengalihan Toko",
			Greeting:    "Halo Pemilik Toko!",
			Description: "Anda sedang mengajukan otorisasi pengalihan kepemilikan bisnis di ToenToen. Masukkan kode verifikasi (OTP) berikut untuk mengotorisasi permohonan transfer toko Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Pastikan Anda hanya memberikan otorisasi transfer kepada pihak yang sah.",
		}
	case strings.Contains(subjLower, "email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Alamat Email",
			Greeting:    "Halo Sobat ToenToen!",
			Description: "Gunakan kode verifikasi (OTP) berikut untuk memverifikasi alamat email Anda di platform ToenToen:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	default:
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Keamanan",
			Greeting:    "Halo Sobat ToenToen!",
			Description: "Gunakan kode verifikasi berikut untuk melanjutkan proses verifikasi di ToenToen:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun termasuk pihak ToenToen.",
		}
	}
}

// SendWAOTP mengirimkan kode OTP ke nomor WhatsApp tujuan.
// Jika FONNTE_TOKEN disetel di environment, mengirim via API Fonnte.
// Jika FONNTE_TOKEN kosong (development/testing), kode dicatat ke log console dan mengembalikan nil.
func SendWAOTP(phoneNumber string, otp string, customMessage ...string) error {
	phoneNumber = strings.TrimSpace(phoneNumber)
	if phoneNumber == "" {
		return errors.New("nomor telepon tujuan tidak boleh kosong")
	}

	fonnteToken := strings.TrimSpace(os.Getenv("FONNTE_TOKEN"))
	if fonnteToken != "" {
		message := fmt.Sprintf("Halo sobat ToenToen!\n\nKode verifikasi (OTP) Anda adalah: *%s*\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.", otp)
		if len(customMessage) > 0 && strings.TrimSpace(customMessage[0]) != "" {
			message = strings.TrimSpace(customMessage[0])
		}
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

