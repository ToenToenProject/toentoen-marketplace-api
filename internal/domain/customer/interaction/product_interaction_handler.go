package interaction

import (
	"strconv"
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

// ----------------------------------------------------
// 1. LIKE PRODUK
// ----------------------------------------------------

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

// ----------------------------------------------------
// 2. REVIEW PRODUK (ULASAN BERBINTANG)
// ----------------------------------------------------

func (h *ProductInteractionHandler) AddProductReview(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateProductReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddProductReview(productID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Ulasan produk berhasil ditambahkan", nil)
}

func (h *ProductInteractionHandler) UpdateProductReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateProductReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateProductReview(reviewID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan produk berhasil diperbarui", nil)
}

func (h *ProductInteractionHandler) DeleteProductReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteProductReview(reviewID, accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan produk berhasil dihapus", nil)
}

func (h *ProductInteractionHandler) GetProductReviews(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))
	accountID, _ := c.Locals("account_id").(string)

	var ratingPtr *int
	if ratingStr := c.Query("rating"); ratingStr != "" {
		if rVal, err := strconv.Atoi(ratingStr); err == nil && rVal >= 1 && rVal <= 5 {
			ratingPtr = &rVal
		}
	}

	reviews, err := h.service.GetProductReviews(productID, accountID, ratingPtr)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan produk berhasil diambil", reviews)
}

func (h *ProductInteractionHandler) GetProductReviewDetail(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, _ := c.Locals("account_id").(string)

	review, err := h.service.GetProductReviewDetail(reviewID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail ulasan produk berhasil diambil", review)
}

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

	message := "Batal menyukai ulasan"
	if isLiked {
		message = "Ulasan berhasil disukai"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

func (h *ProductInteractionHandler) AddReviewReply(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateProductReviewReplyInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddReviewReply(reviewID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Balasan ulasan berhasil ditambahkan", nil)
}

func (h *ProductInteractionHandler) DeleteReviewReply(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteReviewReply(commentID, accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Balasan ulasan berhasil dihapus", nil)
}

func (h *ProductInteractionHandler) ToggleReviewReplyLike(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleReviewReplyLike(commentID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Batal menyukai balasan"
	if isLiked {
		message = "Balasan berhasil disukai"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

func (h *ProductInteractionHandler) ReportReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReportReview(reviewID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan ulasan berhasil dikirim", nil)
}

func (h *ProductInteractionHandler) ReportReviewReply(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReportReviewReply(commentID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan balasan berhasil dikirim", nil)
}

// ----------------------------------------------------
// 3. KOMENTAR DISKUSI / Q&A PRODUK
// ----------------------------------------------------

func (h *ProductInteractionHandler) AddProductComment(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateProductCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddProductComment(productID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Komentar diskusi berhasil ditambahkan", nil)
}

func (h *ProductInteractionHandler) EditProductComment(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateProductCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.EditProductComment(commentID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar diskusi berhasil diperbarui", nil)
}

func (h *ProductInteractionHandler) DeleteProductComment(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteProductComment(commentID, accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar diskusi berhasil dihapus", nil)
}

func (h *ProductInteractionHandler) GetProductComments(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))
	accountID, _ := c.Locals("account_id").(string)

	comments, err := h.service.GetProductComments(productID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar diskusi produk berhasil diambil", comments)
}

func (h *ProductInteractionHandler) ToggleProductCommentLike(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleProductCommentLike(commentID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Batal menyukai komentar"
	if isLiked {
		message = "Komentar berhasil disukai"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

func (h *ProductInteractionHandler) ReportProductComment(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReportProductComment(commentID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan komentar berhasil dikirim", nil)
}