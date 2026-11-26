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

	if mode == "subscribe" && token != "" && (expectedToken == "" || token == expectedToken) {
		log.Printf("[WHATSAPP-WEBHOOK] Webhook verified successfully by Meta! Challenge: %s", challenge)
		return c.Status(fiber.StatusOK).SendString(challenge)
	}

	log.Printf("[WHATSAPP-WEBHOOK-ERROR] Verification rejected: mode=%s, token=%s", mode, token)
	return c.Status(fiber.StatusForbidden).SendString("Verification token mismatch")
}

// HandleWhatsAppWebhookEvent processes Meta WhatsApp Cloud API webhooks.
func (h *WebhookHandler) HandleWhatsAppWebhookEvent(c *fiber.Ctx) error {
	body := c.Body()
	log.Printf("[WHATSAPP-WEBHOOK-EVENT] Received notification from Meta: %s", string(body))

	// Return 200 OK immediately to prevent Meta webhook delivery retries.
	return c.Status(fiber.StatusOK).SendString("EVENT_RECEIVED")
}
