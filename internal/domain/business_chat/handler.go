package business_chat

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type BusinessChatHandler struct {
	service BusinessChatService
}

func NewBusinessChatHandler(service BusinessChatService) *BusinessChatHandler {
	return &BusinessChatHandler{service: service}
}

// InitiateChat memulai atau mengambil percakapan dengan toko
func (h *BusinessChatHandler) InitiateChat(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "business_id wajib diisi", nil)
	}

	var input models.InitiateBusinessChatInput
	_ = c.BodyParser(&input)

	conv, firstMsg, isNew, errSvc := h.service.InitiateChat(accountID, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	msg := "Percakapan toko ditemukan"
	if isNew {
		msg = "Percakapan toko baru berhasil diinisiasi"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, fiber.Map{
		"is_new_conversation": isNew,
		"conversation_id":     conv.ConversationID,
		"conversation":        conv,
		"first_message":       firstMsg,
	})
}

// GetCustomerConversations mengambil daftar percakapan toko milik customer
func (h *BusinessChatHandler) GetCustomerConversations(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	limit := c.QueryInt("limit", 50)
	cursor := strings.TrimSpace(c.Query("cursor", ""))

	convs, errSvc := h.service.GetCustomerConversations(accountID, limit, cursor)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar percakapan toko berhasil dimuat", convs)
}

// GetMerchantConversations mengambil daftar percakapan pelanggan masuk ke toko
func (h *BusinessChatHandler) GetMerchantConversations(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "business_id wajib diisi", nil)
	}

	limit := c.QueryInt("limit", 50)
	cursor := strings.TrimSpace(c.Query("cursor", ""))

	convs, errSvc := h.service.GetMerchantConversations(businessID, accountID, limit, cursor)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar chat pelanggan berhasil dimuat", convs)
}

// GetMessages mengambil riwayat pesan dalam percakapan toko
func (h *BusinessChatHandler) GetMessages(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	conversationID := strings.TrimSpace(c.Params("conversation_id"))
	if conversationID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "conversation_id wajib diisi", nil)
	}

	limit := c.QueryInt("limit", 50)
	cursor := strings.TrimSpace(c.Query("cursor", ""))

	messages, errSvc := h.service.GetMessages(conversationID, accountID, limit, cursor)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pesan berhasil dimuat", messages)
}

// SendMessage mengirim pesan baru ke dalam percakapan toko
func (h *BusinessChatHandler) SendMessage(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	conversationID := strings.TrimSpace(c.Params("conversation_id"))
	if conversationID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "conversation_id wajib diisi", nil)
	}

	var input models.SendBusinessChatMessageInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	message, errSvc := h.service.SendMessage(conversationID, accountID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pesan berhasil dikirim", message)
}
