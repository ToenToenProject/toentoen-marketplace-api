package auth

import (
	"time"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var input models.RegisterUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if input.Identifier == "" || input.Password == "" || input.Role == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Identifier, password, dan role wajib diisi", nil)
	}

	if err := h.service.Register(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Akun anda sudah dibuat, silakan verifikasi OTP", nil)
}

func (h *AuthHandler) VerifyRegisterOTP(c *fiber.Ctx) error {
	var input models.VerifyRegisterOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.VerifyRegisterOTP(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Akun berhasil diverifikasi. Silakan login.", nil)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var input models.LoginUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.Login(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedAuth, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Silakan verifikasi OTP yang telah dikirim", nil)
}

func (h *AuthHandler) VerifyLoginOTP(c *fiber.Ctx) error {
	var input models.VerifyLoginOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	token, err := h.service.VerifyLoginOTP(input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, err.Error(), nil)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
	})

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Login berhasil", nil)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var input models.ForgotPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.ForgotPassword(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP telah dikirim ke email/WA Anda", nil)
}

func (h *AuthHandler) VerifyForgotOTP(c *fiber.Ctx) error {
	var input models.VerifyForgotOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	resetToken, err := h.service.VerifyForgotOTP(input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "OTP terverifikasi. Silakan ubah kata sandi Anda", fiber.Map{"resetToken": resetToken})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var input models.ResetPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.ResetPassword(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kata sandi berhasil diubah. Silakan login.", nil)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	var input models.ChangePasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.ChangePassword(accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kata sandi Anda telah berhasil diperbarui", nil)
}

func (h *AuthHandler) RequestEmailChange(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	var input models.RequestEmailChangeInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.RequestEmailChange(accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP telah dikirim ke email baru Anda", nil)
}

func (h *AuthHandler) VerifyEmailChange(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	var input models.VerifyEmailChangeInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if err := h.service.VerifyEmailChange(accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Email Anda telah berhasil diperbarui", nil)
}

func (h *AuthHandler) CreateBusinessAdmin(c *fiber.Ctx) error {
	var input models.CreateBusinessAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if input.Identifier == "" || input.Password == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Identifier dan kata sandi wajib diisi", nil)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	ownerID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	createdAdmin, err := h.service.CreateBusinessAdmin(ownerID, input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, err.Error(), nil)
	}
	_ = createdAdmin // data tersedia tapi tidak perlu dikembalikan ke client

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Akun Business Admin berhasil dibuat", nil)
}