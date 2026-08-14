package meta

import (
	"strconv"
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type MetaHandler struct {
	service MetaService
}

func NewMetaHandler(service MetaService) *MetaHandler {
	return &MetaHandler{service}
}

func (h *MetaHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.service.GetCategories()
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil daftar kategori", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar kategori", categories)
}

func (h *MetaHandler) GetTags(c *fiber.Ctx) error {
	var categoryID *int64
	if catIDStr := c.Query("category_id"); catIDStr != "" {
		if id, err := strconv.ParseInt(catIDStr, 10, 64); err == nil {
			categoryID = &id
		}
	}

	tags, err := h.service.GetTags(categoryID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil daftar tag", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar tag", tags)
}

func (h *MetaHandler) GetStoreTypes(c *fiber.Ctx) error {
	storeTypes := h.service.GetStoreTypes()
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil tipe bisnis", storeTypes)
}
