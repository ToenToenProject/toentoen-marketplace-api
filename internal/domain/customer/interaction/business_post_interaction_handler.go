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

// 1. HANDLER TOGGLE LIKE
func (h *PostInteractionHandler) ToggleLike(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	
	// Ambil AccountID dari Locals yang diset oleh Middleware RequireAuth
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, err := h.service.TogglePostLike(postID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Post berhasil di-unlike"
	if isLiked {
		message = "Post berhasil di-like"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 2. HANDLER CREATE COMMENT / REPLY
func (h *PostInteractionHandler) AddPostComment(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if strings.TrimSpace(input.Content) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Isi komentar tidak boleh kosong", nil)
	}

	if err := h.service.AddPostComment(postID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Komentar berhasil ditambahkan", nil)
}

// 3. HANDLER GET COMMENTS (Menampilkan hasil 1-Level Nesting)
func (h *PostInteractionHandler) GetPostComments(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))

	comments, err := h.service.GetPostComments(postID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Komentar berhasil diambil", comments)
}

// 4. HANDLER REPORT POST
func (h *PostInteractionHandler) ReportPost(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreatePostReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if strings.TrimSpace(input.Reason) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Alasan laporan tidak boleh kosong", nil)
	}

	if err := h.service.ReportPost(postID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Post berhasil dilaporkan", nil)
}