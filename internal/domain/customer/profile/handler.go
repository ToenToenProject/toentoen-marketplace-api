package profile

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PublicProfileHandler struct {
	service PublicProfileService
}

func NewPublicProfileHandler(service PublicProfileService) *PublicProfileHandler {
	return &PublicProfileHandler{service}
}

func (h *PublicProfileHandler) GetPublicProfileSummary(c *fiber.Ctx) error {
	accountID := c.Params("account_id")

	summary, err := h.service.GetPublicProfileSummary(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ringkasan profil publik berhasil diambil", summary)
}
