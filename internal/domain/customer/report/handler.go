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
