package review

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/customer/interaction"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ReviewHandler struct {
	service        ReviewService
	productService interaction.ProductInteractionService
}

func NewReviewHandler(s ReviewService, ps interaction.ProductInteractionService) *ReviewHandler {
	return &ReviewHandler{s, ps}
}

// 1. Customer Buat Review Bisnis
func (h *ReviewHandler) CreateReview(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	var input models.CreateReviewInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	input.BusinessID = businessID

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	customerID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.CreateReview(customerID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Ulasan berhasil dikirim", nil)
}

// 2. Customer Update Review Bisnis
func (h *ReviewHandler) UpdateReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	if reviewID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Review ID wajib diisi", nil)
	}

	var input models.UpdateReviewInput
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

	if err := h.service.UpdateReview(customerID, reviewID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan berhasil diperbarui", nil)
}

// 3. Delete Review Bisnis (Customer pemilik atau Merchant)
func (h *ReviewHandler) DeleteReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	if reviewID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Review ID wajib diisi", nil)
	}

	requesterID, role, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteReview(requesterID, role, reviewID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Ulasan berhasil dihapus", nil)
}

// 4. Balas Review Bisnis (Customer & Merchant)
func (h *ReviewHandler) ReplyReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	if reviewID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Review ID wajib diisi", nil)
	}

	var input models.CreateReviewReplyInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	requesterID, role, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.ReplyReview(requesterID, role, reviewID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Balasan berhasil dikirim", nil)
}

// 5. Delete Balasan Review Bisnis
func (h *ReviewHandler) DeleteReviewComment(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	if commentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Comment ID wajib diisi", nil)
	}

	requesterID, role, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteReviewComment(requesterID, role, commentID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Balasan berhasil dihapus", nil)
}

// 6. Get Smart Display 5 Review Toko (Publik)
func (h *ReviewHandler) GetSmartDisplayReviews(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("business_id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("businessId"))
	}
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	currentAccountID, _, _ := utils.GetRequesterInfo(c)

	reviews, err := h.service.GetSmartDisplayReviews(businessID, currentAccountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	summary, _ := h.service.GetReviewSummary(businessID)

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Smart display ulasan berhasil diambil", fiber.Map{
		"summary": summary,
		"reviews": reviews,
	})
}

// 7. Get All Reviews Toko (Paginated & Filterable)
func (h *ReviewHandler) GetAllReviews(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("business_id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("businessId"))
	}
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	currentAccountID, _, _ := utils.GetRequesterInfo(c)
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	ratingFilter := c.QueryInt("rating", 0)
	sortStr := c.Query("sort", "newest")

	reviews, total, err := h.service.GetAllReviews(businessID, currentAccountID, page, limit, ratingFilter, sortStr)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	summary, _ := h.service.GetReviewSummary(businessID)

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar ulasan berhasil diambil", fiber.Map{
		"summary":    summary,
		"reviews":    reviews,
		"meta":       fiber.Map{"page": page, "limit": limit, "total_rows": total},
	})
}

// 8. Get Detail Review Toko beserta Balasannya (Publik)
func (h *ReviewHandler) GetReviewDetail(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	if reviewID == "" {
		reviewID = strings.TrimSpace(c.Params("id"))
	}
	if reviewID == "" {
		reviewID = strings.TrimSpace(c.Params("reviewId"))
	}
	if reviewID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Review ID wajib diisi", nil)
	}

	currentAccountID, _, _ := utils.GetRequesterInfo(c)

	review, err := h.service.GetReviewDetail(reviewID, currentAccountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Ulasan tidak ditemukan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail ulasan berhasil diambil", review)
}

// 9. Toggle Like Review Toko
func (h *ReviewHandler) ToggleReviewLike(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleReviewLike(reviewID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	msg := "Ulasan berhasil disukai"
	if !isLiked {
		msg = "Batal menyukai ulasan"
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, fiber.Map{"is_liked": isLiked})
}

// 10. Toggle Like Balasan Review Toko
func (h *ReviewHandler) ToggleReviewCommentLike(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.ToggleReviewCommentLike(commentID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	msg := "Balasan berhasil disukai"
	if !isLiked {
		msg = "Batal menyukai balasan"
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, fiber.Map{"is_liked": isLiked})
}

// 11. Report Review Toko
func (h *ReviewHandler) ReportReview(c *fiber.Ctx) error {
	reviewID := strings.TrimSpace(c.Params("review_id"))
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
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan ulasan berhasil dikirim", nil)
}

// 12. Report Balasan Review Toko
func (h *ReviewHandler) ReportReviewComment(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
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
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan balasan berhasil dikirim", nil)
}