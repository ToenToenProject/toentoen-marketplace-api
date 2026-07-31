package review

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
	"toentoen-backend/internal/domain/customer/interaction"

	"github.com/gofiber/fiber/v2"
)

type ReviewHandler struct {
	service        ReviewService
	productService interaction.ProductInteractionService
}

func NewReviewHandler(s ReviewService, ps interaction.ProductInteractionService) *ReviewHandler {
	return &ReviewHandler{s, ps}
}

// Customer Buat Review
func (h *ReviewHandler) CreateReview(c *fiber.Ctx) error {
	var input models.CreateReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	customerID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.CreateReview(customerID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Ulasan berhasil dikirim", nil)
}

// Owner Balas Review
func (h *ReviewHandler) ReplyReview(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")

	var input models.ReplyReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	ownerID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.ReplyReview(ownerID, reviewID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Balasan berhasil dikirim", nil)
}

// Get All (Publik)
func (h *ReviewHandler) GetAllReviews(c *fiber.Ctx) error {
	businessID := c.Params("business_id")

	// 1. Ambil list business review
	businessReviews, err := h.service.GetAllReviews(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	// 2. Ambil list product review
	productReviews, err := h.productService.GetAllProductCommentsByBusinessID(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	// 3. Ambil data statistik/summary
	summary, _ := h.service.GetReviewSummary(businessID)

	// 4. Gabungkan dalam 1 response JSON
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan berhasil diambil", fiber.Map{
		"summary":          summary,
		"business_reviews": businessReviews,
		"product_reviews":  productReviews,
	})
}

// Get Detail (Publik)
func (h *ReviewHandler) GetReviewDetail(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")
	review, err := h.service.GetReviewDetail(reviewID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Ulasan tidak ditemukan", nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail ulasan berhasil diambil", review)
}

func (h *ReviewHandler) ToggleReviewLike(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleReviewLike(reviewID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil", fiber.Map{"is_liked": isLiked})
}

func (h *ReviewHandler) AddReviewComment(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.AddReviewComment(reviewID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Komentar berhasil ditambahkan", nil)
}

func (h *ReviewHandler) GetReviewComments(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")
	
	// Optional account ID if logged in
	accountID := ""
	if accID, _, errAuth := utils.GetRequesterInfo(c); errAuth == nil {
		accountID = accID
	}

	comments, err := h.service.GetReviewComments(reviewID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar berhasil diambil", comments)
}

func (h *ReviewHandler) ToggleReviewCommentLike(c *fiber.Ctx) error {
	commentID := c.Params("comment_id")
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleReviewCommentLike(commentID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil", fiber.Map{"is_liked": isLiked})
}

func (h *ReviewHandler) ReportReview(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreatePostReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReportReview(reviewID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Ulasan berhasil dilaporkan", nil)
}

func (h *ReviewHandler) ReportReviewComment(c *fiber.Ctx) error {
	commentID := c.Params("comment_id")
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreatePostReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReportReviewComment(commentID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Komentar berhasil dilaporkan", nil)
}