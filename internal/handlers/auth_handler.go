package handlers

import (
	"time"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"

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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	if input.Identifier == "" || input.Password == "" || input.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Identifier, password, and role are required",
		})
	}

	if err := h.service.Register(input); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to register account",
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Account registered successfully, please login to continue",
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var input models.LoginUnifiedInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	if err := h.service.Login(input); err != nil {
		switch err.Error() {
		case "account not found", "invalid credentials":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Email/phone or password incorrect",
			})

		case "google account":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Please login using Google",
			})

		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Go verify your otp",
		"next_step": "OTP sent to your email/WA, please verify to continue",
		"identifier": input.Identifier,
	})
}

func (h *AuthHandler) VerifyLoginOTP(c *fiber.Ctx) error {
	var input models.VerifyLoginOTPInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	account, token, err := h.service.VerifyLoginOTP(input)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	c.Cookie(&fiber.Cookie{
		Name: "auth_token",
		Value: token,
		Path: "/",
		Expires: time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"data": account,
	})
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var input models.ForgotPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	if err := h.service.ForgotPassword(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message":err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "OTP sent to your email/WA",
		"identifier": input.Identifier,
	})
}

func(h *AuthHandler) VerifyForgotOTP(c *fiber.Ctx) error {
	var input models.VerifyForgotOTPInput
	if err := c.BodyParser(&input);err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	resetToken, err := h.service.VerifyForgotOTP(input)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "OTP verified. Please change your password",
		"reset_token": resetToken,
	})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var input models.ResetPasswordInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	if err := h.service.ResetPassword(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password changed successfully. Please login.",
	})
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var input models.ChangePasswordInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	if err := h.service.ChangePassword(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Your password has been changed",
	})
}

func (h *AuthHandler) CreateBusinessAdmin(c *fiber.Ctx) error {
	var input models.CreateBusinessAdminInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	if input.Identifier == "" || input.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Identifier and password are required",
		})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "invalid credentials",
		})
	}

	var ownerID int64
	switch v := accountIDValue.(type){
	case int64:
		ownerID = v

	case float64:
		ownerID = int64(v)
	}

	if err := h.service.CreateBusinessAdmin(ownerID, input); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create business admin",
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Business admin created successfully",
	})
}