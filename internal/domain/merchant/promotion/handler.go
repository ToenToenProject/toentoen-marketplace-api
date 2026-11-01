package promotion

import (
	"strconv"
	"strings"

	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PromotionHandler struct {
	service PromotionService
}

func NewPromotionHandler(service PromotionService) *PromotionHandler {
	return &PromotionHandler{service: service}
}

// 1. GET ELIGIBLE PRODUCTS
func (h *PromotionHandler) GetEligibleProducts(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	targetType := c.Query("target_type", c.Query("promotion_type", "per_product"))
	products, err := h.service.GetEligibleProducts(requesterID, requesterRole, businessID, targetType)
	if err != nil {
		if strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar produk yang dapat dipromosikan berhasil dimuat", products)
}

// 2. CREATE PROMOTION
func (h *PromotionHandler) CreatePromotion(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.CreatePromotionInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data promosi tidak valid", nil)
	}

	if validationErrors := utils.ValidateStruct(&input); validationErrors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(validationErrors), validationErrors)
	}

	detail, err := h.service.CreatePromotion(requesterID, requesterRole, businessID, input)
	if err != nil {
		if strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Promosi berhasil dibuat", detail)
}

// 2b. CREATE BUNDLING PROMOTION
func (h *PromotionHandler) CreateBundlingPromotion(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.CreateBundlingPromotionInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data promosi bundling tidak valid", nil)
	}

	if validationErrors := utils.ValidateStruct(&input); validationErrors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(validationErrors), validationErrors)
	}

	detail, err := h.service.CreateBundlingPromotion(requesterID, requesterRole, businessID, input)
	if err != nil {
		if strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Promosi bundling berhasil dibuat", detail)
}

// 3. GET PROMOTIONS LIST (TABS: active, popular, ended)
func (h *PromotionHandler) GetPromotions(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	tab := strings.ToLower(strings.TrimSpace(c.Query("tab", "active")))
	cursor := strings.TrimSpace(c.Query("cursor"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	res, err := h.service.GetPromotions(requesterID, requesterRole, businessID, tab, cursor, limit)
	if err != nil {
		if strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar promosi berhasil dimuat", res)
}

// 4. GET PROMOTION DETAIL
func (h *PromotionHandler) GetPromotionDetail(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	promotionID := strings.TrimSpace(c.Params("promotion_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	detail, err := h.service.GetPromotionDetail(requesterID, requesterRole, businessID, promotionID)
	if err != nil {
		if strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail promosi berhasil dimuat", detail)
}

// 5. EXTEND PROMOTION
func (h *PromotionHandler) ExtendPromotion(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	promotionID := strings.TrimSpace(c.Params("promotion_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.ExtendPromotionInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data perpanjangan tidak valid", nil)
	}

	if validationErrors := utils.ValidateStruct(&input); validationErrors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(validationErrors), validationErrors)
	}

	if err := h.service.ExtendPromotion(requesterID, requesterRole, businessID, promotionID, input); err != nil {
		if strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Promosi berhasil diperpanjang", nil)
}

// 6. STOP PROMOTION
func (h *PromotionHandler) StopPromotion(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	promotionID := strings.TrimSpace(c.Params("promotion_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	if err := h.service.StopPromotion(requesterID, requesterRole, businessID, promotionID); err != nil {
		if strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Promosi berhasil dihentikan", nil)
}

// 7. PUSH PROMOTION NOTIFICATION
func (h *PromotionHandler) PushPromotionNotification(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	promotionID := strings.TrimSpace(c.Params("promotion_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	res, err := h.service.PushPromotionNotification(requesterID, requesterRole, businessID, promotionID)
	if err != nil {
		if strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		if strings.Contains(err.Error(), "sudah pernah dikirimkan") {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, res.Message, res)
}
