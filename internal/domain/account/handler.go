package account

import (
	"strings"
	"time"
	"toentoen-backend/internal/config"
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
	if errs := utils.ValidateStruct(&input); errs != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errs), errs)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Tipe data sesi tidak valid", nil)
	}

	updatedAccount, err := h.service.UpdateCustomerProfile(accountID, input)
	if err != nil {
		return utils.HandleServiceError(c, err, "Gagal memperbarui profil customer")
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Profil berhasil diperbarui", updatedAccount)
}

func (h *ProfileHandler) UpdateProfileOwner(c *fiber.Ctx) error {
	var input models.UpdateProfileOwnerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", fiber.Map{"error": err.Error()})
	}
	if errs := utils.ValidateStruct(&input); errs != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errs), errs)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Tipe data sesi tidak valid", nil)
	}

	updatedAccount, err := h.service.UpdateOwnerProfile(accountID, input)
	if err != nil {
		return utils.HandleServiceError(c, err, "Gagal memperbarui profil owner")
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Profil berhasil diperbarui", updatedAccount)
}

func (h *ProfileHandler) UpdateProfileBusinessAdmin(c *fiber.Ctx) error {
	var input models.UpdateBusinessAdminKYCInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", fiber.Map{"error": err.Error()})
	}
	if errs := utils.ValidateStruct(&input); errs != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errs), errs)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Tipe data sesi tidak valid", nil)
	}

	updatedAccount, err := h.service.UpdateBusinessAdminProfile(accountID, input)
	if err != nil {
		return utils.HandleServiceError(c, err, "Gagal memperbarui profil admin")
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Profil admin berhasil diperbarui", updatedAccount)
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
		if err.Error() == "akun tidak ditemukan" {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Akun tidak ditemukan atau telah dihapus, silakan login kembali", nil)
		}
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
		if err.Error() == "Akun tidak ditemukan" || strings.Contains(strings.ToLower(err.Error()), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP penghapusan akun telah dikirim ke email terdaftar Anda", nil)
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

	if err := h.service.VerifyDeleteOTPAndExecute(accountID, strings.TrimSpace(input.OTP)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "otp") {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.InternalServerErr, err.Error(), nil)
	}

	// Clear session cookie upon account deletion.
	isProduction := config.Envs != nil && config.Envs.AppEnv == "production"
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-24 * time.Hour),
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: "Lax",
	})

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Akun dan seluruh jejak Anda telah berhasil dihapus secara permanen.", nil)
}

func (h *ProfileHandler) GetKYCStatus(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	res, err := h.service.GetKYCStatus(accountID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Status KYC berhasil diambil", res)
}

func (h *ProfileHandler) SubmitKYC(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.SubmitKYCInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}
	if errs := utils.ValidateStruct(&input); errs != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errs), errs)
	}

	if err := h.service.SubmitKYC(accountID, input); err != nil {
		return utils.HandleServiceError(c, err, "Gagal mengajukan verifikasi KYC")
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pengajuan verifikasi KYC berhasil dikirimkan dan sedang dalam peninjauan", nil)
}

func (h *ProfileHandler) SubmitConsent(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	var input models.ConsentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	updatedAccount, err := h.service.UpdateConsent(accountID, input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Persetujuan Syarat & Ketentuan serta Kebijakan Privasi berhasil disimpan", updatedAccount)
}

func (h *ProfileHandler) RegisterDeviceToken(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || strings.TrimSpace(accountID) == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	var input models.DeviceTokenInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", nil)
	}

	if strings.TrimSpace(input.DeviceToken) == "" && strings.TrimSpace(input.FCMToken) == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "device_token wajib diisi", nil)
	}

	if err := h.service.RegisterDeviceToken(accountID, input); err != nil {
		return utils.HandleServiceError(c, err, "Gagal mendaftarkan token perangkat")
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Device token berhasil didaftarkan", nil)
}

func (h *ProfileHandler) DeleteDeviceToken(c *fiber.Ctx) error {
	accountID, ok := c.Locals("account_id").(string)
	if !ok || strings.TrimSpace(accountID) == "" {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	var input models.DeleteDeviceTokenInput
	_ = c.BodyParser(&input) // optional body

	if err := h.service.DeleteDeviceToken(accountID, input); err != nil {
		return utils.HandleServiceError(c, err, "Gagal mencabut token perangkat")
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Device token berhasil dihapus", nil)
}
