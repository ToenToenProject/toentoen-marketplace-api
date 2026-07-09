package explore

import (
	"strconv"

	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ExploreHandler struct {
	service ExploreService
}

func NewExploreHandler(service ExploreService) *ExploreHandler {
	return &ExploreHandler{service}
}

func (h *ExploreHandler) SearchBusinesses(c *fiber.Ctx) error {
	req := SearchRequest{
		Keyword: c.Query("keyword", ""),
		Sort:    c.Query("sort", "nearest"),
	}

	if lat, err := strconv.ParseFloat(c.Query("lat", "0"), 64); err == nil {
		req.Lat = lat
	}
	if lng, err := strconv.ParseFloat(c.Query("lng", "0"), 64); err == nil {
		req.Lng = lng
	}
	if radius, err := strconv.Atoi(c.Query("radius", "10")); err == nil {
		req.Radius = radius * 1000 // Convert km to meters
	}
	if page, err := strconv.ParseInt(c.Query("page", "1"), 10, 64); err == nil {
		req.Offset = page
	}
	if limit, err := strconv.ParseInt(c.Query("limit", "10"), 10, 64); err == nil {
		req.Limit = limit
	}

	// Validate nearest requirement
	if req.Sort == "nearest" && (req.Lat == 0 || req.Lng == 0) {
		return utils.SendResponse(c, 400, "11", "lat and lng are required for 'nearest' sorting", nil)
	}

	res, err := h.service.GetExploreBusinesses(req)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Failed to search businesses", nil)
	}

	return utils.SendResponse(c, 200, "00", "Success", res)
}
