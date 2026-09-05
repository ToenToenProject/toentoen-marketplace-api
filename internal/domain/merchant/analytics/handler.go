package analytics

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"strings"
)

type AnalyticsHandler struct {
	service AnalyticsService
}

func NewAnalyticsHandler(s AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{s}
}

func (h *AnalyticsHandler) GetDailyAnalytics(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	
	requesterID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	data, errSvc := h.service.GetDailyAnalytics(requesterID, role, businessID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data analitik harian berhasil dimuat", data)
}

func (h *AnalyticsHandler) GetHistoryAnalytics(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	
	// Tangkap query params (format harus YYYY-MM-DD)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "start_date dan end_date wajib diisi (Format: YYYY-MM-DD)", nil)
	}

	requesterID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	data, errSvc := h.service.GetHistoryAnalytics(requesterID, role, businessID, startDate, endDate)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data riwayat analitik berhasil dimuat", data)
}

func (h *AnalyticsHandler) GetExploreAnalytics(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	keyword := strings.TrimSpace(c.Query("keyword")) // Bisa kosong jika user tidak mencari apa-apa

	requesterID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	data, errSvc := h.service.GetExploreAnalytics(requesterID, role, businessID, keyword)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data analitik pencarian berhasil dimuat", data)
}

// TrackImpressions menerima batch impression dari Flutter (fire-and-forget, selalu 200 OK)
func (h *AnalyticsHandler) TrackImpressions(c *fiber.Ctx) error {
	var body struct {
		Impressions []ImpressionItem `json:"impressions"`
	}
	if err := c.BodyParser(&body); err != nil || len(body.Impressions) == 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'impressions' wajib diisi", nil)
	}

	// Jalankan async agar response langsung cepat
	go h.service.TrackImpressions(body.Impressions)

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Impressions recorded", nil)
}