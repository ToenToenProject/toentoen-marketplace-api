package config

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var FCMClient *messaging.Client

func InitFirebase() {
	// Firebase SDK terbaru lebih stabil jika kita membiarkannya membaca ProjectID 
	// langsung dari file JSON secara otomatis.
	opt := option.WithCredentialsFile("pkg/config/toentoen-app-firebase-adminsdk-fbsvc-894ab133f2.json")
	
	// Gunakan nil untuk config agar SDK membaca dari file kredensial
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("⚠️ Warning initializing firebase app: %v (Push notifications disabled)\n", err)
		return
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("⚠️ Warning getting Messaging client: %v (Push notifications disabled)\n", err)
		return
	}

	FCMClient = client
	fmt.Println("✅ Firebase Cloud Messaging berhasil dikoneksikan!")
}

// SendPushNotification diperbarui untuk menghindari error 404 /batch
func SendPushNotification(tokens []string, title, body string, data map[string]string) error {
	if FCMClient == nil || len(tokens) == 0 {
		return nil
	}

	successCount := 0
	for _, token := range tokens {
		notif := &messaging.Notification{
			Title: title,
			Body:  body,
		}
		if img, ok := data["image_url"]; ok && img != "" {
			notif.ImageURL = img
		}

		message := &messaging.Message{
			Token:        token,
			Notification: notif,
			Data:         data,
		}

		// Kirim ke endpoint v1 (/messages:send)
		_, err := FCMClient.Send(context.Background(), message)
		if err != nil {
			// Kita log tapi tidak return error agar token lain tetap terkirim
			log.Printf("⚠️ Gagal mengirim ke token %s: %v\n", token, err)
			continue
		}
		successCount++
	}

	log.Printf("✅ Proses selesai. Sukses kirim ke %d dari %d perangkat\n", successCount, len(tokens))
	return nil
}