package middlewares

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// BusinessAccessGuard memastikan Business Admin yang sedang dalam status scheduled_for_deletion
// hanya memiliki akses Read-Only (GET/HEAD/OPTIONS), dan memblokir akses jika bukan admin aktif
func BusinessAccessGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleValue := c.Locals("role")
		accountIDValue := c.Locals("account_id")

		if roleValue == nil || accountIDValue == nil {
			return utils.SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, "Sesi tidak valid", nil)
		}

		userRole, _ := roleValue.(string)
		accountID, _ := accountIDValue.(string)

		// Jika Owner, izinkan lanjut (kepemilikan akan divalidasi oleh handler/service)
		if userRole == "owner" {
			return c.Next()
		}

		if userRole != "business_admin" {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, "Role Anda tidak memiliki akses ke fitur toko ini", nil)
		}

		businessID := strings.TrimSpace(c.Params("business_id"))
		if businessID == "" {
			businessID = strings.TrimSpace(c.Params("id"))
		}

		// Jika tidak ada parameter business_id di rute, lanjutkan
		if businessID == "" {
			return c.Next()
		}

		var assignment models.BusinessAdminAssignment
		err := database.DB.Where("admin_id = ? AND business_id = ?", accountID, businessID).First(&assignment).Error
		if err != nil {
			return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, "Anda tidak terdaftar sebagai admin di toko ini", nil)
		}

		switch assignment.Status {
		case "active":
			c.Locals("admin_role", assignment.Role)
			c.Locals("assignment_id", assignment.AssignmentID)
			return c.Next()

		case "scheduled_for_deletion":
			// 🔒 READ-ONLY ENFORCEMENT
			method := c.Method()
			if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
				c.Locals("admin_role", assignment.Role)
				c.Locals("assignment_id", assignment.AssignmentID)
				c.Locals("is_read_only", true)
				return c.Next()
			}
			return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, "Akun Anda dalam masa tenggang 3 hari (Read-Only Mode). Anda tidak dapat mengubah data toko.", nil)

		case "pending", "pending_owner_approval":
			return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, "Undangan Anda di toko ini belum aktif", nil)

		case "revoked", "rejected":
			return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, "Akses Anda ke toko ini telah dinonaktifkan", nil)

		default:
			return utils.SendResponse(c, fiber.StatusForbidden, constants.FailedValidation, "Akses ke toko ini tidak diizinkan", nil)
		}
	}
}
