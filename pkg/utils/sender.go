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
	plainBody := fmt.Sprintf("%s\n\n%s\n\nKode verifikasi (OTP) Anda adalah: %s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.\n\nJika Anda tidak merasa melakukan permintaan verifikasi ini, silakan abaikan pesan ini.\n\nSalam,\nTim ToenToen",
		content.Greeting, stripBasicHTML(content.Description), otp)
	mailer.SetBody("text/plain", plainBody)

	// HTML body (modern card design dengan tema ToenToen)
	htmlBody := buildOTPEmailHTML(content, otp, time.Now())
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

func renderOTPDigitsHTML(otp string) string {
	var sb strings.Builder
	for _, ch := range strings.TrimSpace(otp) {
		sb.WriteString(fmt.Sprintf(
			`<td style="padding-right: 4px;" valign="middle">`+
				`<table cellpadding="0" cellspacing="0" border="0" width="28" height="38" bgcolor="#d4f1f1" style="width: 28px; height: 38px; background-color: #d4f1f1; border-radius: 4px; text-align: center; border-collapse: separate;">`+
				`<tr><td align="center" valign="middle" style="font-family: 'Poppins', Arial, sans-serif; font-size: 19px; font-weight: 700; color: #008f88; line-height: 38px; padding: 0;">%c</td></tr>`+
				`</table>`+
				`</td>`,
			ch,
		))
	}
	return sb.String()
}

func buildOTPEmailHTML(content otpEmailContent, otp string, createdAt time.Time) string {
	otpDigitsHTML := renderOTPDigitsHTML(otp)
	footerTime := formatWIBDate(createdAt)

	rawHTML := `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{SUBJECT}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Poppins:wght@400;500;600;700;800&display=swap" rel="stylesheet">
<style>
  * { box-sizing: border-box; }
  body, table, td, a { -webkit-text-size-adjust: 100%; -ms-text-size-adjust: 100%; }
  table, td { mso-table-lspace: 0pt; mso-table-rspace: 0pt; }
  body {
    background-color: #1e1e1e !important;
    font-family: 'Poppins', -apple-system, BlinkMacSystemFont, Arial, sans-serif;
    margin: 0;
    padding: 0;
  }
  .swoosh-tl { position: absolute; top: 0; left: 0; width: 64px; height: 70px; pointer-events: none; }
  .swoosh-br { position: absolute; bottom: 0; right: 0; width: 96px; height: 108px; pointer-events: none; }
  @media (max-width: 440px) {
    .card-box { padding: 32px 18px 20px !important; }
    .email-wrapper { padding: 16px 8px !important; }
  }
</style>
</head>
<body bgcolor="#1e1e1e" style="background-color: #1e1e1e; font-family: 'Poppins', -apple-system, BlinkMacSystemFont, Arial, sans-serif; margin: 0; padding: 0;">
  <!-- Full-width Dark Container Table (Ensures dark background in Gmail & Outlook) -->
  <table class="email-wrapper" width="100%" border="0" cellspacing="0" cellpadding="0" bgcolor="#1e1e1e" style="background-color: #1e1e1e; width: 100%; min-height: 100vh; padding: 24px 12px; margin: 0;">
    <tr>
      <td align="center" valign="top">
        <!-- Frame width container (423px max-width) -->
        <table width="100%" border="0" cellspacing="0" cellpadding="0" style="max-width: 423px; width: 100%; margin: 0 auto;">
          <!-- Label Email -->
          <tr>
            <td align="left" style="color: #9a9a9a; font-size: 12px; font-family: 'Poppins', Arial, sans-serif; padding-bottom: 12px; font-weight: 400;">
              Email
            </td>
          </tr>
          <!-- Main White Card -->
          <tr>
            <td>
              <table class="card-box" width="100%" border="0" cellspacing="0" cellpadding="0" bgcolor="#ffffff" style="background-color: #ffffff; border-radius: 12px; overflow: hidden; padding: 36px 24px 24px; position: relative; color: #000000; box-shadow: 0 4px 16px rgba(0,0,0,0.18);">
                
                <!-- Decorative Top-Left Swoosh (for clients supporting SVG/relative) -->
                <div class="swoosh-tl" style="position: absolute; top: 0; left: 0; width: 64px; height: 70px; pointer-events: none; z-index: 1;">
                  <svg class="swoosh tl" viewBox="0 0 64 70" fill="none" xmlns="http://www.w3.org/2000/svg" style="width: 64px; height: 70px;">
                    <g stroke="#00B5AD" stroke-width="7" stroke-linecap="butt">
                      <path d="M27 -4 C28 14, 19 22, 10 26 S0 33, -6 35"/>
                      <path d="M41 -4 C41 22, 30 33, 17 40 S3 52, -6 56"/>
                    </g>
                  </svg>
                </div>

                <!-- Header: Two-column layout (Greeting on left, ToenToen circular badge on right) -->
                <tr>
                  <td>
                    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="position: relative; z-index: 2; margin-bottom: 12px;">
                      <tr>
                        <td valign="top" align="left" style="padding-right: 12px;">
                          <div style="font-family: 'Poppins', Arial, sans-serif; font-size: 15.5px; font-weight: 600; line-height: 1.45; color: #000000;">
                            {{GREETING}}
                          </div>
                        </td>
                        <td valign="top" align="right" width="46" style="width: 46px; min-width: 46px;">
                          <!-- ToenToen Brand Circle Badge (Renders 100% in all email clients including Gmail) -->
                          <table cellpadding="0" cellspacing="0" border="0" width="46" height="46" bgcolor="#00B5AD" style="width: 46px; height: 46px; background-color: #00B5AD; border-radius: 50%; text-align: center; border-collapse: separate;">
                            <tr>
                              <td align="center" valign="middle" style="color: #ffffff; font-family: 'Poppins', Arial, sans-serif; font-size: 11px; font-weight: 800; line-height: 12px; letter-spacing: 0.5px; padding: 0;">
                                TO<br>EN
                              </td>
                            </tr>
                          </table>
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>

                <!-- Description Paragraph -->
                <tr>
                  <td style="position: relative; z-index: 2;">
                    <p style="font-family: 'Poppins', Arial, sans-serif; font-size: 12px; line-height: 1.55; font-weight: 400; color: #222222; margin: 0 0 14px 0;">
                      {{DESCRIPTION}}
                    </p>
                  </td>
                </tr>

                <!-- OTP Digits Container (Table Row with 6 separate teal rounded boxes) -->
                <tr>
                  <td style="position: relative; z-index: 2; padding: 4px 0 10px 0;">
                    <table cellpadding="0" cellspacing="0" border="0">
                      <tr>
                        {{OTP_DIGITS}}
                      </tr>
                    </table>
                  </td>
                </tr>

                <!-- Salin Kode Tag -->
                <tr>
                  <td style="position: relative; z-index: 2; padding-bottom: 18px;">
                    <table cellpadding="0" cellspacing="0" border="0" bgcolor="#e6e6e6" style="background-color: #e6e6e6; border-radius: 3px; display: inline-table;">
                      <tr>
                        <td style="padding: 4px 10px; font-family: 'Poppins', Arial, sans-serif; font-size: 10px; font-weight: 600; color: #555555;">
                          Salin Kode: <span style="color: #007A74; margin-left: 5px; letter-spacing: 1.2px; font-size: 11px;">{{OTP_RAW}}</span>
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>

                <!-- Warning Paragraph -->
                <tr>
                  <td style="position: relative; z-index: 2;">
                    <p style="font-family: 'Poppins', Arial, sans-serif; font-size: 12px; line-height: 1.55; font-weight: 400; color: #222222; margin: 0 0 10px 0;">
                      {{WARNING}}
                    </p>
                    <p style="font-family: 'Poppins', Arial, sans-serif; font-size: 12px; line-height: 1.55; font-weight: 400; color: #666666; margin: 0 0 8px 0;">
                      Jika Anda tidak merasa melakukan permintaan verifikasi ini, silakan abaikan pesan ini. Akun Anda tetap aman terlindungi.
                    </p>
                  </td>
                </tr>

                <!-- Footer with timestamp -->
                <tr>
                  <td style="position: relative; z-index: 2; padding-top: 18px;">
                    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="border-top: 1px solid #f0f0f0; padding-top: 12px;">
                      <tr>
                        <td align="center" style="font-family: 'Poppins', Arial, sans-serif; font-size: 10.5px; font-weight: 500; color: #666666; text-align: center;">
                          <span style="font-size: 13px; vertical-align: middle; margin-right: 4px;">⏱</span>
                          <span style="vertical-align: middle;">{{FOOTER_TIME}}</span>
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>

                <!-- Decorative Bottom-Right Swoosh -->
                <div class="swoosh-br" style="position: absolute; bottom: 0; right: 0; width: 96px; height: 108px; pointer-events: none; z-index: 1;">
                  <svg class="swoosh br" viewBox="0 0 80 90" fill="none" xmlns="http://www.w3.org/2000/svg" style="width: 96px; height: 108px;">
                    <g stroke="#00B5AD" stroke-width="6.5" stroke-linecap="butt" transform="rotate(180 40 45)">
                      <path d="M33 -4 C33 14, 24 22, 13 27 S0 35, -6 35"/>
                      <path d="M45 -4 C45 22, 32 32, 18 38 S2 52, -6 55"/>
                      <path d="M57 -4 C57 30, 42 44, 26 52 S6 70, -6 76"/>
                    </g>
                  </svg>
                </div>

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
		"{{GREETING}}", content.Greeting,
		"{{DESCRIPTION}}", content.Description,
		"{{OTP_DIGITS}}", otpDigitsHTML,
		"{{OTP_RAW}}", otp,
		"{{WARNING}}", content.Warning,
		"{{FOOTER_TIME}}", footerTime,
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

