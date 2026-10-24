package profile

import (
	"strings"
	"toentoen-backend/internal/constants"

	"github.com/gofiber/fiber/v2"
)

type PublicProfileHandler struct {
	service PublicProfileService
}

func NewPublicProfileHandler(service PublicProfileService) *PublicProfileHandler {
	return &PublicProfileHandler{service}
}

func (h *PublicProfileHandler) GetPublicProfileSummary(c *fiber.Ctx) error {
	accountID := strings.TrimSpace(c.Params("account_id"))
	if accountID == "" {
		accountID = strings.TrimSpace(c.Params("accountId"))
	}

	summary, err := h.service.GetPublicProfileSummary(accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":       "error",
			"responseCode": constants.FailedValidation,
			"message":      err.Error(),
			"data":         nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"responseCode": constants.Success,
		"message":      "Ringkasan profil publik berhasil diambil",
		"data":         summary,
	})
}
