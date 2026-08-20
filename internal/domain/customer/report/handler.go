package report

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// CreateReport Handler Universal untuk melaporkan semua jenis resource (Review, Comment, Post, Product, Business)
func (h *Handler) CreateReport(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	// Cek apakah target_id dipassing dari URL params (jika endpoint spesifik)
	if input.TargetID == "" {
		if c.Params("review_id") != "" {
			input.TargetID = c.Params("review_id")
			input.TargetType = "review"
		} else if c.Params("comment_id") != "" {
			input.TargetID = c.Params("comment_id")
			input.TargetType = "comment"
		} else if c.Params("post_id") != "" {
			input.TargetID = c.Params("post_id")
			input.TargetType = "post"
		} else if c.Params("product_id") != "" {
			input.TargetID = c.Params("product_id")
			input.TargetType = "product"
		} else if c.Params("business_id") != "" {
			input.TargetID = c.Params("business_id")
			input.TargetType = "business"
		}
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.CreateUniversalReport(accountID, input); err != nil {
		if strings.Contains(err.Error(), "sudah pernah melaporkan") {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.DuplicateData, err.Error(), nil)
		}
		if strings.Contains(err.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
		}
		if strings.Contains(err.Error(), "wajib diisi") {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Laporan berhasil dikirim", nil)
}

// GetAllReports Mengambil daftar semua laporan dengan pagination dan filter
func (h *Handler) GetAllReports(c *fiber.Ctx) error {
	var query models.ReportFilterQuery
	if err := c.QueryParser(&query); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Parameter query tidak valid", nil)
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 {
		query.Limit = 20
	}

	reports, total, err := h.service.GetReports(query)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	totalPages := int((total + int64(query.Limit) - 1) / int64(query.Limit))

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar laporan berhasil diambil", fiber.Map{
		"reports": reports,
		"meta": fiber.Map{
			"total":        total,
			"page":         query.Page,
			"limit":        query.Limit,
			"total_pages":  totalPages,
		},
	})
}

// GetMyReports Mengambil daftar laporan yang dibuat oleh pengguna yang sedang login
func (h *Handler) GetMyReports(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	reports, total, err := h.service.GetUserReports(accountID, page, limit)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Riwayat laporan Anda berhasil diambil", fiber.Map{
		"reports": reports,
		"meta": fiber.Map{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// GetReportDetail Mengambil detail 1 laporan berdasarkan report_id
func (h *Handler) GetReportDetail(c *fiber.Ctx) error {
	reportID := strings.TrimSpace(c.Params("report_id"))
	if reportID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Report ID wajib diisi", nil)
	}

	report, err := h.service.GetReportDetail(reportID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail laporan berhasil diambil", report)
}

// DeleteReport Menghapus laporan (oleh pembuat laporan atau admin)
func (h *Handler) DeleteReport(c *fiber.Ctx) error {
	reportID := strings.TrimSpace(c.Params("report_id"))
	if reportID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Report ID wajib diisi", nil)
	}

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	role, _ := c.Locals("role").(string)

	if err := h.service.DeleteReport(reportID, accountID, role); err != nil {
		if strings.Contains(err.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
		}
		if strings.Contains(err.Error(), "tidak memiliki izin") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Laporan berhasil dihapus", nil)
}

// UpdateReportStatus Memperbarui status laporan (oleh Admin)
func (h *Handler) UpdateReportStatus(c *fiber.Ctx) error {
	reportID := strings.TrimSpace(c.Params("report_id"))
	if reportID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Report ID wajib diisi", nil)
	}

	accountID, _ := c.Locals("account_id").(string)

	var input models.UpdateReportStatusInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.UpdateReportStatus(reportID, input, accountID); err != nil {
		if strings.Contains(err.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Status laporan berhasil diperbarui", fiber.Map{
		"report_id": reportID,
		"status":    input.Status,
	})
}

// SubmitReport Handler legacy untuk kompatibilitas route bisnis
func (h *Handler) SubmitReport(c *fiber.Ctx) error {
	businessID := c.Params("business_id")
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Business ID wajib diisi", nil)
	}

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var req SubmitReportRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&req); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	err := h.service.SubmitReport(accountID, businessID, req)
	if err != nil {
		if err.Error() == "business not found" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Toko tidak ditemukan", nil)
		}
		if err.Error() == "reason is required" || err.Error() == "description is required when reason is 'Other'" {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Laporan berhasil dikirim", nil)
}
