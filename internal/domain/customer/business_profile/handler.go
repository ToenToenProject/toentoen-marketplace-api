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

// GetBookmarks Handler untuk menarik daftar toko yang disimpan pengguna (Bookmarks)
// Mendukung query parameter untuk mode pencarian & integrasi wishlist:
//   - include_wishlist: "true" (menggabungkan toko yang produknya di-wishlist pengguna)
//   - filter: "all", "saved", "wishlisted" (filter tipe simpanan toko)
//   - format: "object" (default jika include_wishlist/filter diisi, mengembalikan ID array + list merchant), "array" / "list" (langsung slice merchant)
func (h *Handler) GetBookmarks(c *fiber.Ctx) error {
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

	includeWishlist := strings.EqualFold(strings.TrimSpace(c.Query("include_wishlist", "")), "true")
	filterQuery := strings.ToLower(strings.TrimSpace(c.Query("filter", "")))
	format := strings.ToLower(strings.TrimSpace(c.Query("format", "")))

	// Mode Terpadu: Jika meminta integrasi wishlist atau spesifik filter simpanan toko
	if includeWishlist || filterQuery != "" {
		filter := "all"
		if filterQuery != "" {
			filter = filterQuery
		}

		res, err := h.service.GetSavedAndWishlistedShops(accountID, filter)
		if err != nil {
			return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memuat daftar toko tersimpan dan wishlist", nil)
		}

		if format == "array" || format == "list" {
			return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan berhasil diambil", res.Merchants)
		}

		return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan berhasil diambil", res)
	}

	// Mode Standar (Default): Daftar toko yang dibookmark pengguna
	if accountID == "" {
		// Pengunjung tamu / belum login: kembalikan list kosong tanpa error 401
		return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan berhasil diambil", []BusinessProfileResponse{})
	}

	bookmarks, err := h.service.GetBookmarks(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan berhasil diambil", bookmarks)
}

// GetSavedAndWishlistedShops Handler untuk menarik daftar toko yang disimpan (bookmark) dan/atau di-wishlist pengguna
// GET /shops/saved_and_wishlisted
// Query params:
//   - filter: "all" (default), "saved", "wishlisted"
//   - format: "object" (default, berisi shops + ID list), "array" / "list" (langsung slice shops)
func (h *Handler) GetSavedAndWishlistedShops(c *fiber.Ctx) error {
	accountID, _ := c.Locals("account_id").(string)

	filter := strings.ToLower(strings.TrimSpace(c.Query("filter", "all")))
	format := strings.ToLower(strings.TrimSpace(c.Query("format", "")))

	res, err := h.service.GetSavedAndWishlistedShops(accountID, filter)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memuat daftar toko tersimpan dan wishlist", nil)
	}

	if format == "array" || format == "list" {
		return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan dan wishlist berhasil diambil", res.Shops)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar toko tersimpan dan wishlist berhasil diambil", res)
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
