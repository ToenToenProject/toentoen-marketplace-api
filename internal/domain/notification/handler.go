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
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Tidak ada sesi (Unauthorized)", nil)
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
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Tidak ada sesi (Unauthorized)", nil)
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
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Tidak ada sesi (Unauthorized)", nil)
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
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Tidak ada sesi (Unauthorized)", nil)
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
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Tidak ada sesi (Unauthorized)", nil)
	}
	accountID := accountIDValue.(string)
	notifID := c.Params("id")

	if notifID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Notification ID wajib diisi", nil)
	}

	if err := h.service.DeleteNotification(accountID, notifID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Notifikasi berhasil dihapus", nil)
}

func (h *NotificationHandler) DeleteAllNotifications(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Tidak ada sesi (Unauthorized)", nil)
	}
	accountID := accountIDValue.(string)

	if err := h.service.DeleteAllNotifications(accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Semua notifikasi berhasil dihapus", nil)
}
