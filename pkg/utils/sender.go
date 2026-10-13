package utils

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

//go:embed assets/toentoen-logo.png
var embeddedLogoBytes []byte

// getLogoBytes mengembalikan byte logo resmi ToenToen.
// Prioritas: byte embedded langsung dari binary (go:embed) -> fallback pembacaan disk file.
func getLogoBytes() []byte {
	if len(embeddedLogoBytes) > 0 {
		return embeddedLogoBytes
	}
	if p := findLogoPath(); p != "" {
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			return data
		}
	}
	return nil
}

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
	plainBody := fmt.Sprintf("%s\n\n%s\n\nKode verifikasi (OTP) Anda adalah: %s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.\n\nJika Anda tidak merasa melakukan permintaan verifikasi ini, silakan abaikan pesan ini.\n\nSalam,\nTim ToenToen",
		content.Greeting, stripBasicHTML(content.Description), otp)
	mailer.SetBody("text/plain", plainBody)

	// Embed logo resmi ToenToen (selalu tersedia di memori via go:embed)
	logoBytes := getLogoBytes()
	hasLogo := len(logoBytes) > 0
	if hasLogo {
		mailer.Embed("toentoen-logo.png", gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(logoBytes)
			return err
		}), gomail.SetHeader(map[string][]string{
			"Content-Type": {"image/png"},
		}))
	}

	// HTML body (modern card design dengan tema ToenToen)
	htmlBody := buildOTPEmailHTML(content, otp, time.Now(), hasLogo)
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

func formatWIBDate(t time.Time) string {
	wib := time.FixedZone("WIB", 7*3600)
	tWIB := t.In(wib)

	dayNames := [...]string{
		"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu",
	}
	monthNames := [...]string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}

	dayName := dayNames[tWIB.Weekday()]
	monthName := monthNames[tWIB.Month()]

	return fmt.Sprintf("%s, %d %s %d, %02d:%02d WIB",
		dayName, tWIB.Day(), monthName, tWIB.Year(), tWIB.Hour(), tWIB.Minute())
}

func findLogoPath() string {
	candidates := []string{
		"assets/logo/toentoen-logo.png",
		"./assets/logo/toentoen-logo.png",
		"../assets/logo/toentoen-logo.png",
		"../../assets/logo/toentoen-logo.png",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func renderOTPDigitsHTML(otp string) string {
	clean := strings.TrimSpace(otp)
	var sb strings.Builder
	for i, ch := range clean {
		margin := "margin-right: 4px;"
		if i == len(clean)-1 {
			margin = ""
		}
		sb.WriteString(fmt.Sprintf(
			`<span class="otp-digit-box" style="display: inline-block; width: 26px; height: 36px; line-height: 36px; background-color: #d5f3f1; border-radius: 5px; font-family: 'Poppins', Arial, sans-serif; font-size: 19px; font-weight: 700; color: #007A74; text-align: center; vertical-align: middle; %s">%c</span>`,
			margin, ch,
		))
	}
	return sb.String()
}

func buildOTPEmailHTML(content otpEmailContent, otp string, createdAt time.Time, hasLogoOpt ...bool) string {
	footerTime := formatWIBDate(createdAt)
	cleanOTP := strings.TrimSpace(otp)
	otpDigitsHTML := renderOTPDigitsHTML(cleanOTP)

	hasLogo := true
	if len(hasLogoOpt) > 0 {
		hasLogo = hasLogoOpt[0]
	}

	logoHTML := `<table cellpadding="0" cellspacing="0" border="0" width="46" height="46" bgcolor="#00B5AD" style="width: 46px; height: 46px; background-color: #00B5AD; border-radius: 50%; text-align: center; border-collapse: separate;"><tr><td align="center" valign="middle" style="color: #ffffff; font-family: 'Poppins', Arial, sans-serif; font-size: 11px; font-weight: 800; line-height: 12px; letter-spacing: 0.5px; padding: 0;">TO<br>EN<span style="display:none;">TOEN</span></td></tr></table>`
	if hasLogo {
		logoHTML = `<img src="cid:toentoen-logo.png" alt="ToenToen" width="46" height="46" style="display: block; width: 46px; height: 46px; max-width: 46px; max-height: 46px; border-radius: 50%; object-fit: cover;" />`
	}

	rawHTML := `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>{{SUBJECT}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Poppins:wght@400;500;600;700;800&display=swap" rel="stylesheet">
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body, table, td, a { -webkit-text-size-adjust: 100%; -ms-text-size-adjust: 100%; }
  table, td { mso-table-lspace: 0pt; mso-table-rspace: 0pt; }
  img { -ms-interpolation-mode: bicubic; border: 0; outline: none; text-decoration: none; }
  body {
    background-color: #f8fafc !important;
    font-family: 'Poppins', -apple-system, BlinkMacSystemFont, Arial, sans-serif;
    margin: 0;
    padding: 0;
    color: #1e293b;
  }
  @media only screen and (max-width: 480px) {
    .email-wrapper { padding: 16px 8px !important; }
    .card-box { padding: 24px 20px 20px !important; border-radius: 14px !important; }
    .greeting-title { font-size: 16px !important; }
    .otp-digit-box { width: 24px !important; height: 34px !important; line-height: 34px !important; font-size: 17px !important; margin-right: 3px !important; }
  }
</style>
</head>
<body bgcolor="#f8fafc" style="background-color: #f8fafc; font-family: 'Poppins', -apple-system, BlinkMacSystemFont, Arial, sans-serif; margin: 0; padding: 0;">
  <!-- Full-width Outer Table -->
  <table class="email-wrapper" width="100%" border="0" cellspacing="0" cellpadding="0" bgcolor="#f8fafc" style="background-color: #f8fafc; width: 100%; min-height: 100vh; padding: 32px 12px; margin: 0;">
    <tr>
      <td align="center" valign="top">
        <!-- Container table (420px max-width) -->
        <table width="100%" border="0" cellspacing="0" cellpadding="0" style="max-width: 420px; width: 100%; margin: 0 auto;">
          <!-- Main White Card -->
          <tr>
            <td>
              <table class="card-box" width="100%" border="0" cellspacing="0" cellpadding="0" bgcolor="#ffffff" style="background-color: #ffffff; border-radius: 16px; border: 1px solid #e2e8f0; overflow: hidden; padding: 32px 28px 24px; color: #1e293b; box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05);">
                
                <!-- Header Row: Greeting on Left, Circular ToenToen Logo on Right -->
                <tr>
                  <td>
                    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="margin-bottom: 14px;">
                      <tr>
                        <td valign="middle" align="left" style="padding-right: 12px;">
                          <div class="greeting-title" style="font-family: 'Poppins', Arial, sans-serif; font-size: 17px; font-weight: 700; line-height: 1.35; color: #0f172a;">
                            {{GREETING}}
                          </div>
                        </td>
                        <td valign="middle" align="right" width="46" style="width: 46px; min-width: 46px;">
                          {{LOGO_HTML}}
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>

                <!-- Description Paragraph -->
                <tr>
                  <td>
                    <p style="font-family: 'Poppins', Arial, sans-serif; font-size: 12.5px; line-height: 1.6; color: #334155; margin: 0 0 16px 0;">
                      {{DESCRIPTION}}
                    </p>
                  </td>
                </tr>

                <!-- OTP Section: 6 Soft Mint Digit Boxes & Copy OTP Button -->
                <tr>
                  <td align="left" style="padding-bottom: 20px;">
                    <!-- 6 Digit Boxes Container (Continuous text flow for clean copy without \n) -->
                    <div style="margin-bottom: 8px; user-select: all; -webkit-user-select: all; cursor: pointer;">
                      {{OTP_BOXES}}
                    </div>

                    <!-- Copy OTP Button (Left-aligned, matching boxes width) -->
                    <table cellpadding="0" cellspacing="0" border="0" style="width: 100%; max-width: 176px;">
                      <tr>
                        <td align="center" style="background-color: #e7e7e8; border-radius: 6px; padding: 7px 14px; text-align: center; cursor: pointer; user-select: all; -webkit-user-select: all;">
                          <a href="javascript:void(0)" onclick="if(navigator.clipboard){navigator.clipboard.writeText('{{OTP_RAW}}'); alert('Kode OTP berhasil disalin: {{OTP_RAW}}');}" title="Salin OTP: {{OTP_RAW}}" style="text-decoration: none; display: block; width: 100%;">
                            <span style="font-family: 'Poppins', Arial, sans-serif; font-size: 11.5px; font-weight: 700; color: #555555; letter-spacing: 0.3px;">Copy OTP</span>
                          </a>
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>

                <!-- Security Explanatory Paragraphs (replacing Lorem Ipsum) -->
                <tr>
                  <td>
                    <p style="font-family: 'Poppins', Arial, sans-serif; font-size: 12px; line-height: 1.6; color: #334155; margin: 0 0 12px 0;">
                      {{WARNING}}
                    </p>
                    <p style="font-family: 'Poppins', Arial, sans-serif; font-size: 12px; line-height: 1.6; color: #64748b; margin: 0 0 20px 0;">
                      Jika Anda tidak merasa melakukan permintaan verifikasi ini, silakan abaikan pesan ini. Akun Anda tetap aman terlindungi.
                    </p>
                  </td>
                </tr>

                <!-- Footer with Timestamp (Centered) -->
                <tr>
                  <td style="border-top: 1px solid #f1f5f9; padding-top: 16px;">
                    <table width="100%" cellpadding="0" cellspacing="0" border="0">
                      <tr>
                        <td align="center" style="font-family: 'Poppins', Arial, sans-serif; font-size: 11.5px; font-weight: 500; color: #64748b; text-align: center;">
                          <span style="font-size: 13px; vertical-align: middle; margin-right: 5px;">⏱</span>
                          <span style="vertical-align: middle;">{{FOOTER_TIME}}</span>
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>

              </table>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`

	replacer := strings.NewReplacer(
		"{{SUBJECT}}", content.Subject,
		"{{BADGE}}", content.Badge,
		"{{GREETING}}", content.Greeting,
		"{{DESCRIPTION}}", content.Description,
		"{{OTP_BOXES}}", otpDigitsHTML,
		"{{OTP_DIGITS}}", cleanOTP,
		"{{OTP_RAW}}", cleanOTP,
		"{{WARNING}}", content.Warning,
		"{{FOOTER_TIME}}", footerTime,
		"{{LOGO_HTML}}", logoHTML,
	)

	return replacer.Replace(rawHTML)
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
			Greeting:    "Hey Sobat ToenToen!",
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
			Greeting:    "Hey Rekan Admin!",
			Description: "Anda telah didaftarkan sebagai <strong>Admin Toko</strong> di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru dan mengaktifkan akun admin Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jangan pernah membagikan kode ini kepada siapapun demi keamanan akses toko Anda.",
		}
	case strings.Contains(subjLower, "admin") && strings.Contains(subjLower, "registrasi"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Registrasi Admin Toko",
			Greeting:    "Hey Rekan Admin!",
			Description: "Terima kasih telah mendaftar sebagai <strong>Admin Toko</strong> di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk menyelesaikan proses pendaftaran akun Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "admin") && (strings.Contains(subjLower, "reset") || strings.Contains(subjLower, "kata sandi")):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Reset Password Admin Toko",
			Greeting:    "Hey Rekan Admin!",
			Description: "Kami menerima permintaan untuk mengatur ulang kata sandi akun Admin Toko Anda di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun toko Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "admin") && (strings.Contains(subjLower, "login") || strings.Contains(subjLower, "masuk")):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Login Admin Toko",
			Greeting:    "Hey Rekan Admin!",
			Description: "Kami mendeteksi aktivitas login ke akun Admin Toko Anda di ToenToen. Masukkan kode verifikasi (OTP) berikut untuk melanjutkan:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jika Anda tidak merasa mencoba login, segera hubungi Owner toko Anda.",
		}
	case strings.Contains(subjLower, "admin") && strings.Contains(subjLower, "email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Email Admin",
			Greeting:    "Hey Rekan Admin!",
			Description: "Gunakan kode verifikasi (OTP) berikut untuk memverifikasi alamat email akun Admin Toko Anda di platform ToenToen:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun toko Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "registrasi"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Pendaftaran Akun Baru",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Terima kasih telah bergabung di platform ToenToen. Gunakan kode verifikasi (OTP) berikut untuk menyelesaikan proses pendaftaran dan verifikasi akun Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	case strings.Contains(subjLower, "login") || strings.Contains(subjLower, "masuk"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Masuk (Login)",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Kami mendeteksi aktivitas login ke akun ToenToen Anda. Masukkan kode verifikasi (OTP) berikut untuk melanjutkan proses masuk:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jika Anda tidak merasa mencoba login, segera amankan akun Anda.",
		}
	case strings.Contains(subjLower, "reset password") || strings.Contains(subjLower, "lupa password") || strings.Contains(subjLower, "kata sandi"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Atur Ulang Kata Sandi",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Kami menerima permintaan untuk mengatur ulang kata sandi akun ToenToen Anda. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jika Anda tidak meminta perubahan kata sandi, abaikan pesan ini dan akun Anda tetap aman.",
		}
	case strings.Contains(subjLower, "penautan email") || strings.Contains(subjLower, "tautkan email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Penautan Email",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Anda sedang menautkan alamat email ini ke akun ToenToen Anda. Masukkan kode verifikasi (OTP) berikut untuk mengonfirmasi penautan email:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jangan berikan kode ini kepada orang lain.",
		}
	case strings.Contains(subjLower, "perubahan email") || strings.Contains(subjLower, "ganti email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Perubahan Alamat Email",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Anda mengajukan permohonan perubahan alamat email akun ToenToen. Masukkan kode verifikasi (OTP) berikut untuk mengonfirmasi bahwa email baru ini milik Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Jangan berikan kode ini kepada orang lain.",
		}
	case strings.Contains(subjLower, "penghapusan akun") || strings.Contains(subjLower, "delete"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "⚠️ Konfirmasi Hapus Akun",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Kami menerima permintaan untuk menghapus akun ToenToen Anda secara permanen. Jika ini benar Anda, masukkan kode verifikasi (OTP) berikut untuk mengonfirmasi:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br><strong>PERHATIAN:</strong> Tindakan penghapusan akun bersifat permanen dan data Anda tidak dapat dipulihkan.",
		}
	case strings.Contains(subjLower, "pengalihan bisnis") || strings.Contains(subjLower, "transfer"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Otorisasi Pengalihan Toko",
			Greeting:    "Hey Pemilik Toko!",
			Description: "Anda sedang mengajukan otorisasi pengalihan kepemilikan bisnis di ToenToen. Masukkan kode verifikasi (OTP) berikut untuk mengotorisasi permohonan transfer toko Anda:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Pastikan Anda hanya memberikan otorisasi transfer kepada pihak yang sah.",
		}
	case strings.Contains(subjLower, "email"):
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Alamat Email",
			Greeting:    "Hey Sobat ToenToen!",
			Description: "Gunakan kode verifikasi (OTP) berikut untuk memverifikasi alamat email Anda di platform ToenToen:",
			Warning:     "Kode ini hanya berlaku selama <strong>5 menit</strong>.<br>Demi keamanan akun Anda, jangan pernah membagikan kode ini kepada siapapun.",
		}
	default:
		return otpEmailContent{
			Subject:     subject,
			Badge:       "Verifikasi Keamanan",
			Greeting:    "Hey Sobat ToenToen!",
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

