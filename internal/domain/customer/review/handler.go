package review

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ReviewHandler struct {
	service ReviewService
}

func NewReviewHandler(s ReviewService) *ReviewHandler {
	return &ReviewHandler{s}
}

// Customer Buat Review
func (h *ReviewHandler) CreateReview(c *fiber.Ctx) error {
	var input models.CreateReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	// Asumsi getRequesterInfo mereturn ID user dari token JWT
	customerID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	if err := h.service.CreateReview(customerID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Review submitted successfully", nil)
}

// Owner Balas Review
func (h *ReviewHandler) ReplyReview(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")

	var input models.ReplyReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	ownerID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	if err := h.service.ReplyReview(ownerID, reviewID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Reply submitted successfully", nil)
}

// Get All (Publik)
func (h *ReviewHandler) GetAllReviews(c *fiber.Ctx) error {
	businessID := c.Params("business_id")

	// 1. Ambil list review (komen-komennya)
	reviews, err := h.service.GetAllReviews(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, "FAILED", err.Error(), nil)
	}

	// 2. Ambil data statistik/summary
	summary, _ := h.service.GetReviewSummary(businessID) // Asumsi Anda membuat wrapper ini di Service

	// 3. Gabungkan dalam 1 response JSON yang rapi
	return utils.SendResponse(c, fiber.StatusOK, "SUCCESS", "Reviews retrieved", fiber.Map{
		"summary": summary,
		"reviews": reviews,
	})
}

// Get Detail (Publik)
func (h *ReviewHandler) GetReviewDetail(c *fiber.Ctx) error {
	reviewID := c.Params("review_id")
	review, err := h.service.GetReviewDetail(reviewID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Review detail retrieved", review)
}