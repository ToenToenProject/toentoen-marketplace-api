package repositories

import (
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
)

type AssignmentRepository interface {
	CheckBusinessOwnership(ownerID, businessID string) bool
	CheckBusinessAdminOwnership(ownerID, businessAdminID string) bool
	AssignBusinessAdmin(assignment *models.BusinessAdminAssignment) error
	GetAllAdminsByOwner(ownerId string) ([]models.BusinessAdmin, error)
	GetAssignment(adminID, businessID string) (*models.BusinessAdminAssignment, error)
	DeleteBusinessAdminAccount(adminID string) error
	GetAssignmentWithDetails(adminID, businessID string) (*models.BusinessAdminAssignment, error)
	
	UpdateAdminAccount(adminID string, data map[string]interface{}) error
}

type assignmentRepository struct {
	db *gorm.DB
}

func NewAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &assignmentRepository{db}
}

func (r *assignmentRepository) CheckBusinessOwnership(ownerID, businessID string) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, ownerID).Count(&count)
	return count > 0
}

func (r *assignmentRepository) CheckBusinessAdminOwnership(ownerID, businessAdminID string) bool {
	var count int64
	r.db.Model(&models.BusinessAdmin{}).Where("account_id = ? AND owner_id = ?", businessAdminID, ownerID).Count(&count)
	return count > 0
}

func (r *assignmentRepository) AssignBusinessAdmin(assignment *models.BusinessAdminAssignment) error {
	return r.db.Save(assignment).Error
}

func (r *assignmentRepository) GetAllAdminsByOwner(ownerID string) ([]models.BusinessAdmin, error) {
	var admins []models.BusinessAdmin
	err := r.db.
		Preload("Account").
		Preload("Assignments").
		Preload("Assignments.Business").
		Where("owner_id = ?", ownerID).
		Find(&admins).Error
	return admins, err
}

func (r *assignmentRepository) GetAssignment(adminID, businessID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.Where("admin_id = ? AND business_id = ?", adminID, businessID).First(&assignment).Error
	return &assignment, err
}

func (r *assignmentRepository) DeleteBusinessAdminAccount(adminID string) error {
	return r.db.Where("account_id = ?", adminID).Delete(&models.Account{}).Error
}

func (r *assignmentRepository) GetAssignmentWithDetails(adminID, businessID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Business").
		Preload("Admin.Account").
		Where("admin_id = ? AND business_id = ?", adminID, businessID).
		First(&assignment).Error
		
	return &assignment, err
}

func (r *assignmentRepository) UpdateAdminAccount(adminID string, data map[string]interface{}) error {
	return r.db.Model(&models.Account{}).Where("account_id = ?", adminID).Updates(data).Error
}