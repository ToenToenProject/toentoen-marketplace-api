package promotion

import (
	"strings"

	"toentoen-backend/internal/constants"
	merchantpromo "toentoen-backend/internal/domain/merchant/promotion"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type CustomerPromotionHandler struct {
	service merchantpromo.PromotionService
	repo    merchantpromo.PromotionRepository
}

func NewCustomerPromotionHandler(
	service merchantpromo.PromotionService,
	repo merchantpromo.PromotionRepository,
) *CustomerPromotionHandler {
	return &CustomerPromotionHandler{
		service: service,
		repo:    repo,
	}
}

// 1. RECORD PROMOTION EVENTS (Impression & Click Tracking)
func (h *CustomerPromotionHandler) RecordEvents(c *fiber.Ctx) error {
	var input models.RecordPromotionEventsBatchInput
	if err := c.BodyParser(&input); err != nil || len(input.Events) == 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Payload events tidak valid atau kosong", nil)
	}

	var accountIDPtr *string
	if accID, ok := c.Locals("account_id").(string); ok && strings.TrimSpace(accID) != "" {
		cleaned := strings.TrimSpace(accID)
		accountIDPtr = &cleaned
	}

	events := make([]models.PromotionEvent, 0, len(input.Events))
	for _, item := range input.Events {
		promoID := strings.TrimSpace(item.PromotionID)
		if promoID == "" {
			continue
		}
		eventType := strings.ToLower(strings.TrimSpace(item.EventType))
		if eventType != models.PromotionEventImpression && eventType != models.PromotionEventClick {
			continue
		}
		source := strings.ToLower(strings.TrimSpace(item.Source))
		if source == "" {
			source = models.PromotionEventSourceBusinessProfile
		}

		events = append(events, models.PromotionEvent{
			EventID:     utils.GenerateUUID("PEV"),
			PromotionID: promoID,
			ProductID:   item.ProductID,
			AccountID:   accountIDPtr,
			EventType:   eventType,
			Source:      source,
		})
	}

	// Fire-and-forget recording
	go func() {
		_ = h.service.RecordPromotionEvents(events)
	}()

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Events recorded", nil)
}

// 2. GET PROMOTION DETAIL (CUSTOMER VIEW)
func (h *CustomerPromotionHandler) GetPromotionDetail(c *fiber.Ctx) error {
	promotionID := strings.TrimSpace(c.Params("promotion_id"))
	if promotionID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "promotion_id wajib diisi", nil)
	}

	promo, err := h.repo.GetPromotionByID("", promotionID)
	if err != nil || promo == nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Promosi tidak ditemukan", nil)
	}

	// Record click on detail view if opened via direct notification/link
	var accountIDPtr *string
	if accID, ok := c.Locals("account_id").(string); ok && strings.TrimSpace(accID) != "" {
		cleaned := strings.TrimSpace(accID)
		accountIDPtr = &cleaned
	}
	go func() {
		_ = h.service.RecordPromotionEvents([]models.PromotionEvent{
			{
				EventID:     utils.GenerateUUID("PEV"),
				PromotionID: promo.PromotionID,
				AccountID:   accountIDPtr,
				EventType:   models.PromotionEventClick,
				Source:      models.PromotionEventSourceNotification,
			},
		})
	}()

	detail, err := h.service.GetPromotionDetail(promo.CreatedBy, "owner", promo.BusinessID, promotionID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail promosi berhasil dimuat", detail)
}
