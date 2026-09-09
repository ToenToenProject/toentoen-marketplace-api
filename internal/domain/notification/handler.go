package notification

import (
	"strconv"
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type NotificationHandler struct {
	service NotificationService
}

func NewNotificationHandler(service NotificationService) *NotificationHandler {
	return &NotificationHandler{service}
}

func (h *NotificationHandler) GetMyNotifications(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	accountID := accountIDValue.(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	notifs, total, err := h.service.GetMyNotifications(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil daftar notifikasi", nil)
	}

	unreadCount, _ := h.service.GetUnreadCount(accountID)
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar notifikasi", fiber.Map{
		"notifications": notifs,
		"unread_count":  unreadCount,
		"meta": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total_rows":  total,
			"total_pages": totalPages,
		},
	})
}

func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	accountID := accountIDValue.(string)

	count, err := h.service.GetUnreadCount(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal menghitung notifikasi belum dibaca", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil jumlah notifikasi belum dibaca", fiber.Map{
		"unread_count": count,
	})
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	accountID := accountIDValue.(string)
	notifID := c.Params("id")

	if err := h.service.MarkAsRead(accountID, notifID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Notifikasi ditandai sudah dibaca", nil)
}

func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	accountID := accountIDValue.(string)

	if err := h.service.MarkAllAsRead(accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Semua notifikasi ditandai sudah dibaca", nil)
}

func (h *NotificationHandler) DeleteNotification(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	accountID := accountIDValue.(string)
	notifID := c.Params("id")

	if notifID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID notifikasi wajib diisi", nil)
	}

	if err := h.service.DeleteNotification(accountID, notifID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Notifikasi berhasil dihapus", nil)
}

func (h *NotificationHandler) DeleteAllNotifications(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	accountID := accountIDValue.(string)

	if err := h.service.DeleteAllNotifications(accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Semua notifikasi berhasil dihapus", nil)
}

// InternalNotificationInput payload untuk endpoint internal antar-service
type InternalNotificationInput struct {
	RecipientIDs   []string               `json:"recipient_ids"`
	TriggerActorID string                 `json:"trigger_actor_id"`
	Type           string                 `json:"type"`
	ReferenceID    string                 `json:"reference_id"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	Data           map[string]interface{} `json:"data"`
}

// SendInternalNotification memproses pengiriman notifikasi dari internal service lain (misal: Operations Platform / KYC Review)
// Endpoint ini langsung memicu penyimpanan DB, broadcast WebSocket, dan FCM Push Notification
func (h *NotificationHandler) SendInternalNotification(c *fiber.Ctx) error {
	var input InternalNotificationInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.InvalidParam, "Format payload tidak valid", nil)
	}

	if len(input.RecipientIDs) == 0 || input.Type == "" || input.Title == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "recipient_ids, type, dan title wajib diisi", nil)
	}

	go h.service.SendNotification(
		input.RecipientIDs,
		input.TriggerActorID,
		input.Type,
		input.ReferenceID,
		input.Title,
		input.Body,
		input.Data,
		nil,
	)

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Notifikasi internal berhasil diproses dan dikirimkan", nil)
}

