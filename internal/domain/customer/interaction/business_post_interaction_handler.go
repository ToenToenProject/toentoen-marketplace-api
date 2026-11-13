package interaction

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PostInteractionHandler struct {
	service PostInteractionService
}

func NewPostInteractionHandler(service PostInteractionService) *PostInteractionHandler {
	return &PostInteractionHandler{service}
}

// 1. HANDLER TOGGLE LIKE POSTINGAN
func (h *PostInteractionHandler) ToggleLike(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		postID = strings.TrimSpace(c.Params("postId"))
	}

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.TogglePostLike(postID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Batal menyukai postingan"
	if isLiked {
		message = "Postingan berhasil disukai"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 2. HANDLER CREATE COMMENT / REPLY
func (h *PostInteractionHandler) AddPostComment(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		postID = strings.TrimSpace(c.Params("postId"))
	}

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

	if strings.TrimSpace(input.Content) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Isi komentar tidak boleh kosong", nil)
	}

	if err := h.service.AddPostComment(postID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Komentar berhasil ditambahkan", nil)
}

// 3. HANDLER EDIT COMMENT
func (h *PostInteractionHandler) EditPostComment(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		postID = strings.TrimSpace(c.Params("postId"))
	}
	commentID := strings.TrimSpace(c.Params("comment_id"))

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

	if err := h.service.EditPostComment(postID, commentID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar berhasil diperbarui", nil)
}

// 4. HANDLER DELETE COMMENT
func (h *PostInteractionHandler) DeletePostComment(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		postID = strings.TrimSpace(c.Params("postId"))
	}
	commentID := strings.TrimSpace(c.Params("comment_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeletePostComment(postID, commentID, accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar berhasil dihapus", nil)
}

// 5. HANDLER GET COMMENTS
func (h *PostInteractionHandler) GetPostComments(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		postID = strings.TrimSpace(c.Params("postId"))
	}
	currentAccountID, _, _ := utils.GetRequesterInfo(c)

	comments, err := h.service.GetPostComments(postID, currentAccountID)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":       "success",
			"responseCode": constants.Success,
			"message":      "Komentar berhasil diambil",
			"data":         []models.CommentResponse{},
		})
	}

	if comments == nil {
		comments = []models.CommentResponse{}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"responseCode": constants.Success,
		"message":      "Komentar berhasil diambil",
		"data":         comments,
	})
}

// 6. TOGGLE LIKE KOMENTAR POSTINGAN
func (h *PostInteractionHandler) ToggleCommentLike(c *fiber.Ctx) error {
	commentID := strings.TrimSpace(c.Params("comment_id"))
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.TogglePostCommentLike(commentID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Batal menyukai komentar"
	if isLiked {
		message = "Komentar berhasil disukai"
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 7. HANDLER REPORT POST
func (h *PostInteractionHandler) ReportPost(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		postID = strings.TrimSpace(c.Params("postId"))
	}

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

	if err := h.service.ReportPost(postID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan postingan berhasil dikirim", nil)
}