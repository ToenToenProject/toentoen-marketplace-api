package handlers

import (
	"strconv"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type BusinessHandler struct {
	service services.BusinessService
}

func NewBusinessHandler(service services.BusinessService) *BusinessHandler {
	return &BusinessHandler{service}
}

func (h *BusinessHandler) CreateBusiness(c *fiber.Ctx) error {
	// 1. Parsing Body JSON
	var input models.CreateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Format request tidak valid",
			"error":   err.Error(),
		})
	}

	// 2. Ambil Account ID (Owner) dari context/JWT yang login
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Sesi tidak valid atau Anda belum login"})
	}

	var ownerID int64
	switch v := accountIDValue.(type) {
	case int64:
		ownerID = v
	case float64:
		ownerID = int64(v)
	}

	// 3. Panggil Service
	if err := h.service.CreateBusiness(ownerID, input); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal membuat profil bisnis",
			"error":   err.Error(),
		})
	}

	// 4. Return Success Response
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Profil bisnis berhasil dibuat dan disimpan",
	})
}

func (h *BusinessHandler) GetMyBusinesses(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Sesi tidak valid"})
	}

	var ownerID int64
	switch v := accountIDValue.(type) {
	case int64: ownerID = v
	case float64: ownerID = int64(v)
	}

	businesses, err := h.service.GetMyBusinesses(ownerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Gagal mengambil data bisnis"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil daftar bisnis",
		"data":    businesses,
	})
}

func (h *BusinessHandler) UpdateBusiness(c *fiber.Ctx) error {
	businessIDStr := c.Params("id")
	businessID, err := strconv.ParseInt(businessIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Format ID Bisnis tidak valid"})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Sesi tidak valid"})
	}

	var ownerID int64
	switch v := accountIDValue.(type) {
	case int64: ownerID = v
	case float64: ownerID = int64(v)
	}

	var input models.UpdateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Format request tidak valid"})
	}

	updatedBusiness, err := h.service.UpdateBusiness(ownerID, businessID, input)
	if err != nil {
		// Pembedaan error conflict (409) dan error lain (400/404)
		if err.Error() == "handle sudah digunakan, silakan pilih yang lain" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Bisnis berhasil diperbarui",
		"data":    updatedBusiness,
	})
}

func (h *BusinessHandler) DeleteBusiness(c *fiber.Ctx) error {
	businessIDStr := c.Params("id")
	businessID, err := strconv.ParseInt(businessIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Format ID Bisnis tidak valid"})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Sesi tidak valid"})
	}

	var ownerID int64
	switch v := accountIDValue.(type) {
	case int64: ownerID = v
	case float64: ownerID = int64(v)
	}

	if err := h.service.DeleteBusiness(ownerID, businessID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Bisnis berhasil dihapus",
	})
}