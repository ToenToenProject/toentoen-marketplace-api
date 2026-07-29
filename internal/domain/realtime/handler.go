package realtime

import (
	"fmt"
	"log"

	"toentoen-backend/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
)

// WSHandler menangani upgrade HTTP → WebSocket
type WSHandler struct {
	hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// UpgradeCheck adalah middleware Fiber yang memvalidasi bahwa request
// adalah WebSocket upgrade request sebelum meneruskannya.
func UpgradeCheck(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		// Parse JWT dari query parameter
		tokenString := c.Query("token")
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "token query parameter wajib diisi",
			})
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode enkripsi tidak valid")
			}
			return []byte(config.Envs.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token tidak valid atau sudah expired",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Gagal membaca data token",
			})
		}

		// Simpan data user ke Locals agar bisa diakses di handler WS
		c.Locals("account_id", claims["account_id"])
		c.Locals("role", claims["role"])

		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

// Handle menangani koneksi WebSocket yang sudah di-upgrade
func (h *WSHandler) Handle(c *websocket.Conn) {
	accountID, _ := c.Locals("account_id").(string)
	role, _ := c.Locals("role").(string)

	if accountID == "" {
		log.Println("⚠️ WS: account_id kosong, menutup koneksi")
		c.Close()
		return
	}

	client := &Client{
		Hub:       h.hub,
		Conn:      c,
		Send:      make(chan []byte, 256),
		AccountID: accountID,
		Role:      role,
	}

	h.hub.register <- client

	// Jalankan write pump di goroutine, read pump di goroutine saat ini
	go client.WritePump()
	client.ReadPump() // Blocking sampai koneksi terputus
}
