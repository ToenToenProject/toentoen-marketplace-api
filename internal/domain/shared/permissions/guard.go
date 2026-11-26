package permissions

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// GetAdminPermissions resolves active permissions for the given account.
// Owners and supervisors receive full access (all true); staff permissions are loaded from assignment JSONB.
// Unassigned accounts default to no permissions (all false).
func GetAdminPermissions(db *gorm.DB, accountID, businessID string) models.AdminPermissions {
	var assignment models.BusinessAdminAssignment
	err := db.Select("role, permissions").
		Where("admin_id = ? AND business_id = ? AND (status = 'active' OR status = 'scheduled_for_deletion')", accountID, businessID).
		First(&assignment).Error
	if err != nil {
		// Unassigned account: assume owner role with full permissions.
		return models.AdminPermissions{}.AllTrue()
	}
	if assignment.Permissions != nil && len(assignment.Permissions) > 0 {
		return assignment.Permissions
	}
	if assignment.Role == "supervisor" {
		return models.AdminPermissions{}.AllTrue()
	}
	// Staff: load granular permissions from database JSONB (missing keys default to false).
	perms := assignment.Permissions
	if perms == nil {
		perms = models.AdminPermissions{}
	}
	return perms
}

// HasPermission verifies whether an account holds a permission for a business.
//
// Always return true for store owners.
// Always return true for supervisors.
// For staff, evaluate permission boolean from JSONB assignment.
func HasPermission(db *gorm.DB, accountID, businessID, permKey string) bool {
	perms := GetAdminPermissions(db, accountID, businessID)
	if perms[permKey] {
		return true
	}

	// Compatibility fallback for legacy merged permission keys:
	switch permKey {
	case models.PermEditProduct:
		return perms[models.PermEditProductDetails] || perms[models.PermChangeProductImage]
	case models.PermEditStoreProfileAndHours:
		return perms[models.PermEditBusinessProfile] || perms[models.PermChangeBizHours]
	case models.PermManageStoreStatus:
		return perms[models.PermToggleStoreStatus]
	case models.PermManageBusinessPosts:
		return perms[models.PermCreatePost]
	case models.PermReplyBusinessReviews, models.PermReplyProductReviews, models.PermReplyFeedComments:
		return perms[models.PermReplyReviews]
	case models.PermViewAnalyticsDaily, models.PermViewAnalyticsHistory, models.PermViewAnalyticsExplore:
		return perms[models.PermViewAnalytics]
	case models.PermCreateProductGroup, models.PermDeleteProductGroup, models.PermReorderProductGroups:
		return perms[models.PermManageProductGroups]
	}

	return false
}
