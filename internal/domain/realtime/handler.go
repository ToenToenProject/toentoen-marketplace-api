package realtime

import (
	"fmt"
	"log"
	"strings"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
)

// WSHandler handles HTTP to WebSocket protocol upgrades.
type WSHandler struct {
	hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// UpgradeCheck validates WebSocket upgrade headers before dispatching to connection handler.
func UpgradeCheck(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		// Authenticate WebSocket connection via JWT query parameter.
		tokenString := c.Query("token")
		if tokenString == "" {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "token query parameter wajib diisi", nil)
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode enkripsi tidak valid")
			}
			return []byte(config.Envs.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Token tidak valid atau sudah expired", nil)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Gagal membaca data token", nil)
		}

		accountID, ok := claims["account_id"].(string)
		if !ok || strings.TrimSpace(accountID) == "" {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Token tidak memiliki identitas akun yang valid", nil)
		}
		if config.RDB != nil {
			if jti, ok := claims["jti"].(string); ok && jti != "" {
				if revoked, _ := config.RDB.Exists(config.Ctx, "revoked_jti:"+jti).Result(); revoked > 0 {
					return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi Anda telah dicabut, silakan login kembali", nil)
				}
			}
			if revoked, _ := config.RDB.Exists(config.Ctx, "revoked_account:"+accountID).Result(); revoked > 0 {
				return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi Anda telah dicabut atau akun sedang diblokir", nil)
			}
		}
		var account models.Account
		if err := database.DB.Select("last_login_at, account_status").Where("account_id = ?", accountID).First(&account).Error; err != nil {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akun tidak ditemukan atau telah dihapus", nil)
		}
		if account.AccountStatus == "banned" || account.AccountStatus == "suspended" || (account.AccountStatus != "active" && account.AccountStatus != "inactive") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Akun Anda sedang ditangguhkan atau diblokir", nil)
		}
		if account.LastLoginAt != nil {
			if iatFloat, ok := claims["iat"].(float64); ok {
				if int64(iatFloat) < account.LastLoginAt.Unix() {
					return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi Anda sudah tidak valid, silakan login kembali", nil)
				}
			}
		}

		c.Locals("account_id", claims["account_id"])
		c.Locals("role", claims["role"])

		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

// Handle manages an upgraded WebSocket client lifecycle.
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

	// Run write pump in goroutine; execute read pump on current goroutine.
	go client.WritePump()
	client.ReadPump() // Blocking sampai koneksi terputus
}
