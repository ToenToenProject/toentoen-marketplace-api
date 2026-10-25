package chat

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type SupportChatHandler struct {
	service SupportChatService
}

func NewSupportChatHandler(service SupportChatService) *SupportChatHandler {
	return &SupportChatHandler{service: service}
}

func getSessionIDParam(c *fiber.Ctx) string {
	sessionID := strings.TrimSpace(c.Params("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.Params("sessionId"))
	}
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.Params("room_id"))
	}
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.Params("roomId"))
	}
	return sessionID
}

// InitiateChat memulai atau membuka kembali sesi live chat CS aktif
func (h *SupportChatHandler) InitiateChat(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.InitiateCSChatInput
	_ = c.BodyParser(&input)

	session, greetingMsg, isNew, errSvc := h.service.InitiateChat(accountID, role, input.BusinessID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	message := "Sesi live chat aktif ditemukan"
	if isNew {
		message = "Sesi live chat baru berhasil dibuat"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{
		"session":          session,
		"session_id":       session.SessionID,
		"greeting_message": greetingMsg,
		"is_new_session":   isNew,
	})
}

// GetActiveChat mengambil sesi live chat yang sedang aktif untuk user yang login
func (h *SupportChatHandler) GetActiveChat(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	session, errSvc := h.service.GetActiveChat(accountID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	if session == nil {
		return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Tidak ada sesi live chat aktif", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Sesi live chat aktif ditemukan", session)
}

// GetMessages mengambil riwayat pesan dalam sesi live chat
func (h *SupportChatHandler) GetMessages(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	sessionID := getSessionIDParam(c)
	if sessionID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "session_id wajib diisi", nil)
	}

	limit := c.QueryInt("limit", 50)
	cursor := strings.TrimSpace(c.Query("cursor", ""))
	isCS := role == "customer_service" || role == "cs" || role == "super_admin"

	messages, errSvc := h.service.GetChatMessages(sessionID, accountID, isCS, limit, cursor)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pesan berhasil dimuat", messages)
}

// SendMessage mengirim pesan baru ke dalam sesi live chat
func (h *SupportChatHandler) SendMessage(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	sessionID := getSessionIDParam(c)
	if sessionID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "session_id wajib diisi", nil)
	}

	var input models.SendCSMessageInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	senderType := "user"
	senderName := "Pengguna ToenToen"
	isCS := role == "customer_service" || role == "cs" || role == "super_admin"

	if isCS {
		senderType = "cs"
		senderName = "Customer Service ToenToen"
	}

	message, errSvc := h.service.SendMessage(sessionID, accountID, senderType, senderName, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pesan berhasil terkirim", message)
}

// EndChat mengakhiri sesi live chat dan menyimpan rating/feedback
func (h *SupportChatHandler) EndChat(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	sessionID := getSessionIDParam(c)
	if sessionID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "session_id wajib diisi", nil)
	}

	var input models.EndCSChatInput
	_ = c.BodyParser(&input)

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	isCS := role == "customer_service" || role == "cs" || role == "super_admin"
	if errSvc := h.service.EndChat(sessionID, accountID, isCS, input); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Sesi live chat telah berhasil diakhiri. Terima kasih atas masukan Anda!", nil)
}

// MarkAsRead menandai pesan dalam sesi live chat sebagai telah dibaca
// POST /api/v1/support/chat/:session_id/read atau /api/v1/support/chat/:sessionId/read
func (h *SupportChatHandler) MarkAsRead(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	sessionID := getSessionIDParam(c)
	if sessionID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "session_id wajib diisi", nil)
	}

	isCS := role == "customer_service" || role == "cs" || role == "super_admin"
	if errSvc := h.service.MarkAsRead(sessionID, accountID, isCS); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pesan telah ditandai sebagai dibaca", fiber.Map{
		"session_id":     sessionID,
		"is_read":        true,
		"is_new_session": false,
	})
}
