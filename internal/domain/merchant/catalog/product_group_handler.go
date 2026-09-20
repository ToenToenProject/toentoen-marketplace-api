package catalog

import (
	"strings"
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
	businessID := strings.TrimSpace(c.Params("business_id"))
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
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.CreateGroup(requesterID, requesterRole, businessID, input); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Grup berhasil dibuat", nil)
}

func (h *ProductGroupHandler) UpdateGroup(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productGroupID := strings.TrimSpace(c.Params("product_group_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateGroup(requesterID, requesterRole, businessID, productGroupID, input); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Grup berhasil diperbarui", nil)
}

func (h *ProductGroupHandler) ReorderGroups(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.ReorderGroupsInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ReorderGroups(requesterID, requesterRole, businessID, input); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Urutan grup berhasil diperbarui", nil)
}

func (h *ProductGroupHandler) UpdateContents(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productGroupID := strings.TrimSpace(c.Params("product_group_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateGroupContentsInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateGroupContents(requesterID, requesterRole, businessID, productGroupID, input); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Isi grup berhasil diperbarui", nil)
}

func (h *ProductGroupHandler) DeleteGroup(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	productGroupID := strings.TrimSpace(c.Params("product_group_id"))
	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeleteGroup(requesterID, requesterRole, businessID, productGroupID); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Grup berhasil dihapus", nil)
}
