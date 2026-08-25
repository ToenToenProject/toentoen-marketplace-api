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

// 1. REKRUT / INVITE ADMIN (Owner atau Supervisor)
func (h *BusinessAdminHandler) InviteAdmin(c *fiber.Ctx) error {
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	var input models.InviteAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	input.BusinessID = businessID
	if strings.TrimSpace(input.InvitationCode) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invitation Code wajib diisi", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.InviteAdmin(requesterID, businessID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Undangan / pengajuan admin berhasil diproses", nil)
}

// 2. OWNER MERESPONS APPROVAL INVITASI DARI SUPERVISOR
func (h *BusinessAdminHandler) RespondOwnerApproval(c *fiber.Ctx) error {
	ownerID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if businessID == "" || assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID dan Assignment ID wajib diisi", nil)
	}

	var input models.RespondOwnerApprovalInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.RespondOwnerApproval(ownerID, businessID, assignmentID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	msg := "Pengajuan penambahan admin telah disetujui dan undangan diteruskan ke calon admin"
	if input.Action == "reject" {
		msg = "Pengajuan penambahan admin telah ditolak"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, nil)
}

// 3. CALON ADMIN MERESPONS UNDANGAN (Accept / Reject)
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

	msg := "Anda berhasil bergabung dengan toko"
	if input.Action == "reject" {
		msg = "Undangan telah ditolak"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, nil)
}

// 4. OWNER MEMPERBARUI PERAN / SHIFT / SUPERVISOR ADMIN
func (h *BusinessAdminHandler) UpdateAdminAssignment(c *fiber.Ctx) error {
	ownerID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if businessID == "" || assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID dan Assignment ID wajib diisi", nil)
	}

	var input models.UpdateAssignmentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateAdminAssignment(ownerID, businessID, assignmentID, input); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Penugasan admin berhasil diperbarui", nil)
}

// 5. MEMBERHENTIKAN ADMIN OLEH OWNER/SPV (Grace Period 3 Hari, Read-Only)
func (h *BusinessAdminHandler) TerminateAdmin(c *fiber.Ctx) error {
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if businessID == "" || assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID dan Assignment ID wajib diisi", nil)
	}

	if err := h.service.TerminateAdmin(requesterID, businessID, assignmentID); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pemberhentian admin berhasil dijadwalkan dengan masa tenggang 3 hari (Read-Only)", nil)
}

// 6. ADMIN RESIGN MANDIRI DARI TOKO (Grace Period 3 Hari, Read-Only)
func (h *BusinessAdminHandler) ResignFromBusiness(c *fiber.Ctx) error {
	adminID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID wajib diisi", nil)
	}

	if err := h.service.ResignFromBusiness(adminID, assignmentID); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pengunduran diri berhasil diajukan dengan masa tenggang 3 hari (Read-Only)", nil)
}

// 7. BATALKAN PENGHENTIAN AKSES (Cancel Termination / Resign)
func (h *BusinessAdminHandler) CancelTermination(c *fiber.Ctx) error {
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID wajib diisi", nil)
	}

	if err := h.service.CancelTermination(requesterID, assignmentID); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Penghentian akses berhasil dibatalkan. Status admin kembali aktif normal.", nil)
}

// 8. LIHAT STRUKTUR ADMIN TREE (Owner -> Supervisors -> Staff)
func (h *BusinessAdminHandler) GetAdminTree(c *fiber.Ctx) error {
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	tree, err := h.service.GetAdminTree(requesterID, businessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Struktur pohon admin berhasil diambil", tree)
}

// 9. OWNER MELIHAT DAFTAR ADMIN DI SELURUH TOKONYA
func (h *BusinessAdminHandler) GetAdminsByOwner(c *fiber.Ctx) error {
	ownerID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	adminsTree, err := h.service.GetAdminsByOwner(ownerID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar admin seluruh toko berhasil diambil", adminsTree)
}

// 10. ADMIN MELIHAT DAFTAR TEMPAT KERJA / UNDANGANNYA
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

// 11. LIHAT PERMISSION ADMIN (Owner atau Supervisor)
// 11. GET PERMISSION ADMIN (Owner atau Supervisor)
// GET /business/:business_id/admins/:assignment_id/permissions ATAU /admins/:assignment_id/permissions
func (h *BusinessAdminHandler) GetPermissions(c *fiber.Ctx) error {
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID wajib diisi", nil)
	}

	result, err := h.service.GetPermissions(requesterID, businessID, assignmentID)
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		} else if strings.Contains(err.Error(), "tidak ditemukan") {
			status = fiber.StatusNotFound
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Izin admin berhasil diambil", result)
}

// 12. UPDATE PERMISSION ADMIN (Owner atau Supervisor)
// PUT/PATCH /business/:business_id/admins/:assignment_id/permissions ATAU /admins/:assignment_id/permissions
func (h *BusinessAdminHandler) UpdatePermissions(c *fiber.Ctx) error {
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	assignmentID := strings.TrimSpace(c.Params("assignment_id"))
	if assignmentID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Assignment ID wajib diisi", nil)
	}

	var input models.UpdatePermissionsInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	// Dukung format payload flat {"change_product_image": true, ...} jika tidak dibungkus {"permissions": {...}}
	if len(input.Permissions) == 0 {
		var flatPerms models.AdminPermissions
		if err := c.BodyParser(&flatPerms); err == nil && len(flatPerms) > 0 {
			input.Permissions = flatPerms
		} else {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'permissions' wajib diisi", nil)
		}
	}

	result, err := h.service.UpdatePermissions(requesterID, businessID, assignmentID, input)
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "forbidden") {
			status = fiber.StatusForbidden
		} else if strings.Contains(err.Error(), "tidak ditemukan") {
			status = fiber.StatusNotFound
		}
		return utils.SendResponse(c, status, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Izin admin berhasil diperbarui", result)
}