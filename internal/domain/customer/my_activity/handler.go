package my_activity

import (
	"strconv"
	"toentoen-backend/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type MyActivityHandler struct {
	service MyActivityService
}

func NewMyActivityHandler(service MyActivityService) *MyActivityHandler {
	return &MyActivityHandler{service: service}
}

func (h *MyActivityHandler) GetMyProfileSummary(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Sesi tidak valid", nil)
	}

	summary, err := h.service.GetMyProfileSummary(accountID)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal memuat ringkasan profil", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat ringkasan profil", summary)
}

func (h *MyActivityHandler) GetMyComments(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Sesi tidak valid", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	comments, err := h.service.GetMyComments(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal memuat komentar", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat komentar", comments)
}

func (h *MyActivityHandler) GetMyBusinessReviews(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Sesi tidak valid", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	reviewType := c.Query("type", c.Query("filter", "all"))

	reviews, err := h.service.GetMyBusinessReviews(accountID, page, limit, reviewType)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal memuat ulasan", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat ulasan", reviews)
}

func (h *MyActivityHandler) GetMyBookmarks(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Sesi tidak valid", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	bookmarks, err := h.service.GetMyBookmarks(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal memuat toko tersimpan", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat toko tersimpan", bookmarks)
}

func (h *MyActivityHandler) GetMyWishlists(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Sesi tidak valid", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	wishlists, err := h.service.GetMyWishlists(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal memuat daftar keinginan", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat daftar keinginan", wishlists)
}

func (h *MyActivityHandler) ToggleWishlist(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Sesi tidak valid", nil)
	}

	productID := c.Params("product_id")
	
	isAdded, err := h.service.ToggleWishlist(accountID, productID)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal mengubah status daftar keinginan", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memperbarui daftar keinginan", fiber.Map{
		"is_wishlisted": isAdded,
	})
}
