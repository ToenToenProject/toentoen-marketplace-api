package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"os"
	"strings"

	"toentoen-backend/internal/config"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct{}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// VerifyWhatsAppWebhook handles the GET verification handshake from Meta Webhooks API,
// validating the verify token and echoing hub.challenge with a 200 OK response.
func (h *WebhookHandler) VerifyWhatsAppWebhook(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	expectedToken := ""
	if config.Envs != nil {
		expectedToken = strings.TrimSpace(config.Envs.WhatsAppVerifyToken)
	}

	if expectedToken == "" {
		log.Println("[WHATSAPP-WEBHOOK-ERROR] WHATSAPP_VERIFY_TOKEN is not configured")
		return c.Status(fiber.StatusServiceUnavailable).SendString("Webhook verification not configured")
	}

	if mode == "subscribe" && subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1 {
		log.Println("[WHATSAPP-WEBHOOK] Webhook verified successfully by Meta")
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return c.Status(fiber.StatusOK).SendString(challenge)
	}

	log.Printf("[WHATSAPP-WEBHOOK-ERROR] Verification rejected: mode=%q", mode)
	return c.Status(fiber.StatusForbidden).SendString("Verification token mismatch")
}

// HandleWhatsAppWebhookEvent processes Meta WhatsApp Cloud API webhooks.
func (h *WebhookHandler) HandleWhatsAppWebhookEvent(c *fiber.Ctx) error {
	appSecret := strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET"))
	if appSecret == "" && config.Envs != nil && config.Envs.AppEnv == "production" {
		log.Println("[WHATSAPP-WEBHOOK-ERROR] WHATSAPP_APP_SECRET is not configured")
		return c.Status(fiber.StatusUnauthorized).SendString("Webhook signature verification not configured")
	}

	body := c.Body()
	if appSecret != "" {
		sigHeader := strings.TrimSpace(c.Get("X-Hub-Signature-256"))
		if !strings.HasPrefix(sigHeader, "sha256=") {
			return c.Status(fiber.StatusUnauthorized).SendString("Missing or invalid signature")
		}
		providedSig, err := hex.DecodeString(strings.TrimPrefix(sigHeader, "sha256="))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid signature encoding")
		}
		mac := hmac.New(sha256.New, []byte(appSecret))
		mac.Write(body)
		expectedSig := mac.Sum(nil)
		if !hmac.Equal(providedSig, expectedSig) {
			return c.Status(fiber.StatusUnauthorized).SendString("Signature mismatch")
		}
	}

	log.Printf("[WHATSAPP-WEBHOOK-EVENT] Received verified Meta webhook (%d bytes)", len(body))

	// Return 200 OK immediately to prevent Meta webhook delivery retries.
	return c.Status(fiber.StatusOK).SendString("EVENT_RECEIVED")
}
