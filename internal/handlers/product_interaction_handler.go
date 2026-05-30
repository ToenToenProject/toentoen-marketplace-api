package handlers

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductInteractionHandler struct {
	service services.ProductInteractionService
}

func NewProductInteractionHandler(service services.ProductInteractionService) *ProductInteractionHandler {
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

	if err := h.service.AddProductComment(productID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Comment added successfully", nil)
}

// 3. HANDLER GET KOMENTAR PRODUK (Struktur 1-Level Nesting)
func (h *ProductInteractionHandler) GetProductComments(c *fiber.Ctx) error {
	// Ambil product_id dari parameter URL (Flat Routing)
	productID := strings.TrimSpace(c.Params("product_id"))

	comments, err := h.service.GetProductComments(productID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Comments retrieved successfully", comments)
}