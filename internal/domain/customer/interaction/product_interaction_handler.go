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

	// Ambil AccountID dari Locals yang diset oleh Middleware RequireAuth
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

// 2. HANDLER CREATE KOMENTAR / BALASAN PRODUK
func (h *ProductInteractionHandler) AddProductComment(c *fiber.Ctx) error {
	// 1. Cukup ambil product_id dari URL (Flat Routing)
	productID := strings.TrimSpace(c.Params("product_id"))

	// 2. Ambil AccountID dari Auth Middleware
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	// 3. Parse Body JSON
	var input models.CreateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body format", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddProductComment(productID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Comment added successfully", nil)
}

// 3. HANDLER GET KOMENTAR PRODUK (Struktur 1-Level Nesting)
func (h *ProductInteractionHandler) GetProductComments(c *fiber.Ctx) error {
	// Ambil product_id dari parameter URL (Flat Routing)
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, _ := c.Locals("account_id").(string)

	comments, err := h.service.GetProductComments(productID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Comments retrieved successfully", comments)
}

func (h *ProductInteractionHandler) ToggleCommentLike(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	isLiked, err := h.service.ToggleCommentLike(commentID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Comment unliked successfully"
	if isLiked {
		message = "Comment liked successfully"
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

func (h *ProductInteractionHandler) ReportComment(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
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

	if err := h.service.ReportComment(commentID, accountID, req.Reason); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to report comment", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Comment reported successfully", nil)
}