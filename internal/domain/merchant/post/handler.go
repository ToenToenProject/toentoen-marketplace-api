package post

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"strings"
)

type BusinessPostHandler struct {
	service BusinessPostService
}

func NewBusinessPostHandler(s BusinessPostService) *BusinessPostHandler {
	return &BusinessPostHandler{
		service:        s,
	}
}

func (h *BusinessPostHandler) CreatePost(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreatePostInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	_, errSvc := h.service.CreatePost(requesterID, role, businessID, input)
	if errSvc != nil {
		if strings.Contains(errSvc.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Post berhasil dibuat", nil)
}

func (h *BusinessPostHandler) GetPosts(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	posts, totalRows, errSvc := h.service.GetPosts(businessID, page, limit)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data post", nil)
	}

	responseData := fiber.Map{
		"posts": posts,
		"meta":  fiber.Map{"total_rows": totalRows, "page": page, "limit": limit},
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar post berhasil diambil", responseData)
}

func (h *BusinessPostHandler) GetPostDetail(c *fiber.Ctx) error {
	postID := strings.TrimSpace(c.Params("post_id"))
	if postID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Post ID wajib diisi", nil)
	}

	post, err := h.service.GetPostByID(postID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Post tidak ditemukan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail post berhasil diambil", post)
}

func (h *BusinessPostHandler) UpdatePost(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	postID := strings.TrimSpace(c.Params("post_id"))

	requesterID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdatePostInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	updatedPost, errSvc := h.service.UpdatePost(requesterID, role, businessID, postID, input)
	if errSvc != nil {
		if strings.Contains(errSvc.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
		}
		if strings.Contains(errSvc.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Post berhasil diperbarui", updatedPost)
}

func (h *BusinessPostHandler) DeletePost(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	postID := strings.TrimSpace(c.Params("post_id"))

	// Ambil data user yang sedang login
	requesterID, role, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	// Panggil service untuk mengeksekusi penghapusan
	if errSvc := h.service.DeletePost(requesterID, role, businessID, postID); errSvc != nil {
		if strings.Contains(errSvc.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Post berhasil dihapus", nil)
}