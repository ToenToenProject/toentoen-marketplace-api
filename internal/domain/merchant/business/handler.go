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
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Request format is not valid", fiber.Map{"error": err.Error()})
	}

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid or you are not logged in yet", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to process user ID", nil)
	}

	err := h.service.CreateBusiness(ownerID, input)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to create business profile", fiber.Map{"error": err.Error()})
	}

	return utils.SendResponse(c, fiber.StatusCreated, constants.Success, "Business profile created and saved successfully", nil)
}

func (h *BusinessHandler) GetMyBusinesses(c *fiber.Ctx) error {
	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid or you are not logged in yet", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to process user ID", nil)
	}

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
	businessesResponse, err := h.service.GetMyBusinesses(ownerID, search, categoryID, tagIDs, cursor, limit)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to fetch business data", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Successfully fetched list of businesses", businessesResponse)
}

func (h *BusinessHandler) UpdateBusiness(c *fiber.Ctx) error {
	businessID := c.Params("id")

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid or you are not logged in yet", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to process user ID", nil)
	}

	var input models.UpdateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Request format is not valid", nil)
	}

	err := h.service.UpdateBusiness(ownerID, businessID, input)
	if err != nil {
		if err.Error() == "handle already exists" {
			return utils.SendResponse(c, fiber.StatusConflict, constants.HandleTaken, err.Error(), nil)
		}
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.ForbiddenAccess, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Business updated successfully", nil)
}

func (h *BusinessHandler) DeleteBusiness(c *fiber.Ctx) error {
	businessID := c.Params("id")

	accountIDValue := c.Locals("account_id")
	if accountIDValue == nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Session is not valid or you are not logged in yet", nil)
	}

	var ownerID, ok = accountIDValue.(string)
	if !ok {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Failed to process user ID", nil)
	}

	if err := h.service.DeleteBusiness(ownerID, businessID); err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Business deleted successfully", nil)
}

func (h *BusinessHandler) GetBusinessDetail(c *fiber.Ctx) error {
	businessID := c.Params("business_id")

	business, err := h.service.GetBusinessDetail(businessID)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, "SUCCESS", "Business details retrieved successfully", business)
}

func (h *BusinessHandler) ToggleStoreStatus(c *fiber.Ctx) error {
	businessID := strings.TrimSpace(c.Params("business_id"))

	// Ambil ID owner dari token JWT 
	// (Pastikan fungsi getRequesterInfo sudah ada di file handler lain dalam package yang sama, misal di product_handler.go)
	requesterID, _, err := utils.GetRequesterInfo(c)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Unauthorized", nil)
	}

	isClosed, errSvc := h.service.ToggleStoreStatus(requesterID, businessID)
	if errSvc != nil {
		return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, errSvc.Error(), nil)
	}

	// Jika isClosed == true, berarti toko ditutup manual
	msg := "Store is now operating normally (Open)"
	if isClosed {
		msg = "Store is manually Closed"
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, msg, fiber.Map{
		"is_manually_closed": isClosed,
	})
}