package interaction

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductInteractionHandler struct {
	service ProductInteractionService
}

func NewProductInteractionHandler(service ProductInteractionService) *ProductInteractionHandler {
	return &ProductInteractionHandler{service}
}

// 1. HANDLER TOGGLE LIKE PRODUK
func (h *ProductInteractionHandler) ToggleLike(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	isLiked, err := h.service.ToggleProductLike(productID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Product unliked successfully"
	if isLiked {
		message = "Product liked successfully"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 2. HANDLER CREATE REVIEW / BALASAN PRODUK
func (h *ProductInteractionHandler) AddProductReview(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	var input models.CreateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body format", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddProductReview(productID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Review added successfully", nil)
}

// 3. HANDLER GET REVIEW PRODUK
func (h *ProductInteractionHandler) GetProductReviews(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, _ := c.Locals("account_id").(string)

	reviews, err := h.service.GetProductReviews(productID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Reviews retrieved successfully", reviews)
}

func (h *ProductInteractionHandler) ToggleReviewLike(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	isLiked, err := h.service.ToggleReviewLike(reviewID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Review unliked successfully"
	if isLiked {
		message = "Review liked successfully"
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

func (h *ProductInteractionHandler) ReportReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	var req struct {
		Reason string `json:"reason" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body format", nil)
	}

	if errors := utils.ValidateStruct(&req); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}
	if strings.TrimSpace(req.Reason) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Reason cannot be empty", nil)
	}

	if err := h.service.ReportReview(reviewID, accountID, req.Reason); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to report review", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Review reported successfully", nil)
}