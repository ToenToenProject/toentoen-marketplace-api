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

	isCS := role == "platform_admin" || role == "customer_service" || role == "cs" || role == "super_admin"

	detail, errSvc := h.service.GetTicketDetail(ticketID, accountID, isCS)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail tiket berhasil dimuat", detail)
}

// AdminGetTickets mengambil antrian tiket masuk untuk dashboard Customer Service
func (h *SupportTicketHandler) AdminGetTickets(c *fiber.Ctx) error {
	status := strings.TrimSpace(c.Query("status", ""))
	limit := c.QueryInt("limit", 50)
	cursor := strings.TrimSpace(c.Query("cursor", ""))

	tickets, errSvc := h.service.AdminGetTickets(status, limit, cursor)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar tiket admin berhasil dimuat", tickets)
}

// AdminResolveTicket memberikan balasan dan memperbarui status tiket
func (h *SupportTicketHandler) AdminResolveTicket(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	ticketID := strings.TrimSpace(c.Params("ticket_id"))
	if ticketID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ticket_id wajib diisi", nil)
	}

	var input models.ResolveTicketInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	res, errSvc := h.service.AdminResolveTicket(ticketID, input, accountID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Tiket berhasil ditanggapi dan diperbarui", res)
}
