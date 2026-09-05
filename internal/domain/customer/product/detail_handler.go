package product

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductDetailHandler struct {
	service ProductDetailService
}

func NewProductDetailHandler(service ProductDetailService) *ProductDetailHandler {
	return &ProductDetailHandler{service}
}

func (h *ProductDetailHandler) GetProductDetail(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("product_id"))

	res, err := h.service.GetProductDetail(productID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Produk tidak ditemukan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail produk berhasil diambil", res)
}
