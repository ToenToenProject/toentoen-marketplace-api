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
