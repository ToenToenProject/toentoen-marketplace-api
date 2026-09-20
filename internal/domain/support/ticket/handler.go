package ticket

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type SupportTicketHandler struct {
	service SupportTicketService
}

func NewSupportTicketHandler(service SupportTicketService) *SupportTicketHandler {
	return &SupportTicketHandler{service: service}
}

// CreateTicket membuat tiket bantuan baru
func (h *SupportTicketHandler) CreateTicket(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.CreateTicketInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	ticketRes, errSvc := h.service.CreateTicket(accountID, role, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Tiket bantuan berhasil dikirim", ticketRes)
}

// GetMyTickets mengambil daftar seluruh tiket yang pernah dikirim oleh pengguna
func (h *SupportTicketHandler) GetMyTickets(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	limit := c.QueryInt("limit", 50)
	cursor := strings.TrimSpace(c.Query("cursor", ""))

	tickets, errSvc := h.service.GetMyTickets(accountID, limit, cursor)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar tiket bantuan berhasil dimuat", tickets)
}

// GetTicketDetail mengambil detail informasi tiket beserta lampiran dan balasan CS
func (h *SupportTicketHandler) GetTicketDetail(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	ticketID := strings.TrimSpace(c.Params("ticket_id"))
	if ticketID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ticket_id wajib diisi", nil)
	}

	isCS := role == "customer_service" || role == "cs" || role == "super_admin"

	detail, errSvc := h.service.GetTicketDetail(ticketID, accountID, isCS)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail tiket berhasil dimuat", detail)
}

// SendFeedback mengirimkan feedback dari pengguna ke email resmi ToenToen
func (h *SupportTicketHandler) SendFeedback(c *fiber.Ctx) error {
	accountID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.CreateFeedbackInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if errSvc := h.service.SendFeedback(accountID, role, input); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pesan feedback berhasil dikirimkan ke tim ToenToen. Terima kasih atas masukan Anda!", nil)
}


