package signal

import (
	"strconv"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type MerchantHandler struct {
	service SignalService
}

func NewMerchantHandler(service SignalService) *MerchantHandler {
	return &MerchantHandler{service}
}

// ExploreSignals menampilkan feed signal aktif berdasarkan lokasi merchant (PostGIS)
// GET /api/v1/merchant/signals/explore?lat=-6.2&lng=106.8&business_id=BUS-xxx
func (h *MerchantHandler) ExploreSignals(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := c.Query("business_id")

	var lat, lng float64
	if latStr := c.Query("lat"); latStr != "" {
		lat, _ = strconv.ParseFloat(latStr, 64)
	}
	if lngStr := c.Query("lng"); lngStr != "" {
		lng, _ = strconv.ParseFloat(lngStr, 64)
	}

	signals, errSvc := h.service.ExploreSignals(accID, businessID, lat, lng)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil feed sinyal aktif", signals)
}

// GetSignalDetail mengambil detail signal yang dipilih merchant (beserta status roomchat merchant)
// GET /api/v1/merchant/signals/:signal_id
func (h *MerchantHandler) GetSignalDetail(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	signal, errSvc := h.service.GetMerchantSignalDetail(signalID, accID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil detail signal", signal)
}

// GetMyConversations mengambil semua roomchat percakapan milik merchant
// GET /api/v1/merchant/conversations
func (h *MerchantHandler) GetMyConversations(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	conversations, errSvc := h.service.GetMerchantConversations(accID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil daftar percakapan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar percakapan", conversations)
}

// ReplyToSignal memulai conversation baru dari merchant
// POST /api/v1/merchant/signals/:signal_id/reply
func (h *MerchantHandler) ReplyToSignal(c *fiber.Ctx) error {
	accID, role, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	var input models.InitiateConversationInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if input.BusinessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'business_id' wajib diisi", nil)
	}

	conv, errSvc := h.service.ReplyToSignal(accID, role, signalID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Berhasil memulai percakapan dengan customer", fiber.Map{
		"conversation_id": conv.ConversationID,
	})
}

// GetMessages membaca isi chat dalam conversation
// GET /api/v1/merchant/conversations/:conversation_id/messages
func (h *MerchantHandler) GetMessages(c *fiber.Ctx) error {
	accID, role, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	convID := c.Params("conversation_id")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	messages, total, errSvc := h.service.GetMessages(accID, role, convID, page, limit)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil pesan", fiber.Map{
		"messages": messages,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// SendMessage mengirim pesan balasan dari merchant
// POST /api/v1/merchant/conversations/:conversation_id/messages
func (h *MerchantHandler) SendMessage(c *fiber.Ctx) error {
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
