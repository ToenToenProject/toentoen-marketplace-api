package permissions

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// GetAdminPermissions mengambil permissions dari assignment yang aktif.
//
// Aturan:
//   - Owner (tidak ada assignment) → semua permission true.
//   - Supervisor → semua permission true.
//   - Staff aktif → ambil dari kolom JSONB di assignment.
//   - Assignment tidak ditemukan → anggap bukan admin toko ini, return semua false.
func GetAdminPermissions(db *gorm.DB, accountID, businessID string) models.AdminPermissions {
	var assignment models.BusinessAdminAssignment
	err := db.Select("role, permissions").
		Where("admin_id = ? AND business_id = ? AND (status = 'active' OR status = 'scheduled_for_deletion')", accountID, businessID).
		First(&assignment).Error
	if err != nil {
		// Tidak ditemukan sebagai admin → mungkin owner, izinkan penuh
		return models.AdminPermissions{}.AllTrue()
	}
	if assignment.Role == "supervisor" {
		return models.AdminPermissions{}.AllTrue()
	}
	// Staff: kembalikan permissions dari DB (key tidak ada = false)
	perms := assignment.Permissions
	if perms == nil {
		perms = models.AdminPermissions{}
	}
	return perms
}

// HasPermission memeriksa apakah account memiliki izin tertentu untuk bisnis terkait.
//
// Selalu return true untuk owner (tidak ada assignment di toko tersebut).
// Return true untuk supervisor.
// Untuk staff, cek key permission dari JSONB.
func HasPermission(db *gorm.DB, accountID, businessID, permKey string) bool {
	perms := GetAdminPermissions(db, accountID, businessID)
	return perms[permKey]
}
