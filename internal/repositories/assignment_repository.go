package repositories

import (
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
)

type AssignmentRepository interface {
	CheckBusinessOwnership(ownerID, businessID int64) bool
	CheckBusinessAdminOwnership(ownerID, businessAdminID int64) bool
	AssignBusinessAdmin(assignment *models.BusinessAdminAssignment) error
	GetAllAdminsByOwner(ownerId int64) ([]models.BusinessAdmin, error)
	GetAssignment(adminID, businessID int64) (*models.BusinessAdminAssignment, error)
	DeleteBusinessAdminAccount(adminID int64) error
	GetAssignmentWithDetails(adminID, businessID int64) (*models.BusinessAdminAssignment, error)
}

type assignmentRepository struct {
	db *gorm.DB
}

func NewAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &assignmentRepository{db}
}

func (r *assignmentRepository) CheckBusinessOwnership(ownerID, businessID int64) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, ownerID).Count(&count)
	return count > 0
}

func (r *assignmentRepository) CheckBusinessAdminOwnership(ownerID, businessAdminID int64) bool {
	var count int64
	r.db.Model(&models.BusinessAdmin{}).Where("account_id = ? AND owner_id = ?", businessAdminID, ownerID).Count(&count)
	return count > 0
}

func (r *assignmentRepository) AssignBusinessAdmin(assignment *models.BusinessAdminAssignment) error {
	return r.db.Save(assignment).Error
}

func (r *assignmentRepository) GetAllAdminsByOwner(ownerID int64) ([]models.BusinessAdmin, error) {
	var admins []models.BusinessAdmin
	err := r.db.
		Preload("Account").
		Preload("Assignments").
		Preload("Assignments.Business").
		Where("owner_id = ?", ownerID).
		Find(&admins).Error
	return admins, err
}

func (r *assignmentRepository) GetAssignment(adminID, businessID int64) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.Where("admin_id = ? AND business_id = ?", adminID, businessID).First(&assignment).Error
	return &assignment, err
}

func (r *assignmentRepository) DeleteBusinessAdminAccount(adminID int64) error {
	return r.db.Where("account_id = ?", adminID).Delete(&models.Account{}).Error
}

func (r *assignmentRepository) GetAssignmentWithDetails(adminID, businessID int64) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Business").
		Preload("Admin.Account").
		Where("admin_id = ? AND business_id = ?", adminID, businessID).
		First(&assignment).Error
		
	return &assignment, err
}