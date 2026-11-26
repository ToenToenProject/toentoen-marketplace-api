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

// CreateReport handles universal moderation reports across all resource types (reviews, comments, posts, products, businesses).
func (h *Handler) CreateReport(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreateReportInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

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

// GetAllReports retrieves paginated moderation reports with optional filters.
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

// GetMyReports retrieves reports filed by the authenticated user.
func (h *Handler) GetMyReports(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 20)
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

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

// GetReportDetail retrieves report details by report_id (restricted to report owner or admin).
func (h *Handler) GetReportDetail(c *fiber.Ctx) error {
	reportID := strings.TrimSpace(c.Params("report_id"))
	if reportID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Report ID wajib diisi", nil)
	}

	accountID, ok := c.Locals("account_id").(string)
	if !ok || accountID == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}
	role, _ := c.Locals("role").(string)

	report, err := h.service.GetReportDetail(reportID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	isOwner := strings.EqualFold(strings.TrimSpace(report.AccountID), strings.TrimSpace(accountID))
	isAdmin := strings.EqualFold(role, "platform_admin") || strings.EqualFold(role, "super_admin") ||
		strings.EqualFold(role, "admin") || strings.EqualFold(role, "superadmin")
	if !isOwner && !isAdmin {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, "Anda tidak memiliki izin untuk melihat laporan ini", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail laporan berhasil diambil", report)
}

// DeleteReport removes a report record.
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

// UpdateReportStatus updates moderation report status (admin only).
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

// SubmitReport handles legacy business report endpoint.
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

// GetReportCategories mengembalikan daftar master jenis/kategori laporan beserta sub-kategorinya.
// Endpoint: GET /api/v1/reports/categories?target_type=business
func (h *Handler) GetReportCategories(c *fiber.Ctx) error {
	targetType := c.Query("target_type")
	categories := models.GetReportCategories(targetType)
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar jenis laporan berhasil diambil", categories)
}
