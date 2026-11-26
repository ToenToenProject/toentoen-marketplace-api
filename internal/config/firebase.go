package config

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var (
	FCMClient          *messaging.Client
	FirebaseAuthClient *auth.Client
)

func InitFirebase() {
	// Let the Firebase SDK auto-detect ProjectID from credentials file.
	opt := option.WithCredentialsFile("pkg/config/toentoen-app-firebase-adminsdk-fbsvc-894ab133f2.json")
	
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("⚠️ Warning initializing firebase app: %v (Push notifications disabled)\n", err)
		return
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("⚠️ Warning getting Messaging client: %v (Push notifications disabled)\n", err)
	} else {
		FCMClient = client
		fmt.Println("✅ Firebase Cloud Messaging berhasil dikoneksikan!")
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		log.Printf("⚠️ Warning getting Firebase Auth client: %v (Phone Auth disabled)\n", err)
	} else {
		FirebaseAuthClient = authClient
		fmt.Println("✅ Firebase Authentication berhasil dikoneksikan!")
	}
}

// VerifyFirebaseIDToken validates a Firebase ID token and returns the decoded token.
func VerifyFirebaseIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	if FirebaseAuthClient == nil {
		return nil, fmt.Errorf("Firebase Auth client belum diinisialisasi")
	}
	return FirebaseAuthClient.VerifyIDToken(ctx, idToken)
}

// SendPushNotification dispatches messages via individual v1 endpoints to avoid batch API deprecation issues.
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

		// Dispatch via v1 endpoint (/messages:send).
		_, err := FCMClient.Send(context.Background(), message)
		if err != nil {
			// Log individual delivery errors without failing remaining recipients.
			log.Printf("⚠️ Gagal mengirim ke token %s: %v\n", token, err)
			continue
		}
		successCount++
	}

	log.Printf("✅ Proses selesai. Sukses kirim ke %d dari %d perangkat\n", successCount, len(tokens))
	return nil
}