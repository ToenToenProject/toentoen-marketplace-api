package business

import (
	"strconv"
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type BusinessHandler struct {
	service BusinessService
}

func NewBusinessHandler(service BusinessService) *BusinessHandler {
	return &BusinessHandler{service}
}

func (h *BusinessHandler) CreateBusiness(c *fiber.Ctx) error {
	var input models.CreateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	businessID, err := h.service.CreateBusiness(ownerID, input)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "belum terverifikasi") || strings.Contains(errMsg, "KYC") || strings.Contains(errMsg, "identitas") || strings.Contains(errMsg, "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errMsg, fiber.Map{"error": errMsg})
		}
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal membuat profil toko", fiber.Map{"error": err.Error()})
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Profil toko berhasil dibuat", fiber.Map{
		"business_id": businessID,
	})
}

func (h *BusinessHandler) GetMyBusinesses(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var accountID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	roleValue := c.Locals("role")
	userRole, _ := roleValue.(string)

	// 1. Tangkap Parameter Pencarian dan Kursor
	search := strings.TrimSpace(c.Query("search", ""))
	cursor := strings.TrimSpace(c.Query("cursor", ""))
	
	limit := c.QueryInt("limit", 10)
	if limit > 50 {
		limit = 50
	}

	// 2. Tangkap Parameter Category ID
	categoryID := int64(0)
	catQuery := strings.TrimSpace(c.Query("category_id", ""))
	if catQuery != "" {
		if id, err := strconv.ParseInt(catQuery, 10, 64); err == nil {
			categoryID = id
		}
	}

	// 3. Tangkap Parameter Tags (Contoh dari URL: ?tags=1,5,12)
	var tagIDs []int64
	tagsQuery := strings.TrimSpace(c.Query("tags", ""))
	if tagsQuery != "" {
		tagStringArray := strings.Split(tagsQuery, ",") // Pecah berdasarkan koma
		for _, tagStr := range tagStringArray {
			if id, err := strconv.ParseInt(strings.TrimSpace(tagStr), 10, 64); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// 4. Lempar Semuanya ke Service
	businessesResponse, err := h.service.GetMyBusinesses(accountID, userRole, search, categoryID, tagIDs, cursor, limit)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal mengambil data toko", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Berhasil mengambil daftar toko", businessesResponse)
}

func (h *BusinessHandler) UpdateBusiness(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}

	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.UpdateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	if errors := utils.ValidateStruct(&input); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	err = h.service.UpdateBusiness(requesterID, businessID, input)
	if err != nil {
		if strings.Contains(err.Error(), "handle sudah digunakan") || strings.Contains(err.Error(), "handle already exists") {
			return utils.SendResponse(c, fiber.StatusConflict, constants.HandleTaken, "Handle sudah digunakan, silakan pilih yang lain", nil)
		}
		if strings.Contains(err.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
		}
		if strings.Contains(err.Error(), "tidak memiliki akses") || strings.Contains(err.Error(), "tidak memiliki izin") || strings.Contains(err.Error(), "forbidden") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Data toko berhasil diperbarui", nil)
}

func (h *BusinessHandler) DeleteBusiness(c *fiber.Ctx) error {
	businessID := c.Params("id")

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal memproses ID pengguna", nil)
	}

	if err := h.service.DeleteBusiness(ownerID, businessID); err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Toko tidak ditemukan atau Anda tidak memiliki akses", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Toko berhasil dihapus", nil)
}

func (h *BusinessHandler) GetBusinessDetail(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}

	business, err := h.service.GetBusinessDetail(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Toko tidak ditemukan", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Detail toko berhasil diambil", business)
}

func (h *BusinessHandler) ToggleStoreStatus(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))

	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	isClosed, errSvc := h.service.ToggleStoreStatus(requesterID, businessID)
	if errSvc != nil {
		if strings.Contains(errSvc.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
	}

	msg := "Toko sekarang berstatus Buka"
	if isClosed {
		msg = "Toko sekarang berstatus Tutup (manual)"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, fiber.Map{
		"is_manually_closed": isClosed,
	})
}

func (h *BusinessHandler) ToggleLiveStatus(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Params("id"))
	}

	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid atau Anda belum login", nil)
	}

	var input models.ToggleLiveInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	res, errSvc := h.service.ToggleLiveStatus(requesterID, businessID, input)
	if errSvc != nil {
		if strings.Contains(errSvc.Error(), "tidak ditemukan") {
			return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, errSvc.Error(), nil)
		}
		if strings.Contains(errSvc.Error(), "forbidden") || strings.Contains(errSvc.Error(), "tidak memiliki akses") {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, errSvc.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, errSvc.Error(), nil)
	}

	msg := "Siaran live streaming berhasil diaktifkan"
	if !res.IsLive {
		msg = "Siaran live streaming berhasil dinonaktifkan"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, res)
}