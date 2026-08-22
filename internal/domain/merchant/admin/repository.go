package admin

import (
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type BusinessAdminRepository interface {
	// Profil Admin
	CreateAdminProfileIfNotExist(accountID string) error

	// Undangan & Penugasan
	CreateAssignment(assignment *models.BusinessAdminAssignment) error
	GetAssignmentByID(assignmentID string) (*models.BusinessAdminAssignment, error)
	GetAssignmentByAdminAndBusiness(adminID, businessID string) (*models.BusinessAdminAssignment, error)
	UpdateAssignment(assignment *models.BusinessAdminAssignment) error
	ReplaceShifts(assignmentID string, shifts []models.BusinessAdminShift) error

	// List untuk Owner & Admin
	GetAssignmentsByBusinessID(businessID string) ([]models.BusinessAdminAssignment, error)
	GetAssignmentsByAdminID(adminID string) ([]models.BusinessAdminAssignment, error)
	GetActiveAssignment(adminID, businessID string) (*models.BusinessAdminAssignment, error)

	// Pencarian Akun
	FindAccountByInvitationCode(code string) (*models.Account, error)
	FindAccountByID(accountID string) (*models.Account, error)

	// Hierarki & Admin Tree
	GetSubordinates(businessID, supervisorID string) ([]models.BusinessAdminAssignment, error)
	ReassignSubordinatesToOwner(businessID, oldSupervisorID string) error

	// Grace Period & Worker
	GetExpiredScheduledDeletions() ([]models.BusinessAdminAssignment, error)
	BatchRevokeAssignments(assignmentIDs []string) error

	// Permissions
	GetPermissions(assignmentID string) (*models.BusinessAdminAssignment, error)
	UpdatePermissions(assignmentID string, permissions models.AdminPermissions) error
}

type businessAdminRepository struct {
	db *gorm.DB
}

func NewBusinessAdminRepository(db *gorm.DB) BusinessAdminRepository {
	return &businessAdminRepository{db}
}

func (r *businessAdminRepository) CreateAdminProfileIfNotExist(accountID string) error {
	admin := models.BusinessAdmin{AccountID: accountID}
	return r.db.Where("account_id = ?", accountID).FirstOrCreate(&admin).Error
}

func (r *businessAdminRepository) CreateAssignment(assignment *models.BusinessAdminAssignment) error {
	return r.db.Create(assignment).Error
}

func (r *businessAdminRepository) GetAssignmentByID(assignmentID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Business").
		Preload("Admin.Account").
		Preload("Supervisor").
		Preload("Inviter").
		Preload("Shifts").
		Where("assignment_id = ?", assignmentID).
		First(&assignment).Error
	return &assignment, err
}

func (r *businessAdminRepository) GetAssignmentByAdminAndBusiness(adminID, businessID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Business").
		Preload("Admin.Account").
		Preload("Supervisor").
		Preload("Inviter").
		Preload("Shifts").
		Where("admin_id = ? AND business_id = ?", adminID, businessID).
		First(&assignment).Error
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (r *businessAdminRepository) GetActiveAssignment(adminID, businessID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Shifts").
		Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", adminID, businessID).
		First(&assignment).Error
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (r *businessAdminRepository) UpdateAssignment(assignment *models.BusinessAdminAssignment) error {
	return r.db.Save(assignment).Error
}

func (r *businessAdminRepository) ReplaceShifts(assignmentID string, shifts []models.BusinessAdminShift) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Hapus shift lama
		if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.BusinessAdminShift{}).Error; err != nil {
			return err
		}
		// Insert shift baru
		if len(shifts) > 0 {
			if err := tx.Create(&shifts).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *businessAdminRepository) GetAssignmentsByBusinessID(businessID string) ([]models.BusinessAdminAssignment, error) {
	var assignments []models.BusinessAdminAssignment
	err := r.db.
		Preload("Admin.Account").
		Preload("Supervisor").
		Preload("Inviter").
		Preload("Shifts").
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&assignments).Error
	return assignments, err
}

func (r *businessAdminRepository) GetAssignmentsByAdminID(adminID string) ([]models.BusinessAdminAssignment, error) {
	var assignments []models.BusinessAdminAssignment
	err := r.db.
		Preload("Business").
		Preload("Supervisor").
		Preload("Inviter").
		Preload("Shifts").
		Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Find(&assignments).Error
	return assignments, err
}

func (r *businessAdminRepository) FindAccountByInvitationCode(code string) (*models.Account, error) {
	var admin models.BusinessAdmin
	err := r.db.Preload("Account").Where("invitation_code = ?", code).First(&admin).Error
	if err != nil {
		return nil, err
	}
	if admin.Account == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return admin.Account, nil
}

func (r *businessAdminRepository) FindAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *businessAdminRepository) GetSubordinates(businessID, supervisorID string) ([]models.BusinessAdminAssignment, error) {
	var assignments []models.BusinessAdminAssignment
	err := r.db.
		Preload("Admin.Account").
		Preload("Shifts").
		Where("business_id = ? AND supervisor_id = ? AND status = 'active'", businessID, supervisorID).
		Find(&assignments).Error
	return assignments, err
}

func (r *businessAdminRepository) ReassignSubordinatesToOwner(businessID, oldSupervisorID string) error {
	// Pindahkan bawahan supervisor ini ke Owner (supervisor_id = NULL)
	return r.db.Model(&models.BusinessAdminAssignment{}).
		Where("business_id = ? AND supervisor_id = ?", businessID, oldSupervisorID).
		Update("supervisor_id", nil).Error
}

func (r *businessAdminRepository) GetExpiredScheduledDeletions() ([]models.BusinessAdminAssignment, error) {
	var assignments []models.BusinessAdminAssignment
	err := r.db.
		Where("status = 'scheduled_for_deletion' AND deletion_scheduled_at <= ?", time.Now()).
		Find(&assignments).Error
	return assignments, err
}

func (r *businessAdminRepository) BatchRevokeAssignments(assignmentIDs []string) error {
	if len(assignmentIDs) == 0 {
		return nil
	}
	return r.db.Model(&models.BusinessAdminAssignment{}).
		Where("assignment_id IN ?", assignmentIDs).
		Update("status", "revoked").Error
}

// GetPermissions mengambil assignment beserta nama admin untuk ditampilkan di response.
func (r *businessAdminRepository) GetPermissions(assignmentID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Admin.Account").
		Where("assignment_id = ?", assignmentID).
		First(&assignment).Error
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// UpdatePermissions mengupdate kolom permissions JSONB pada assignment tertentu.
func (r *businessAdminRepository) UpdatePermissions(assignmentID string, permissions models.AdminPermissions) error {
	return r.db.Model(&models.BusinessAdminAssignment{}).
		Where("assignment_id = ?", assignmentID).
		Update("permissions", permissions).Error
}