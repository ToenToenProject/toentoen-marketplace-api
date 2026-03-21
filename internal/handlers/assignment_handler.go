package handlers

import (
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"

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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	if input.BusinessAdminID == 0 || input.BusinessID == 0 || input.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "BusinessAdminID, businessID, and role must be filled",
		})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Session is not valid",
		})
	}

	var ownerID int64
	switch v := accountIDValue.(type) {
	case int64:
		ownerID = v

	case float64:
		ownerID = int64(v)
	}

	assignmentDetails, err := h.service.AssignBusinessAdminToBusiness(ownerID, input)
	if err != nil {
		if err.Error() == "admin has been placed in this business, please use the update feature to change working hours or role" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Business admin successfully placed in the business",
		"data": assignmentDetails,
	})
}

func (h *AssignmentHandler) GetMyAdmins(c *fiber.Ctx) error {
	ownerID := c.Locals("account_id").(int64)

	admins, err := h.service.GetMyAdmins(ownerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to retrieve business admin data"})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Successfully retrieved admin data",
		"data": admins,
	})
}

func (h *AssignmentHandler) UpdateAssignment(c *fiber.Ctx) error {
	var input models.UpdateAssignmentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	adminID, _ := c.ParamsInt("admin_id")
	businessID, _ := c.ParamsInt("business_id")
	ownerID, _ := c.Locals("account_id").(int64)

	updatedAssignment, err := h.service.UpdateAssignment(ownerID, int64(adminID), int64(businessID), input);
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Admin placement data successfully updated",
		"data": updatedAssignment,
	})
}

func (h *AssignmentHandler) DeleteBusinessAdmin(c *fiber.Ctx) error {
	adminID, _ := c.ParamsInt("admin_id")
	ownerID := c.Locals("account_id").(int64)

	if err := h.service.DeleteBusinessAdmin(ownerID, int64(adminID)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Admin berhasil dihapus dari sistem secara permanen"})
}