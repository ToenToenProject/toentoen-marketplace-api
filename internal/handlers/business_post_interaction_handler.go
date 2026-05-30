package handlers

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PostInteractionHandler struct {
	service services.PostInteractionService
}

func NewPostInteractionHandler(service services.PostInteractionService) *PostInteractionHandler {
	return &PostInteractionHandler{service}
}

// 1. HANDLER TOGGLE LIKE
func (h *PostInteractionHandler) ToggleLike(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	
	// Ambil AccountID dari Locals yang diset oleh Middleware RequireAuth
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	isLiked, err := h.service.TogglePostLike(postID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Post unliked successfully"
	if isLiked {
		message = "Post liked successfully"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, message, fiber.Map{"is_liked": isLiked})
}

// 2. HANDLER CREATE COMMENT / REPLY
func (h *PostInteractionHandler) AddPostComment(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized access", nil)
	}

	var input models.CreateCommentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body format", nil)
	}

	if strings.TrimSpace(input.Content) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Comment content cannot be empty", nil)
	}

	if err := h.service.AddPostComment(postID, accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Comment added successfully", nil)
}

// 3. HANDLER GET COMMENTS (Menampilkan hasil 1-Level Nesting)
func (h *PostInteractionHandler) GetPostComments(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))

	comments, err := h.service.GetPostComments(postID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Comments retrieved successfully", comments)
}