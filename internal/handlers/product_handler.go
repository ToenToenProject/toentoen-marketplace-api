package handlers

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
	service services.ProductService
}

func NewProductHandler(service services.ProductService) *ProductHandler {
	return &ProductHandler{service}
}

// Helper function untuk mengambil data requester dari Context
func getRequesterInfo(c *fiber.Ctx) (string, string, error) {
	accountIDValue := c.Locals("account_id")
	roleValue := c.Locals("role")

	if accountIDValue == nil || roleValue == nil {
		return "", "", fiber.ErrUnauthorized
	}

	requesterID, ok1 := accountIDValue.(string)
	requesterRole, ok2 := roleValue.(string)
	if !ok1 || !ok2 {
		return "", "", fiber.ErrInternalServerError
	}

	return requesterID, requesterRole, nil
}

// 1. CREATE PRODUCT
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Invalid session or unauthorized access", nil)
	}

	var input models.CreateProductInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request format", nil)
	}

	if strings.TrimSpace(input.ProductName) == "" || input.Price < 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Product name is required and price cannot be negative", nil)
	}

	errSvc := h.service.CreateProduct(requesterID, requesterRole, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Product created successfully", nil)
}

// ==========================================
// HANDLERS UNTUK PRODUCT VARIANTS
// ==========================================

func (h *ProductHandler) AddVariant(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input models.CreateVariantInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	if strings.TrimSpace(input.VariantName) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Variant name is required", nil)
	}

	if err := h.service.AddVariant(requesterID, requesterRole, businessID, productID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Variant added successfully", nil)
}

func (h *ProductHandler) UpdateVariant(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))
	variantID := strings.TrimSpace(c.Params("variant_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input models.UpdateVariantInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	if err := h.service.UpdateVariant(requesterID, requesterRole, businessID, productID, variantID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Variant updated successfully", nil)
}

func (h *ProductHandler) DeleteVariant(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))
	variantID := strings.TrimSpace(c.Params("variant_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	if err := h.service.RemoveVariant(requesterID, requesterRole, businessID, productID, variantID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Variant deleted successfully", nil)
}

// 2. UPDATE PRODUCT
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Invalid session or unauthorized access", nil)
	}

	var input models.UpdateProductInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request format", nil)
	}

	errSvc := h.service.UpdateProduct(requesterID, requesterRole, businessID, productID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Product updated successfully", nil)
}

// 3. DELETE PRODUCT
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Invalid session or unauthorized access", nil)
	}

	if errSvc := h.service.DeleteProduct(requesterID, requesterRole, businessID, productID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Product deleted successfully", nil)
}

// 4. GET DETAIL PRODUCT (Single)
func (h *ProductHandler) GetProductDetail(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productID := strings.TrimSpace(c.Params("product_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Invalid session or unauthorized access", nil)
	}

	product, errSvc := h.service.GetProductDetail(requesterID, requesterRole, businessID, productID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Successfully retrieved product details", product)
}

// ==========================================
// HANDLER UNTUK KATALOG (NESTED)
// ==========================================

func (h *ProductHandler) GetCatalog(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))

	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	// Panggil Service khusus Katalog
	catalog, errSvc := h.service.GetCatalog(requesterID, requesterRole, businessID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Catalog retrieved successfully", catalog)
}

// 6. BULK UPDATE STATUS
func (h *ProductHandler) BulkUpdateStatus(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	// Struct inline untuk menangkap body JSON array
	var input struct {
		ProductIDs []string `json:"product_ids"`
		IsActive   bool     `json:"is_active"`
	}
	if err := c.BodyParser(&input); err != nil || len(input.ProductIDs) == 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request format or empty product_ids", nil)
	}

	if err := h.service.BulkUpdateStatus(requesterID, requesterRole, businessID, input.ProductIDs, input.IsActive); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Products status updated successfully", nil)
}

// 7. BULK DELETE
func (h *ProductHandler) BulkDelete(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input struct {
		ProductIDs []string `json:"product_ids"`
	}
	if err := c.BodyParser(&input); err != nil || len(input.ProductIDs) == 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request format or empty product_ids", nil)
	}

	if err := h.service.BulkDelete(requesterID, requesterRole, businessID, input.ProductIDs); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Products deleted successfully", nil)
}
