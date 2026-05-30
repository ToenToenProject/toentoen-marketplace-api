package handlers

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"strings"
)

type BusinessPostHandler struct {
	service services.BusinessPostService
}

func NewBusinessPostHandler(s services.BusinessPostService) *BusinessPostHandler {
	return &BusinessPostHandler{
		service:        s,
	}
}

func (h *BusinessPostHandler) CreatePost(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, role, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input models.CreatePostInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request format", nil)
	}

	post, errSvc := h.service.CreatePost(requesterID, role, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Post created successfully", post)
}

func (h *BusinessPostHandler) GetPosts(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	posts, totalRows, errSvc := h.service.GetPosts(businessID, page, limit)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to fetch posts", nil)
	}

	responseData := fiber.Map{
		"posts": posts,
		"meta":  fiber.Map{"total_rows": totalRows, "page": page, "limit": limit},
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Successfully retrieved posts", responseData)
}

func (h *BusinessPostHandler) DeletePost(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	postID := strings.TrimSpace(c.Params("post_id"))

	// Ambil data user yang sedang login
	requesterID, role, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	// Panggil service untuk mengeksekusi penghapusan
	if errSvc := h.service.DeletePost(requesterID, role, businessID, postID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Post deleted successfully", nil)
}