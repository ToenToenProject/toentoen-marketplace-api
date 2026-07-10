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
		return utils.SendResponse(c, 401, "99", "Unauthorized", nil)
	}

	summary, err := h.service.GetMyProfileSummary(accountID)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to get profile summary", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", summary)
}

func (h *MyActivityHandler) GetMyComments(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Unauthorized", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	comments, err := h.service.GetMyComments(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to get comments", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", comments)
}

func (h *MyActivityHandler) GetMyBusinessReviews(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Unauthorized", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	reviews, err := h.service.GetMyBusinessReviews(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to get reviews", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", reviews)
}

func (h *MyActivityHandler) GetMyBookmarks(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Unauthorized", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	bookmarks, err := h.service.GetMyBookmarks(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to get bookmarks", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", bookmarks)
}

func (h *MyActivityHandler) GetMyWishlists(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Unauthorized", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	wishlists, err := h.service.GetMyWishlists(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to get wishlists", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", wishlists)
}

func (h *MyActivityHandler) ToggleWishlist(c *fiber.Ctx) error {
	accountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth != nil {
		return utils.SendResponse(c, 401, "99", "Unauthorized", nil)
	}

	productID := c.Params("product_id")
	
	isAdded, err := h.service.ToggleWishlist(accountID, productID)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to toggle wishlist", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", fiber.Map{
		"is_wishlisted": isAdded,
	})
}
