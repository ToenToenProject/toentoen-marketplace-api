package report

import (
	"github.com/gofiber/fiber/v2"
	"toentoen-backend/pkg/utils"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// SubmitReport Handler untuk mengirimkan laporan bisnis
func (h *Handler) SubmitReport(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Business ID is required", "D1", nil)
	}

	// Wajib login
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, "Unauthorized", "A1", nil)
	}

	var req SubmitReportRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Invalid request payload", "D1", nil)
	}

	err := h.service.SubmitReport(accountID, businessID, req)
	if err != nil {
		if err.Error() == "business not found" {
			return utils.SendResponse(c, fiber.StatusNotFound, "Business not found", "D1", nil)
		}
		if err.Error() == "reason is required" || err.Error() == "description is required when reason is 'Other'" {
			return utils.SendResponse(c, fiber.StatusBadRequest, err.Error(), "D1", nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, "Report submitted successfully", "00", nil)
}
