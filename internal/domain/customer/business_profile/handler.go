package business_profile

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// GetProfile Handler untuk menarik detail profile Business
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("businessId"))
	}
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	// Ambil account_id (opsional) dari middleware auth (jika dipasang)
	var accountID string
	if id, ok := c.Locals("account_id").(string); ok && id != "" {
		accountID = id
	}
	if accountID == "" {
		authID, _, errAuth := utils.GetRequesterInfo(c)
		if errAuth == nil && authID != "" {
			accountID = authID
		}
	}

	// Baca konteks sumber kunjungan (dikirim dari Flutter untuk analytics)
	sourceType := c.Query("source")
	searchedTag := c.Query("tag")
	productID := c.Query("product_id")

	profile, err := h.service.GetBusinessProfile(businessID, accountID, sourceType, searchedTag, productID)
	if err != nil {
		if err.Error() == "business not found" || err.Error() == "bisnis tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Toko tidak ditemukan", nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"responseCode": constants.Success,
		"message":      "Profil toko berhasil diambil",
		"data":         profile,
	})
}

// ToggleLike Handler untuk aksi Like/Unlike profil bisnis
func (h *Handler) ToggleLike(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("businessId"))
	}
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	// Wajib login
	var accountID string
	if id, ok := c.Locals("account_id").(string); ok && id != "" {
		accountID = id
	}
	if accountID == "" {
		authID, _, errAuth := utils.GetRequesterInfo(c)
		if errAuth == nil && authID != "" {
			accountID = authID
		}
	}
	if accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isLiked, totalLikes, err := h.service.ToggleLike(accountID, businessID)
	if err != nil {
		if err.Error() == "business not found" || err.Error() == "bisnis tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Toko tidak ditemukan", nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	message := "Profil bisnis berhasil batal disukai"
	if isLiked {
		message = "Profil bisnis berhasil disukai"
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"responseCode": constants.Success,
		"message":      message,
		"data": fiber.Map{
			"business_id": businessID,
			"is_liked":    isLiked,
			"total_likes": totalLikes,
			"like_count":  totalLikes,
		},
	})
}

// ToggleBookmark Handler untuk menyimpan/menghapus toko favorit
func (h *Handler) ToggleBookmark(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	// Wajib login
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	isBookmarked, err := h.service.ToggleBookmark(accountID, businessID)
	if err != nil {
		if err.Error() == "business not found" || err.Error() == "bisnis tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Toko tidak ditemukan", nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	msg := "Toko berhasil dihapus dari daftar simpan"
	if isBookmarked {
		msg = "Toko berhasil disimpan ke daftar simpan"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, fiber.Map{
		"is_bookmarked": isBookmarked,
	})
}

// GetBookmarks Handler untuk menarik daftar toko yang disimpan pengguna
func (h *Handler) GetBookmarks(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	bookmarks, err := h.service.GetBookmarks(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan berhasil diambil", bookmarks)
}

func (h *Handler) GetProductGroups(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	groups, err := h.service.GetAllProductGroups(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Grup produk berhasil diambil", groups)
}

func (h *Handler) GetProducts(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	groupID := c.Query("group_id")

	products, err := h.service.GetAllProducts(businessID, groupID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar produk berhasil diambil", products)
}

func (h *Handler) GetPosts(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("businessId"))
	}
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	var accountID string
	if id, ok := c.Locals("account_id").(string); ok && id != "" {
		accountID = id
	}
	if accountID == "" {
		authID, _, errAuth := utils.GetRequesterInfo(c)
		if errAuth == nil && authID != "" {
			accountID = authID
		}
	}

	posts, err := h.service.GetAllPosts(businessID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}
	if posts == nil {
		posts = []models.PostResponse{}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"responseCode": constants.Success,
		"message":      "Daftar postingan berhasil diambil",
		"data":         posts,
	})
}
