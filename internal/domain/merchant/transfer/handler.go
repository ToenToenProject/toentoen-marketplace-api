package transfer

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type BusinessTransferHandler struct {
	service BusinessTransferService
}

func NewBusinessTransferHandler(service BusinessTransferService) *BusinessTransferHandler {
	return &BusinessTransferHandler{service: service}
}

// CheckTarget memvalidasi dan menampilkan preview profil calon pemilik baru
func (h *BusinessTransferHandler) CheckTarget(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "business_id wajib diisi", nil)
	}

	var input models.CheckTransferTargetInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	preview, errSvc := h.service.CheckTarget(accountID, businessID, input.TargetIdentifier)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data calon pemilik baru ditemukan", preview)
}

// InitiateTransfer mengirim kode OTP ke pemilik lama untuk otorisasi transfer
func (h *BusinessTransferHandler) InitiateTransfer(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "business_id wajib diisi", nil)
	}

	var input models.InitiateTransferInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	res, errSvc := h.service.InitiateTransfer(accountID, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Kode OTP otorisasi transfer telah dikirim", res)
}

// VerifyOTP memvalidasi kode OTP pemilik lama dan menerbitkan permohonan transfer ke calon owner baru
func (h *BusinessTransferHandler) VerifyOTP(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "business_id wajib diisi", nil)
	}

	var input models.VerifyTransferOTPInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	transferReq, errSvc := h.service.VerifyOTPAndCreateTransfer(accountID, businessID, input)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Permohonan transfer berhasil dibuat dan undangan telah dikirimkan ke calon pemilik baru", transferReq)
}

// CancelTransfer membatalkan permohonan transfer aktif oleh pemilik lama
func (h *BusinessTransferHandler) CancelTransfer(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "business_id wajib diisi", nil)
	}

	if errSvc := h.service.CancelTransfer(accountID, businessID); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Permohonan transfer berhasil dibatalkan", nil)
}

// GetIncomingTransfers menampilkan daftar undangan transfer bisnis yang masuk ke akun yang sedang login
func (h *BusinessTransferHandler) GetIncomingTransfers(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	list, errSvc := h.service.GetIncomingTransfers(accountID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, errSvc.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar undangan transfer bisnis berhasil dimuat", list)
}

// RespondTransfer merespon undangan transfer bisnis (accept atau reject) oleh calon pemilik baru
func (h *BusinessTransferHandler) RespondTransfer(c *fiber.Ctx) error {
	accountID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
	}

	transferID := strings.TrimSpace(c.Params("transfer_id"))
	if transferID == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "transfer_id wajib diisi", nil)
	}

	var input models.RespondTransferInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	if errSvc := h.service.RespondTransfer(accountID, transferID, input); errSvc != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	actionMsg := "Undangan transfer bisnis berhasil disetujui. Bisnis sekarang menjadi milik Anda."
	if strings.ToLower(input.Action) == "reject" {
		actionMsg = "Undangan transfer bisnis berhasil ditolak."
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, actionMsg, nil)
}
