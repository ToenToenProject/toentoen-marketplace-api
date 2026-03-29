package handlers

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type BusinessHandler struct {
	service services.BusinessService
}

func NewBusinessHandler(service services.BusinessService) *BusinessHandler {
	return &BusinessHandler{service}
}

func (h *BusinessHandler) CreateBusiness(c *fiber.Ctx) error {
	var input models.CreateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", fiber.Map{"error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	createdBusiness, err := h.service.CreateBusiness(ownerID, input)
	if err != nil {
		if err.Error() == "nama bisnis sudah terdaftar di akun Anda" {
			return utils.SendResponse(c, fiber.StatusConflict, constants.DuplicateData, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal membuat profil bisnis", fiber.Map{"error": err.Error()})
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Profil bisnis berhasil dibuat dan disimpan", createdBusiness)
}

func (h *BusinessHandler) GetMyBusinesses(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	businesses, err := h.service.GetMyBusinesses(ownerID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data bisnis", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar bisnis", businesses)
}

func (h *BusinessHandler) UpdateBusiness(c *fiber.Ctx) error {
	businessID := c.Params("id")

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	var input models.UpdateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	updatedBusiness, err := h.service.UpdateBusiness(ownerID, businessID, input)
	if err != nil {
		if err.Error() == "handle sudah digunakan, silakan pilih yang lain" {
			return utils.SendResponse(c, fiber.StatusConflict, constants.HandleTaken, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.ForbiddenAccess, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Bisnis berhasil diperbarui", updatedBusiness)
}

func (h *BusinessHandler) DeleteBusiness(c *fiber.Ctx) error {
	businessID := c.Params("id")

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	if err := h.service.DeleteBusiness(ownerID, businessID); err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Bisnis berhasil dihapus", nil)
}
