package catalog

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductGroupHandler struct {
	service ProductGroupService
}

func NewProductGroupHandler(s ProductGroupService) *ProductGroupHandler {
	return &ProductGroupHandler{s}
}

func (h *ProductGroupHandler) GetGroups(c *fiber.Ctx) error {
	// Tangkap :business_id dari URL
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	// Panggil service
	groups, err := h.service.GetGroups(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar grup berhasil diambil", groups)
}

func (h *ProductGroupHandler) CreateGroup(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	var input models.CreateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.CreateGroup(businessID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Grup berhasil dibuat", nil)
}

func (h *ProductGroupHandler) UpdateGroup(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	productGroupID := c.Params("product_group_id")
	var input models.UpdateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateGroup(businessID, productGroupID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Grup berhasil diperbarui", nil)
}

func (h *ProductGroupHandler) ReorderGroups(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	
	var input models.ReorderGroupsInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
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
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateGroupContents(businessID, productGroupID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Isi grup berhasil diperbarui", nil)
}

func (h *ProductGroupHandler) DeleteGroup(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	productGroupID := c.Params("product_group_id")

	if err := h.service.DeleteGroup(businessID, productGroupID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Grup berhasil dihapus", nil)
}
