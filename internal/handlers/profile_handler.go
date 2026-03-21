package handlers

import (
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type ProfileHandler struct {
	service services.ProfileService
}

func NewProfileHandler(service services.ProfileService) *ProfileHandler {
	return &ProfileHandler{service}
}

func (h *ProfileHandler) UpdateProfileUser(c *fiber.Ctx) error {
	var input models.UpdateProfileUserInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Format request tidak valid", "error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Akses ditolak, sesi tidak valid"})
	}

	var accountID int64
	switch v := accountIDValue.(type) {
	case int64:
		accountID = v
	case float64:
		accountID = int64(v)
	}

	account, user, err := h.service.UpdateUserProfile(accountID, input)
	if err != nil {
		if err.Error() == "akun tidak ditemukan" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Profil berhasil diperbarui",
		"data": fiber.Map{
			"account_data": account,
			"user_data":    user,
		},
	})
}

func (h *ProfileHandler) UpdateProfileOwner(c *fiber.Ctx) error {
	var input models.UpdateProfileOwnerInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Format request tidak valid", "error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Akses ditolak, sesi tidak valid"})
	}

	var accountID int64
	switch v := accountIDValue.(type) {
	case int64:
		accountID = v
	case float64:
		accountID = int64(v)
	}

	account, owner, err := h.service.UpdateOwnerProfile(accountID, input)
	if err != nil {
		if err.Error() == "akun tidak ditemukan" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Profil berhasil diperbarui",
		"data": fiber.Map{
			"account_data": account,
			"owner_data":   owner,
		},
	})
}