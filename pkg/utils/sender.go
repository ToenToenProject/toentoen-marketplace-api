package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"gopkg.in/gomail.v2"
)

func SendEmailOTP(toEmail string, otp string) {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", os.Getenv("SMTP_EMAIL"))
	mailer.SetHeader("To", toEmail)
	mailer.SetHeader("Subject", "Kode Verifikasi Login ToenToen")

	body := fmt.Sprintf(`
		<h2>Halo!</h2>
		<p>Kode verifikasi (OTP) Anda adalah: <b>%s</b></p>
		<p>Kode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.</p>
	`, otp)
	mailer.SetBody("text/html", body)

	dialer := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		587,
		os.Getenv("SMTP_EMAIL"),
		os.Getenv("SMTP_PASSWORD"),
	)

	err := dialer.DialAndSend(mailer)
	if err != nil {
		fmt.Printf("❌ Gagal mengirim email ke %s: %v\n", toEmail, err)
	} else {
		fmt.Printf("✅ Email OTP berhasil dikirim ke %s\n", toEmail)
	}
}

func generateUniqueOTPMessage(otp string) string {
	greetings := []string{
		"Halo kak!",
		"Selamat datang,",
		"Hai sobat ToenToen,",
		"Salam hangat!",
		"Halo pengguna ToenToen,",
	}

	ctas := []string{
		"Balas 'OK' jika kamu sudah berhasil masuk ya.",
		"Ketik 'SIP' kalau aplikasinya sudah bisa dibuka.",
		"Balas pesan ini jika kamu mengalami kendala login.",
		"Ketik 'DONE' setelah memasukkan kode ini.",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	randomGreeting := greetings[r.Intn(len(greetings))]
	randomCTA := ctas[r.Intn(len(ctas))]

	currentTime := time.Now().Format("02 Jan 2006, 15:04:05")

	msg := fmt.Sprintf("%s\n\nKode verifikasi ToenToen Anda adalah: \n\n*%s*\n\n_Diminta pada: %s_\n_Rahasiakan kode ini dari siapa pun._\n\n%s", randomGreeting, otp, currentTime, randomCTA)

	return msg
}

func SendWAOTP(phoneNumber string, otp string) error {
	message := generateUniqueOTPMessage(otp)

	payload := map[string]interface{}{
		"target":  phoneNumber,
		"message": message,
		"delay":   "1",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.fonnte.com/send", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	fonnteToken := os.Getenv("FONNTE_TOKEN")
	req.Header.Set("Authorization", fonnteToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed sent OTP to WA, Fonnte status: %d", resp.StatusCode)
	}

	return nil
}
