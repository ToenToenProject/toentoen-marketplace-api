package auth

import (
	"errors"
	"strings"
	"time"
	"toentoen-backend/internal/config"
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

func (h *AuthHandler) CheckReferralCode(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		var body models.CheckReferralInput
		if err := c.BodyParser(&body); err == nil {
			code = body.Code
		}
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Kode referral wajib diisi", nil)
	}

	res, err := h.service.CheckReferralCode(code)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, res.Message, res)
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var input models.RegisterUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.Register(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Kode verifikasi OTP telah dikirimkan ke email Anda, silakan verifikasi untuk menyelesaikan pendaftaran", nil)
}

func (h *AuthHandler) VerifyRegisterOTP(c *fiber.Ctx) error {
	var input models.VerifyRegisterOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	token, err := h.service.VerifyRegisterOTP(input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	isProduction := config.Envs != nil && config.Envs.AppEnv == "production"
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: "Lax",
	})

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Pendaftaran berhasil dan Anda telah otomatis masuk", nil)
}

func (h *AuthHandler) SendEmailOTP(c *fiber.Ctx) error {
	var input models.SendEmailOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.SendEmailOTP(input.Email, input.Role); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP verifikasi berhasil dikirimkan ke email Anda", nil)
}

func (h *AuthHandler) VerifyEmailOTP(c *fiber.Ctx) error {
	var input models.VerifyEmailOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.VerifyEmailOTP(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Email berhasil diverifikasi.", nil)
}

func (h *AuthHandler) SendPhoneOTP(c *fiber.Ctx) error {
	var input models.SendPhoneOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.SendPhoneOTP(input.Phone); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP berhasil dikirimkan", nil)
}

func (h *AuthHandler) VerifyPhoneOTP(c *fiber.Ctx) error {
	var input models.VerifyPhoneOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.VerifyPhoneOTP(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Nomor telepon berhasil diverifikasi", nil)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var input models.LoginUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if input.ExpectedRole == "" && input.Role != "" {
		input.ExpectedRole = input.Role
	}
	if input.ExpectedRole == "" {
		input.ExpectedRole = c.Query("expected_role")
		if input.ExpectedRole == "" {
			input.ExpectedRole = c.Query("role")
		}
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.Login(input); err != nil {
		var mrErr *MultiRoleRequiredError
		if errors.As(err, &mrErr) {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"responseCode": "MR",
				"status":       "multi_role",
				"message":      mrErr.Error(),
				"data": fiber.Map{
					"roles": mrErr.Roles,
					"email": mrErr.Email,
				},
			})
		}
		var roleErr *RoleMismatchError
		if errors.As(err, &roleErr) {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, roleErr.Error(), fiber.Map{
				"actual_role":     roleErr.ActualRole,
				"expected_role":   roleErr.ExpectedRole,
				"redirect_portal": roleErr.RedirectPortal,
			})
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedAuth, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Silakan verifikasi OTP yang telah dikirim", nil)
}

func (h *AuthHandler) VerifyLoginOTP(c *fiber.Ctx) error {
	var input models.VerifyLoginOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if input.ExpectedRole == "" && input.Role != "" {
		input.ExpectedRole = input.Role
	}
	if input.ExpectedRole == "" {
		input.ExpectedRole = c.Query("expected_role")
		if input.ExpectedRole == "" {
			input.ExpectedRole = c.Query("role")
		}
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	token, account, err := h.service.VerifyLoginOTP(input)
	if err != nil {
		var roleErr *RoleMismatchError
		if errors.As(err, &roleErr) {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, roleErr.Error(), fiber.Map{
				"actual_role":     roleErr.ActualRole,
				"expected_role":   roleErr.ExpectedRole,
				"redirect_portal": roleErr.RedirectPortal,
			})
		}
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, err.Error(), nil)
	}

	isProduction := config.Envs != nil && config.Envs.AppEnv == "production"
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: "Lax",
	})

	data := fiber.Map{
		"token": token,
	}
	if account != nil {
		data["account_id"] = account.AccountID
		data["accountId"] = account.AccountID
		data["role"] = account.Role
		if account.Email != nil {
			data["email"] = *account.Email
		}
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Login berhasil", data)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var input models.ForgotPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.ForgotPassword(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP telah dikirim ke email Anda", nil)
}

func (h *AuthHandler) VerifyForgotOTP(c *fiber.Ctx) error {
	var input models.VerifyForgotOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
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
	if input.Role == "" {
		input.Role = c.Query("role")
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
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

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
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

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
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

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.VerifyEmailChange(accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Email Anda telah berhasil diperbarui", nil)
}

func (h *AuthHandler) RequestPhoneChange(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	var input models.RequestPhoneChangeInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.RequestPhoneChange(accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP telah dikirim ke nomor WhatsApp baru Anda", nil)
}

func (h *AuthHandler) VerifyPhoneChange(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	accountID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	var input models.VerifyPhoneChangeInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if err := h.service.VerifyPhoneChange(accountID, input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Nomor telepon Anda telah berhasil diperbarui", nil)
}

func (h *AuthHandler) CreateBusinessAdmin(c *fiber.Ctx) error {
	var input models.CreateBusinessAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
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