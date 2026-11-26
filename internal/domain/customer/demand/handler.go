package demand

import (
	"strconv"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type DemandHandler struct {
	service DemandService
}

func NewDemandHandler(service DemandService) *DemandHandler {
	return &DemandHandler{service}
}

func getSessionInfo(c *fiber.Ctx) (string, string, error) {
	accID := c.Locals("account_id")
	role := c.Locals("role")
	if accID == nil || role == nil { return "", "", fiber.ErrUnauthorized }
	return accID.(string), role.(string), nil
}

func (h *DemandHandler) CreateSignal(c *fiber.Ctx) error {
	accID, _, err := getSessionInfo(c)
	if err != nil { return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil) }

	var input models.CreateSignalInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	signal, errSvc := h.service.CreateSignal(accID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Demand signal berhasil dibuat", signal)
}

func (h *DemandHandler) ExploreSignals(c *fiber.Ctx) error {
	category := c.Query("category")
	if category == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Parameter category wajib diisi", nil)
	}

	signals, errSvc := h.service.ExploreOpenSignals(category)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data signal", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil data demand signals", signals)
}

func (h *DemandHandler) SubmitResponse(c *fiber.Ctx) error {
	accID, role, err := getSessionInfo(c)
	if err != nil { return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil) }

	signalID := c.Params("signal_id")

	var input models.CreateResponseInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	response, errSvc := h.service.SubmitResponse(accID, role, signalID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Penawaran berhasil dikirim", response)
}

func (h *DemandHandler) AcceptOffer(c *fiber.Ctx) error {
	accID, _, err := getSessionInfo(c)
	if err != nil { return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil) }

	signalID := c.Params("signal_id")
	responseID := c.Params("response_id")

	if errSvc := h.service.AcceptOffer(accID, signalID, responseID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Penawaran berhasil diterima, silakan lanjut ke WhatsApp", nil)
}

func (h *DemandHandler) GetActiveSignals(c *fiber.Ctx) error {
	accID, _, err := getSessionInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := c.Query("business_id")
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	var currentLat, currentLng float64
	if latStr != "" && lngStr != "" {
		parsedLat, errLat := strconv.ParseFloat(latStr, 64)
		parsedLng, errLng := strconv.ParseFloat(lngStr, 64)
		
		if errLat == nil && errLng == nil {
			currentLat = parsedLat
			currentLng = parsedLng
		}
	}

	signals, errSvc := h.service.GetActiveFeed(accID, businessID, currentLat, currentLng)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil feed sinyal aktif untuk owner", signals)
}

func (h *DemandHandler) GetSignalDetail(c *fiber.Ctx) error {
	_, _, err := getSessionInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	signalID := c.Params("signal_id")
	if signalID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Signal ID wajib diisi", nil)
	}

	signal, errSvc := h.service.GetSignalDetail(signalID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil detail sinyal", signal)
}