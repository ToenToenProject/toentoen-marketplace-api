package catalog

import (
	"strings"

	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PackageHandler struct {
	service PackageService
}

func NewPackageHandler(service PackageService) *PackageHandler {
	return &PackageHandler{service}
}

// 1. CALCULATE PACKAGE PRICE (HELPER PRE-CALCULATE)
func (h *PackageHandler) CalculatePackagePrice(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CalculatePackageInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	res, errSvc := h.service.CalculatePackagePrice(requesterID, requesterRole, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kalkulasi harga paket berhasil", res)
}

// 2. CREATE PACKAGE
func (h *PackageHandler) CreatePackage(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.CreatePackageInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	res, errSvc := h.service.CreatePackage(requesterID, requesterRole, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Paket produk berhasil dibuat", res)
}

// 3. GET ALL PACKAGES
func (h *PackageHandler) GetPackages(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis wajib diisi", nil)
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	res, errSvc := h.service.GetPackages(requesterID, requesterRole, businessID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar paket produk berhasil diambil", res)
}

// 4. GET PACKAGE DETAIL
func (h *PackageHandler) GetPackageDetail(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	packageID := strings.TrimSpace(c.Params("package_id"))
	if packageID == "" {
		packageID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" || packageID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis dan ID paket wajib diisi", nil)
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	res, errSvc := h.service.GetPackageDetail(requesterID, requesterRole, businessID, packageID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail paket produk berhasil diambil", res)
}

// 5. UPDATE PACKAGE
func (h *PackageHandler) UpdatePackage(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	packageID := strings.TrimSpace(c.Params("package_id"))
	if packageID == "" {
		packageID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" || packageID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis dan ID paket wajib diisi", nil)
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdatePackageInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	res, errSvc := h.service.UpdatePackage(requesterID, requesterRole, businessID, packageID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Paket produk berhasil diperbarui", res)
}

// 6. DELETE PACKAGE
func (h *PackageHandler) DeletePackage(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	packageID := strings.TrimSpace(c.Params("package_id"))
	if packageID == "" {
		packageID = strings.TrimSpace(c.Params("id"))
	}
	if businessID == "" || packageID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "ID bisnis dan ID paket wajib diisi", nil)
	}

	requesterID, requesterRole, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	if err := h.service.DeletePackage(requesterID, requesterRole, businessID, packageID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Paket produk berhasil dihapus", nil)
}
