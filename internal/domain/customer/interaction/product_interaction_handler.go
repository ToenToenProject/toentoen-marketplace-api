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
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleProductLike(productID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Batal menyukai produk"
	if isLiked {
		message = "Produk berhasil disukai"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 2. HANDLER CREATE REVIEW / COMMENT / BALASAN PRODUK
func (h *ProductInteractionHandler) AddProductReview(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddProductReview(productID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	msg := "Komentar berhasil ditambahkan"
	if input.Rating != nil && *input.Rating > 0 {
		msg = "Ulasan produk berhasil ditambahkan"
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, msg, nil)
}

// 3. HANDLER EDIT REVIEW / COMMENT PRODUK
func (h *ProductInteractionHandler) EditProductComment(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))
	reviewID := strings.TrimSpace(c.Params("review_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.EditProductComment(productID, reviewID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan/komentar berhasil diperbarui", nil)
}

// 4. HANDLER DELETE REVIEW / COMMENT PRODUK
func (h *ProductInteractionHandler) DeleteProductComment(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))
	reviewID := strings.TrimSpace(c.Params("review_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteProductComment(productID, reviewID, accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan/komentar berhasil dihapus", nil)
}

// 5. HANDLER GET REVIEWS / COMMENTS PRODUK
func (h *ProductInteractionHandler) GetProductReviews(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))
	accountID, _ := c.Locals("account_id").(string)

	reviews, err := h.service.GetProductReviews(productID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan/komentar produk berhasil diambil", reviews)
}

// 6. TOGGLE LIKE REVIEW PRODUK
func (h *ProductInteractionHandler) ToggleReviewLike(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleReviewLike(reviewID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Batal menyukai ulasan/komentar"
	if isLiked {
		message = "Ulasan/komentar berhasil disukai"
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 7. REPORT REVIEW PRODUK
func (h *ProductInteractionHandler) ReportReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreatePostReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReportReview(reviewID, accountID, input.Reason); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan ulasan berhasil dikirim", nil)
}