package signal

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type CustomerHandler struct {
	service SignalService
}

func NewCustomerHandler(service SignalService) *CustomerHandler {
	return &CustomerHandler{service}
}

// CreateSignal membuat signal baru dari customer
// POST /api/v1/customer/signals
func (h *CustomerHandler) CreateSignal(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.CreateSignalInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	signal, errSvc := h.service.CreateSignal(accID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Signal berhasil dibuat", fiber.Map{
		"signal_id":    signal.SignalID,
		"category_id":  signal.CategoryID,
		"category":     signal.Category,
		"category_ids": signal.CategoryIDs,
		"categories":   signal.Categories,
	})
}

// GetMySignals mengambil daftar signal milik customer
// GET /api/v1/customer/signals/me
func (h *CustomerHandler) GetMySignals(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signals, errSvc := h.service.GetMySignals(accID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data signal", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar signal", signals)
}

// GetMyConversations mengambil semua roomchat percakapan milik customer
// GET /api/v1/customer/conversations
func (h *CustomerHandler) GetMyConversations(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	conversations, errSvc := h.service.GetCustomerConversations(accID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil daftar percakapan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar percakapan", conversations)
}

// GetSignalDetail mengambil detail signal beserta daftar merchant yang merespons
// GET /api/v1/customer/signals/:signal_id
func (h *CustomerHandler) GetSignalDetail(c *fiber.Ctx) error {
	accID, role, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	signal, errSvc := h.service.GetSignalDetail(signalID, accID, role)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil detail signal", signal)
}

// UpdateSignal memperbarui informasi signal (deskripsi, kategori, tags, lokasi, radius, dll)
// PUT /api/v1/customer/signals/:signal_id
func (h *CustomerHandler) UpdateSignal(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	var input models.UpdateSignalInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	updatedSignal, errSvc := h.service.UpdateSignal(accID, signalID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Signal berhasil diperbarui", updatedSignal)
}

// DeleteSignal membatalkan / menghapus signal milik customer
// DELETE /api/v1/customer/signals/:signal_id
func (h *CustomerHandler) DeleteSignal(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	if errSvc := h.service.DeleteSignal(accID, signalID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Signal berhasil dihapus/dibatalkan", nil)
}

// CompleteSignal menyelesaikan signal (status → completed, semua conversation → locked)
// PUT /api/v1/customer/signals/:signal_id/complete
func (h *CustomerHandler) CompleteSignal(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	if errSvc := h.service.CompleteSignal(accID, signalID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Signal berhasil diselesaikan. Semua percakapan telah ditutup", nil)
}

// GetMessages membaca isi chat dalam conversation
// GET /api/v1/customer/conversations/:conversation_id/messages
func (h *CustomerHandler) GetMessages(c *fiber.Ctx) error {
	accID, role, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	convID := c.Params("conversation_id")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	messages, total, signalRef, errSvc := h.service.GetMessages(accID, role, convID, page, limit)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	respData := fiber.Map{
		"messages": messages,
		"total":    total,
		"page":     page,
		"limit":    limit,
	}
	if signalRef != nil {
		respData["signal_post_reference"] = signalRef
		respData["signal_id"] = signalRef.SignalID
		respData["signal_title"] = signalRef.Title
		respData["signal_description"] = signalRef.Description
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil pesan", respData)
}

// SendMessage mengirim pesan balasan dari customer
// POST /api/v1/customer/conversations/:conversation_id/messages
func (h *CustomerHandler) SendMessage(c *fiber.Ctx) error {
	accID, role, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	convID := c.Params("conversation_id")

	var input models.SendMessageInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	msg, errSvc := h.service.SendMessage(accID, role, convID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Pesan berhasil dikirim", msg)
}

// ================================================================
// HELPER
// ================================================================

func getSession(c *fiber.Ctx) (string, string, error) {
	accID := c.Locals("account_id")
	role := c.Locals("role")
	if accID == nil || role == nil {
		return "", "", fiber.ErrUnauthorized
	}
	return accID.(string), role.(string), nil
}
