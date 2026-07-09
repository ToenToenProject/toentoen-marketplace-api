package admin

import (
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

	// List untuk Owner & Admin
	GetAssignmentsByBusinessID(businessID string) ([]models.BusinessAdminAssignment, error)
	GetAssignmentsByAdminID(adminID string) ([]models.BusinessAdminAssignment, error)
	FindAccountByEmail(email string) (*models.Account, error)
}

type businessAdminRepository struct {
	db *gorm.DB
}

func NewBusinessAdminRepository(db *gorm.DB) BusinessAdminRepository {
	return &businessAdminRepository{db}
}

// Memastikan profil BusinessAdmin ada sebelum assignment dibuat
func (r *businessAdminRepository) CreateAdminProfileIfNotExist(accountID string) error {
	admin := models.BusinessAdmin{AccountID: accountID}
	// FirstOrCreate akan mengecek apakah AccountID sudah ada. Jika belum, akan dibuat.
	return r.db.Where("account_id = ?", accountID).FirstOrCreate(&admin).Error
}

// Membuat Undangan (Assignment)
func (r *businessAdminRepository) CreateAssignment(assignment *models.BusinessAdminAssignment) error {
	return r.db.Create(assignment).Error
}

// Mengambil 1 Assignment berdasarkan ID (Biasa dipakai untuk Update/Terima Undangan)
func (r *businessAdminRepository) GetAssignmentByID(assignmentID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Preload("Business").
		Preload("Admin.Account"). // Ambil juga data akun si Admin
		Where("assignment_id = ?", assignmentID).
		First(&assignment).Error
	return &assignment, err
}

// Mengecek apakah seseorang sudah pernah diundang ke toko tertentu (Mencegah duplikat undangan)
func (r *businessAdminRepository) GetAssignmentByAdminAndBusiness(adminID, businessID string) (*models.BusinessAdminAssignment, error) {
	var assignment models.BusinessAdminAssignment
	err := r.db.
		Where("admin_id = ? AND business_id = ?", adminID, businessID).
		First(&assignment).Error
	return &assignment, err
}

// Menyimpan perubahan (Terima/Tolak, Ubah Shift, Pemecatan)
func (r *businessAdminRepository) UpdateAssignment(assignment *models.BusinessAdminAssignment) error {
	return r.db.Save(assignment).Error
}

// UNTUK OWNER: Melihat siapa saja yang menjadi admin (atau masih pending) di tokonya
func (r *businessAdminRepository) GetAssignmentsByBusinessID(businessID string) ([]models.BusinessAdminAssignment, error) {
	var assignments []models.BusinessAdminAssignment
	err := r.db.
		Preload("Admin.Account"). // Sangat penting agar Owner bisa melihat nama & email admin-nya
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&assignments).Error
	return assignments, err
}

// UNTUK CALON ADMIN: Melihat daftar undangan toko atau toko tempat ia bekerja sekarang
func (r *businessAdminRepository) GetAssignmentsByAdminID(adminID string) ([]models.BusinessAdminAssignment, error) {
	var assignments []models.BusinessAdminAssignment
	err := r.db.
		Preload("Business"). // Penting agar si Admin tahu dia diundang oleh toko apa
		Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Find(&assignments).Error
	return assignments, err
}

// FUNGSI BARU: Mencari AccountID berdasarkan Email
func (r *businessAdminRepository) FindAccountByEmail(email string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("email = ?", email).First(&account).Error
	return &account, err
}