package account

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProfileHandler struct {
	service ProfileService
}

func NewProfileHandler(service ProfileService) *ProfileHandler {
	return &ProfileHandler{service}
}

func (h *ProfileHandler) UpdateProfileCustomer(c *fiber.Ctx) error {
	var input models.UpdateProfileCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", fiber.Map{"error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	// CARA BARU: Langsung tegaskan bahwa ini adalah string
	accountID, ok := accountIDValue.(string)
	if !ok {
		// Jika ternyata bukan string (misal JWT salah konfigurasi), tolak agar tidak error di database
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Tipe data sesi tidak valid", nil)
	}

	account, err := h.service.UpdateCustomerProfile(accountID, input)
	if err != nil {
		if err.Error() == "akun tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.UserNotFound, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Profil berhasil diperbarui", account)
}

func (h *ProfileHandler) UpdateProfileOwner(c *fiber.Ctx) error {
	var input models.UpdateProfileOwnerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", fiber.Map{"error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	// CARA BARU: Langsung tegaskan bahwa ini adalah string
	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Tipe data sesi tidak valid", nil)
	}

	account, err := h.service.UpdateOwnerProfile(accountID, input)
	if err != nil {
		if err.Error() == "akun tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.UserNotFound, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Profil berhasil diperbarui", account)
}