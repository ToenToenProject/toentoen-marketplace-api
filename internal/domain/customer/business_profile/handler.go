package business_profile

import (
	"github.com/gofiber/fiber/v2"
	"toentoen-backend/pkg/utils"
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
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Business ID is required", "D1", nil)
	}

	// Ambil account_id (opsional) dari middleware auth (jika dipasang)
	var accountID string
	if id, ok := c.Locals("account_id").(string); ok {
		accountID = id
	}

	profile, err := h.service.GetBusinessProfile(businessID, accountID)
	if err != nil {
		if err.Error() == "business not found" {
			return utils.SendResponse(c, fiber.StatusNotFound, "Business not found", "D1", nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, "Business profile fetched successfully", "00", profile)
}

// ToggleBookmark Handler untuk menyimpan/menghapus toko favorit
func (h *Handler) ToggleBookmark(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Business ID is required", "D1", nil)
	}

	// Wajib login
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, "Unauthorized", "A1", nil)
	}

	isBookmarked, err := h.service.ToggleBookmark(accountID, businessID)
	if err != nil {
		if err.Error() == "business not found" {
			return utils.SendResponse(c, fiber.StatusNotFound, "Business not found", "D1", nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	msg := "Business removed from bookmarks"
	if isBookmarked {
		msg = "Business bookmarked successfully"
	}

	return utils.SendResponse(c, fiber.StatusOK, msg, "00", fiber.Map{
		"is_bookmarked": isBookmarked,
	})
}

// GetBookmarks Handler untuk menarik daftar toko yang disimpan pengguna
func (h *Handler) GetBookmarks(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, "Unauthorized", "A1", nil)
	}

	bookmarks, err := h.service.GetBookmarks(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, "Bookmarks fetched successfully", "00", bookmarks)
}

func (h *Handler) GetProductGroups(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Business ID is required", "D1", nil)
	}

	groups, err := h.service.GetAllProductGroups(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, "Product groups fetched successfully", "00", groups)
}

func (h *Handler) GetProducts(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Business ID is required", "D1", nil)
	}

	groupID := c.Query("group_id")

	products, err := h.service.GetAllProducts(businessID, groupID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, "Products fetched successfully", "00", products)
}

func (h *Handler) GetPosts(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, "Business ID is required", "D1", nil)
	}

	var accountID string
	if id, ok := c.Locals("account_id").(string); ok {
		accountID = id
	}

	posts, err := h.service.GetAllPosts(businessID, accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, err.Error(), "E1", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, "Posts fetched successfully", "00", posts)
}
