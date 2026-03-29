package handlers

import (
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/utils"

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
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format request tidak valid", fiber.Map{"error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Akses ditolak, sesi tidak valid", nil)
	}

	var accountID int64
	switch v := accountIDValue.(type) {
	case int64:
		accountID = v
	case float64:
		accountID = int64(v)
	}

	account, err := h.service.UpdateUserProfile(accountID, input)
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

	var accountID int64
	switch v := accountIDValue.(type) {
	case int64:
		accountID = v
	case float64:
		accountID = int64(v)
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