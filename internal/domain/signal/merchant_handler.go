package signal

import (
	"strconv"
	"strings"
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

// ExploreSignals lists active signals filtered by merchant location and categories.
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

// GetSignalDetail retrieves signal details and merchant conversation status.
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

// ReplyToSignal initiates a new conversation from merchant.
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
func (h *MerchantHandler) GetMessages(c *fiber.Ctx) error {
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

// SendMessage posts merchant reply.
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

// GetProductSuggestions returns store catalog recommendations matching a signal.
func (h *MerchantHandler) GetProductSuggestions(c *fiber.Ctx) error {
	accID, _, err := getSession(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}
	_ = accID

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	businessID := c.Query("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Query parameter 'business_id' wajib diisi", nil)
	}
	if _, errExplore := h.service.ExploreSignals(accID, businessID, 1, 1); errExplore != nil && strings.Contains(errExplore.Error(), "akses ditolak") {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Anda tidak memiliki akses ke bisnis ini", nil)
	}

	products, errSvc := h.service.GetProductSuggestions(businessID, signalID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil rekomendasi produk katalog", products)
}
