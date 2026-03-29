package handlers

import (
	"time"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	service services.AuthService
}

func NewAuthHandler(service services.AuthService) *AuthHandler {
	return &AuthHandler{service}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var input models.RegisterUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	if input.Identifier == "" || input.Password == "" || input.Role == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Identifier, password, and role are required", nil)
	}

	if err := h.service.Register(input); err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to register account", fiber.Map{"error": err.Error()})
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Account registered successfully, please login to continue", nil)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var input models.LoginUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	if err := h.service.Login(input); err != nil {
		switch err.Error() {
		case "account not found", "invalid credentials":
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Email/phone or password incorrect", nil)

		case "google account":
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.AuthThirdParty, "Please login using Google", nil)

		default:
			return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Login failed", fiber.Map{"error": err.Error()})
		}
	}

	responseData := fiber.Map{
		"nextStep":  "OTP sent to your email/WA, please verify to continue",
		"identifier": input.Identifier,
	}
	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Go verify your otp", responseData)
}

func (h *AuthHandler) VerifyLoginOTP(c *fiber.Ctx) error {
	var input models.VerifyLoginOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request body", nil)
	}

	account, token, err := h.service.VerifyLoginOTP(input)
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

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Login successful", account)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var input models.ForgotPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", nil)
	}

	if err := h.service.ForgotPassword(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "OTP sent to your email/WA", fiber.Map{"identifier": input.Identifier})
}

func (h *AuthHandler) VerifyForgotOTP(c *fiber.Ctx) error {
	var input models.VerifyForgotOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", nil)
	}

	resetToken, err := h.service.VerifyForgotOTP(input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "OTP verified. Please change your password", fiber.Map{"resetToken": resetToken})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var input models.ResetPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", nil)
	}

	if err := h.service.ResetPassword(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Password changed successfully. Please login.", nil)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var input models.ChangePasswordInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", nil)
	}

	if err := h.service.ChangePassword(input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Your password has been changed", nil)
}

func (h *AuthHandler) CreateBusinessAdmin(c *fiber.Ctx) error {
	var input models.CreateBusinessAdminInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Invalid request", nil)
	}

	if input.Identifier == "" || input.Password == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Identifier and password are required", nil)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "invalid credentials", nil)
	}

	ownerID, ok := accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID sesi pengguna", nil)
	}

	createdAdmin, err := h.service.CreateBusinessAdmin(ownerID, input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to create business admin", fiber.Map{"error": err.Error()})
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Business admin created successfully", createdAdmin)
}