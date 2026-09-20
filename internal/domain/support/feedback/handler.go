package feedback

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type FeedbackHandler struct {
	service FeedbackService
}

func NewFeedbackHandler(service FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{service: service}
}

// SendFeedback mengirimkan pesan feedback dari pengguna ke email resmi ToenToen
func (h *FeedbackHandler) SendFeedback(c *fiber.Ctx) error {
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
