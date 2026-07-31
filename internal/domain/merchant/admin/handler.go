package admin

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type BusinessAdminHandler struct {
	service BusinessAdminService
}

func NewBusinessAdminHandler(service BusinessAdminService) *BusinessAdminHandler {
	return &BusinessAdminHandler{service}
}

// 1. OWNER MENGUNDANG CALON ADMIN
func (h *BusinessAdminHandler) InviteAdmin(c *fiber.Ctx) error {
	ownerID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.InviteAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	// Opsional: Jika BusinessID dikirim lewat parameter URL (/business/:business_id/admins/invite)
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID != "" {
		input.BusinessID = businessID
	}

	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.BusinessID) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Email dan business_id wajib diisi", nil)
	}

	if err := h.service.InviteAdmin(ownerID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Undangan berhasil dikirim", nil)
}

// 2. CALON ADMIN MERESPONS UNDANGAN (Terima / Tolak)
func (h *BusinessAdminHandler) RespondToInvitation(c *fiber.Ctx) error {
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID wajib diisi", nil)
	}

	adminID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.RespondInvitationInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.RespondToInvitation(adminID, assignmentID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Respons undangan berhasil disimpan", nil)
}

// 3. OWNER MENGUBAH ROLE / SHIFT / MEMECAT ADMIN
func (h *BusinessAdminHandler) UpdateAdminAssignment(c *fiber.Ctx) error {
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID wajib diisi", nil)
	}

	ownerID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateAssignmentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateAdminAssignment(ownerID, assignmentID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data admin berhasil diperbarui", nil)
}

// 4. OWNER MELIHAT DAFTAR ADMIN DI TOKONYA
func (h *BusinessAdminHandler) GetAdminsByBusiness(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	ownerID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	admins, err := h.service.GetAdminsByBusiness(ownerID, businessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar admin toko berhasil diambil", admins)
}

// 5. ADMIN MELIHAT DAFTAR UNDANGAN / TEMPAT KERJANYA SENDIRI
func (h *BusinessAdminHandler) GetMyAssignments(c *fiber.Ctx) error {
	adminID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	assignments, err := h.service.GetMyAssignments(adminID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data penugasan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar penugasan berhasil diambil", assignments)
}