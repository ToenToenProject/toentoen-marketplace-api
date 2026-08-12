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

	_, err := h.service.UpdateOwnerProfile(accountID, input)
	if err != nil {
		if err.Error() == "akun tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.UserNotFound, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Profil berhasil diperbarui", nil)
}

func (h *ProfileHandler) GetMyProfile(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Tipe data sesi tidak valid", nil)
	}

	account, err := h.service.GetMyProfile(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil data profil", account)
}

func (h *ProfileHandler) RequestDeleteOTP(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	if err := h.service.RequestDeleteOTP(accountID); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "OTP penghapusan akun telah dikirim", nil)
}

func (h *ProfileHandler) VerifyDeleteOTPAndExecute(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.VerifyDeleteOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}
	if errs := utils.ValidateStruct(&input); errs != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Input tidak lengkap", errs)
	}

	if err := h.service.VerifyDeleteOTPAndExecute(accountID, input.OTP); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Akun dan seluruh jejak Anda telah berhasil dihapus secara permanen.", nil)
}

func (h *ProfileHandler) ForceDeleteAccount(c *fiber.Ctx) error {
	var input models.ForceDeleteAccountInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}
	if errs := utils.ValidateStruct(&input); errs != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Input tidak lengkap", errs)
	}

	if err := h.service.ForceDeleteAccount(input.Identifier); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.InternalServerErr, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Akun " + input.Identifier + " telah dihapus secara permanen.", nil)
}