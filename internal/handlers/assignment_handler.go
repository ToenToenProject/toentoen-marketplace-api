package handlers

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type AssignmentHandler struct {
	service services.AssignmentService
}

func NewAssignmentHandler(service services.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{service}
}

func (h *AssignmentHandler) AssignBusinessAdmin(c *fiber.Ctx) error {
	var input models.AssignBusinessAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", fiber.Map{"error": err.Error()})
	}

	if input.BusinessAdminID == "" || input.BusinessID == "" || input.Role == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "BusinessAdminID, businessID, and role must be filled", nil)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid", nil)
	}

	ownerID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	assignmentDetails, err := h.service.AssignBusinessAdminToBusiness(ownerID, input)
	if err != nil {
		if err.Error() == "admin has been placed in this business, please use the update feature to change working hours or role" {
			return utils.SendResponse(c, fiber.StatusConflict, constants.DuplicateData, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Business admin successfully placed in the business", assignmentDetails)
}

func (h *AssignmentHandler) GetMyAdmins(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid", nil)
	}

	ownerID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	admins, err := h.service.GetMyAdmins(ownerID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to retrieve business admin data", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Successfully retrieved admin data", admins)
}

func (h *AssignmentHandler) UpdateAssignment(c *fiber.Ctx) error {
	var input models.UpdateAssignmentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", fiber.Map{"error": err.Error()})
	}

	adminID := c.Params("admin_id")
	businessID := c.Params("business_id")

	accountIDValue := c.Locals("account_id")
	roleValue := c.Locals("role") 

	if accountIDValue == nil || roleValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid", nil)
	}

	requesterID, ok1 := accountIDValue.(string)
	requesterRole, ok2 := roleValue.(string)
	if !ok1 || !ok2 {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses data sesi", nil)
	}

	updatedAssignment, err := h.service.UpdateAssignment(requesterID, requesterRole, adminID, businessID, input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data berhasil diperbarui", updatedAssignment)
}

func (h *AssignmentHandler) DeleteBusinessAdmin(c *fiber.Ctx) error {
	adminID := c.Params("admin_id")

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid", nil)
	}

	ownerID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	if err := h.service.DeleteBusinessAdmin(ownerID, adminID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Admin berhasil dihapus dari sistem secara permanen", nil)
}