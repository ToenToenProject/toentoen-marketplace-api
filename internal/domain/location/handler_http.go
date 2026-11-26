package location

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// HTTPLocationHandler serves REST endpoints for live location queries.
type HTTPLocationHandler struct {
	service *LocationService
}

// NewHTTPLocationHandler membuat instance baru
func NewHTTPLocationHandler(service *LocationService) *HTTPLocationHandler {
	return &HTTPLocationHandler{service: service}
}

// GetNearbyMerchants returns nearest merchants from customer location.
func (h *HTTPLocationHandler) GetNearbyMerchants(c *fiber.Ctx) error {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	if latStr == "" || lngStr == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Parameter lat dan lng wajib diisi", nil)
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Latitude tidak valid (harus antara -90 dan 90)", nil)
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Longitude tidak valid (harus antara -180 dan 180)", nil)
	}

	radiusKm := 5.0
	if r := c.Query("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 && parsed <= 50 {
			radiusKm = parsed
		}
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	merchants, err := h.service.GetNearbyMerchants(lat, lng, radiusKm, limit)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data merchant terdekat", nil)
	}

	if merchants == nil {
		merchants = []NearbyMerchant{}
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil merchant terdekat", fiber.Map{
		"merchants": merchants,
		"total":     len(merchants),
	})
}

// TrackMerchant mengambil posisi terkini satu merchant tertentu
func (h *HTTPLocationHandler) TrackMerchant(c *fiber.Ctx) error {
	merchantID := c.Params("merchant_id")
	if merchantID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Parameter merchant_id wajib diisi", nil)
	}

	merchant, err := h.service.GetMerchantPosition(merchantID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Merchant tidak ditemukan atau tidak sedang aktif", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil posisi merchant", fiber.Map{
		"merchant": merchant,
	})
}

// UpdateLocationHTTP is a fallback REST endpoint when WebSocket disconnects.
func (h *HTTPLocationHandler) UpdateLocationHTTP(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var update LocationUpdate
	if err := c.BodyParser(&update); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format body tidak valid", nil)
	}

	if update.Lat < -90 || update.Lat > 90 || update.Lng < -180 || update.Lng > 180 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Koordinat tidak valid", nil)
	}

	merchants, err := h.service.GetNearbyMerchants(update.Lat, update.Lng, 5, 50)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data merchant terdekat", nil)
	}

	if merchants == nil {
		merchants = []NearbyMerchant{}
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil merchant terdekat", fiber.Map{
		"merchants": merchants,
		"total":     len(merchants),
	})
}
