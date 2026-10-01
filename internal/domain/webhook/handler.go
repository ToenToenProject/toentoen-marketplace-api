package webhook

import (
	"log"
	"strings"
	"toentoen-backend/internal/config"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct{}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// VerifyWhatsAppWebhook menangani GET request verifikasi handshake dari Meta for Developers.
// Meta mengirimkan:
//   - hub.mode=subscribe
//   - hub.challenge=<angka_challenge>
//   - hub.verify_token=<token_yang_didaftarkan>
// Server wajib merespon dengan status 200 OK dan body berisi string hub.challenge secara langsung (plain text).
func (h *WebhookHandler) VerifyWhatsAppWebhook(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	expectedToken := ""
	if config.Envs != nil {
		expectedToken = strings.TrimSpace(config.Envs.WhatsAppVerifyToken)
	}

	if mode == "subscribe" && token != "" && (expectedToken == "" || token == expectedToken) {
		log.Printf("[WHATSAPP-WEBHOOK] Webhook verified successfully by Meta! Challenge: %s", challenge)
		return c.Status(fiber.StatusOK).SendString(challenge)
	}

	log.Printf("[WHATSAPP-WEBHOOK-ERROR] Verification rejected: mode=%s, token=%s", mode, token)
	return c.Status(fiber.StatusForbidden).SendString("Verification token mismatch")
}

// HandleWhatsAppWebhookEvent menangani POST event notifikasi dari Meta (status pesan terkirim, dibaca, atau pesan masuk).
func (h *WebhookHandler) HandleWhatsAppWebhookEvent(c *fiber.Ctx) error {
	body := c.Body()
	log.Printf("[WHATSAPP-WEBHOOK-EVENT] Received notification from Meta: %s", string(body))

	// Respon 200 OK sesegera mungkin agar Meta tidak mengirim ulang (retry)
	return c.Status(fiber.StatusOK).SendString("EVENT_RECEIVED")
}
