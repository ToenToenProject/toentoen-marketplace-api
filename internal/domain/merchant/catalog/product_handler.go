package catalog

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
	service ProductService
}

func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{service}
}

// 1. CREATE PRODUCT
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateProductInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if strings.TrimSpace(input.ProductName) == "" || input.Price < 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Nama produk wajib diisi dan harga tidak boleh negatif", nil)
	}

	errSvc := h.service.CreateProduct(requesterID, requesterRole, businessID, input)
	if errSvc != nil {
		if strings.Contains(errSvc.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
		}
		if strings.HasPrefix(errSvc.Error(), "gagal membuat produk:") {
			return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Produk berhasil dibuat", nil)
}

// ==========================================
// HANDLERS UNTUK PRODUCT VARIANTS
// ==========================================

func (h *ProductHandler) AddVariant(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateVariantInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if strings.TrimSpace(input.VariantName) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Nama varian wajib diisi", nil)
	}

	if err := h.service.AddVariant(requesterID, requesterRole, businessID, productID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Varian berhasil ditambahkan", nil)
}

func (h *ProductHandler) UpdateVariant(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))
	variantID := strings.TrimSpace(c.Params("variant_id"))

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateVariantInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateVariant(requesterID, requesterRole, businessID, productID, variantID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Varian berhasil diperbarui", nil)
}

func (h *ProductHandler) DeleteVariant(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))
	variantID := strings.TrimSpace(c.Params("variant_id"))

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.RemoveVariant(requesterID, requesterRole, businessID, productID, variantID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Varian berhasil dihapus", nil)
}

// 2. UPDATE PRODUCT
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateProductInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	errSvc := h.service.UpdateProduct(requesterID, requesterRole, businessID, productID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Produk berhasil diperbarui", nil)
}

// 3. DELETE PRODUCT
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if errSvc := h.service.DeleteProduct(requesterID, requesterRole, businessID, productID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Produk berhasil dihapus", nil)
}

// 4. GET DETAIL PRODUCT (Single)
func (h *ProductHandler) GetProductDetail(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	product, errSvc := h.service.GetProductDetail(requesterID, requesterRole, businessID, productID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Produk tidak ditemukan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail produk berhasil diambil", product)
}

// ==========================================
// HANDLER UNTUK LIST FLAT (FILTER)
// ==========================================

func (h *ProductHandler) GetFlatProducts(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	search := strings.TrimSpace(c.Query("search", ""))
	productGroupID := strings.TrimSpace(c.Query("product_group_id", ""))
	stockType := strings.TrimSpace(c.Query("stock_type", ""))
	cursor := strings.TrimSpace(c.Query("cursor", ""))
	sort := strings.TrimSpace(c.Query("sort", ""))
	limit := c.QueryInt("limit", 20)

	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		active := isActiveStr == "true"
		isActive = &active
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	response, errSvc := h.service.GetAllProducts(requesterID, requesterRole, businessID, search, productGroupID, stockType, isActive, sort, cursor, limit)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar produk berhasil diambil", response)
}

// 6. BULK UPDATE STATUS
func (h *ProductHandler) BulkUpdateStatus(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	// Struct inline untuk menangkap body JSON array
	var input struct {
		ProductIDs []string `json:"product_ids"`
		IsActive   bool     `json:"is_active"`
	}
	if err := c.BodyParser(&input); err != nil || len(input.ProductIDs) == 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid atau product_ids kosong", nil)
	}

	if err := h.service.BulkUpdateStatus(requesterID, requesterRole, businessID, input.ProductIDs, input.IsActive); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Status produk berhasil diperbarui", nil)
}

// 7. BULK DELETE
func (h *ProductHandler) BulkDelete(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input struct {
		ProductIDs []string `json:"product_ids"`
	}
	if err := c.BodyParser(&input); err != nil || len(input.ProductIDs) == 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid atau product_ids kosong", nil)
	}

	if err := h.service.BulkDelete(requesterID, requesterRole, businessID, input.ProductIDs); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Produk berhasil dihapus", nil)
}
