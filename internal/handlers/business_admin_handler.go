package handlers

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type BusinessAdminHandler struct {
	service services.BusinessAdminService
}

func NewBusinessAdminHandler(service services.BusinessAdminService) *BusinessAdminHandler {
	return &BusinessAdminHandler{service}
}

// 1. OWNER MENGUNDANG CALON ADMIN
func (h *BusinessAdminHandler) InviteAdmin(c *fiber.Ctx) error {
	// Ambil requester ID dari token (Berperan sebagai Owner)
	ownerID, _, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input models.InviteAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	// Opsional: Jika BusinessID dikirim lewat parameter URL (/business/:business_id/admins/invite)
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID != "" {
		input.BusinessID = businessID
	}

	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.BusinessID) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Email and business_id are required", nil)
	}

	if err := h.service.InviteAdmin(ownerID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Invitation sent successfully", nil)
}

// 2. CALON ADMIN MERESPONS UNDANGAN (Terima / Tolak)
func (h *BusinessAdminHandler) RespondToInvitation(c *fiber.Ctx) error {
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID is required", nil)
	}

	// Ambil requester ID dari token (Berperan sebagai Calon Admin)
	adminID, _, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input models.RespondInvitationInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	if err := h.service.RespondToInvitation(adminID, assignmentID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Invitation responded successfully", nil)
}

// 3. OWNER MENGUBAH ROLE / SHIFT / MEMECAT ADMIN
func (h *BusinessAdminHandler) UpdateAdminAssignment(c *fiber.Ctx) error {
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID is required", nil)
	}

	// Ambil requester ID dari token (Berperan sebagai Owner)
	ownerID, _, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	var input models.UpdateAssignmentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	if err := h.service.UpdateAdminAssignment(ownerID, assignmentID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Admin assignment updated successfully", nil)
}

// 4. OWNER MELIHAT DAFTAR ADMIN DI TOKONYA
func (h *BusinessAdminHandler) GetAdminsByBusiness(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID is required", nil)
	}

	// Ambil requester ID dari token (Berperan sebagai Owner)
	ownerID, _, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	admins, err := h.service.GetAdminsByBusiness(ownerID, businessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Successfully fetched business admins", admins)
}

// 5. ADMIN MELIHAT DAFTAR UNDANGAN / TEMPAT KERJANYA SENDIRI
func (h *BusinessAdminHandler) GetMyAssignments(c *fiber.Ctx) error {
	// Ambil requester ID dari token (Berperan sebagai Admin)
	adminID, _, err := getRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	assignments, err := h.service.GetMyAssignments(adminID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to fetch assignments", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Successfully fetched my assignments", assignments)
}