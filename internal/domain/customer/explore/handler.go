package explore

import (
	"strconv"
	"strings"

	"toentoen-backend/internal/domain/shared/models"
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
	keyword := strings.TrimSpace(c.Query("keyword", ""))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("search", ""))
	}
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("q", ""))
	}

	sort := strings.ToLower(strings.TrimSpace(c.Query("sort", "")))

	// Parse category_ids, category_id, category
	var catIDs []int64
	var catName string

	rawCatIDs := c.Query("category_ids")
	if rawCatIDs != "" {
		for _, part := range strings.Split(rawCatIDs, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
				catIDs = append(catIDs, id)
			}
		}
	}
	if singleCatID, err := strconv.ParseInt(c.Query("category_id", ""), 10, 64); err == nil && singleCatID > 0 {
		catIDs = append(catIDs, singleCatID)
	}

	rawCat := strings.TrimSpace(c.Query("category"))
	if rawCat != "" {
		if id, err := strconv.ParseInt(rawCat, 10, 64); err == nil && id > 0 {
			catIDs = append(catIDs, id)
		} else {
			catName = rawCat
		}
	}

	// Parse tags (mendukung format nama string misal "WiFi, AC" maupun array ID "10, 11")
	var tagIDs []int64
	var tagNames []string

	rawTags := c.Query("tags")
	if rawTags == "" {
		rawTags = c.Query("tag")
	}
	if rawTags == "" {
		rawTags = c.Query("tag_ids")
	}

	if rawTags != "" {
		for _, item := range strings.Split(rawTags, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if id, err := strconv.ParseInt(trimmed, 10, 64); err == nil && id > 0 {
				tagIDs = append(tagIDs, id)
			} else {
				tagNames = append(tagNames, trimmed)
			}
		}
	}

	req := SearchRequest{
		Keyword:     keyword,
		CategoryIDs: catIDs,
		Category:    catName,
		TagIDs:      tagIDs,
		TagNames:    tagNames,
		RawTags:     rawTags,
		Sort:        sort,
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

	// Default sort resolution
	if req.Sort == "" {
		if req.Lat != 0 && req.Lng != 0 {
			req.Sort = "nearest"
		} else {
			req.Sort = "rating"
		}
	}

	// Validate nearest requirement hanya jika user meminta secara eksplisit sort nearest tanpa koordinat
	if req.Sort == "nearest" && (req.Lat == 0 || req.Lng == 0) {
		return utils.SendResponse(c, 400, "11", "Latitude dan longitude wajib diisi untuk pengurutan 'terdekat'", nil)
	}

	res, err := h.service.GetExploreBusinesses(req)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal mencari bisnis", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat eksplorasi bisnis", res)
}

func (h *ExploreHandler) GetTimeline(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	lat, _ := strconv.ParseFloat(c.Query("lat", "0"), 64)
	lng, _ := strconv.ParseFloat(c.Query("lng", "0"), 64)
	radius, _ := strconv.ParseFloat(c.Query("radius", "10"), 64)
	categoryID, _ := strconv.ParseInt(c.Query("category_id", "0"), 10, 64)

	// Optional auth
	var accountID string
	authAccountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth == nil {
		accountID = authAccountID
	}

	query := models.TimelineQueryInput{
		Lat:        lat,
		Lng:        lng,
		RadiusKm:   radius,
		CategoryID: categoryID,
		Page:       page,
		Limit:      limit,
	}

	posts, err := h.service.GetTimelinePosts(query, accountID)
	if err != nil {
		return utils.SendResponse(c, 500, "99", "Gagal memuat postingan linimasa", nil)
	}

	return utils.SendResponse(c, 200, "00", "Berhasil memuat linimasa", posts)
}

func (h *ExploreHandler) RecordFeedEvents(c *fiber.Ctx) error {
	var input models.FeedEventBatchInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, "11", "Format request tidak valid", nil)
	}

	var accountID string
	authAccountID, _, errAuth := utils.GetRequesterInfo(c)
	if errAuth == nil {
		accountID = authAccountID
	}

	// Proses batch events di background agar response ke client instan (<5ms)
	go func(accID string, batch models.FeedEventBatchInput) {
		_ = h.service.RecordFeedEvents(accID, batch)
	}(accountID, input)

	return utils.SendResponse(c, fiber.StatusOK, "00", "Aktivitas feed berhasil direkam", nil)
}

