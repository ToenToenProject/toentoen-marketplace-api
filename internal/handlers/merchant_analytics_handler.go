package handlers

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"strings"
)

type AnalyticsHandler struct {
	service services.AnalyticsService
}

func NewAnalyticsHandler(s services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{s}
}

func (h *AnalyticsHandler) GetDailyAnalytics(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	
	requesterID, role, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	data, errSvc := h.service.GetDailyAnalytics(requesterID, role, businessID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daily analytics retrieved", data)
}

func (h *AnalyticsHandler) GetHistoryAnalytics(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	
	// Tangkap query params (format harus YYYY-MM-DD)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "start_date and end_date are required (Format: YYYY-MM-DD)", nil)
	}

	requesterID, role, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	data, errSvc := h.service.GetHistoryAnalytics(requesterID, role, businessID, startDate, endDate)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "History analytics retrieved", data)
}

func (h *AnalyticsHandler) GetExploreAnalytics(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	keyword := strings.TrimSpace(c.Query("keyword")) // Bisa kosong jika user tidak mencari apa-apa

	requesterID, role, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	data, errSvc := h.service.GetExploreAnalytics(requesterID, role, businessID, keyword)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Explore analytics retrieved", data)
}