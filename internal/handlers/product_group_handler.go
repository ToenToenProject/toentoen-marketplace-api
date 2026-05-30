package handlers

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductGroupHandler struct {
	service services.ProductGroupService
}

func NewProductGroupHandler(s services.ProductGroupService) *ProductGroupHandler {
	return &ProductGroupHandler{s}
}

func (h *ProductGroupHandler) GetGroups(c *fiber.Ctx) error {
	// Tangkap :business_id dari URL
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID is required", nil)
	}

	// Panggil service
	groups, err := h.service.GetGroups(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.FailedValidation, err.Error(), nil)
	}

	// Kembalikan response sukses sesuai format responseCode Anda ("00" atau SUCCESS)
	return utils.SendResponse(c, fiber.StatusOK, "00", "Successfully retrieved groups", groups)
}

func (h *ProductGroupHandler) CreateGroup(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	var input models.CreateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid input", nil)
	}

	if err := h.service.CreateGroup(businessID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Group created successfully", nil)
}

func (h *ProductGroupHandler) UpdateGroup(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	productGroupID := c.Params("product_group_id")
	var input models.UpdateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid input", nil)
	}

	if err := h.service.UpdateGroup(businessID, productGroupID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Group updated successfully", nil)
}

func (h *ProductGroupHandler) ReorderGroups(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	
	var input models.ReorderGroupsInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format input tidak valid", nil)
	}

	if err := h.service.ReorderGroups(businessID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Urutan grup berhasil diperbarui", nil)
}

func (h *ProductGroupHandler) UpdateContents(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	productGroupID := c.Params("product_group_id")
	var input models.UpdateGroupContentsInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid input", nil)
	}

	if err := h.service.UpdateGroupContents(businessID, productGroupID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Group contents updated successfully", nil)
}

func (h *ProductGroupHandler) DeleteGroup(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	productGroupID := c.Params("product_group_id")

	if err := h.service.DeleteGroup(businessID, productGroupID); err != nil {
		// Kirim HTTP 400 Bad Request jika gagal (misal: grup tidak kosong)
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Group deleted successfully", nil)
}
