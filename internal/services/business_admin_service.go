package services

import (
	"errors"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
	"toentoen-backend/pkg/utils"
)

type BusinessAdminService interface {
	InviteAdmin(ownerID string, input models.InviteAdminInput) error
	RespondToInvitation(adminID string, assignmentID string, input models.RespondInvitationInput) error
	UpdateAdminAssignment(ownerID string, assignmentID string, input models.UpdateAssignmentInput) error
	GetAdminsByBusiness(ownerID, businessID string) ([]models.BusinessAdminAssignment, error)
	GetMyAssignments(adminID string) ([]models.BusinessAdminAssignment, error)
}

type businessAdminService struct {
	adminRepo repositories.BusinessAdminRepository
	bizRepo   repositories.BusinessRepository // Butuh ini untuk memvalidasi kepemilikan toko
}

// Inject kedua repository saat inisialisasi
func NewBusinessAdminService(adminRepo repositories.BusinessAdminRepository, bizRepo repositories.BusinessRepository) BusinessAdminService {
	return &businessAdminService{adminRepo, bizRepo}
}

// 1. OWNER MENGUNDANG ADMIN
func (s *businessAdminService) InviteAdmin(ownerID string, input models.InviteAdminInput) error {
	// Validasi: Pastikan pengundang benar-benar Owner dari bisnis tersebut
	_, err := s.bizRepo.FindBusinessByIDAndOwner(input.BusinessID, ownerID)
	if err != nil {
		return errors.New("forbidden: you do not own this business")
	}

	// 🪄 CARI AKUN PEKERJA BERDASARKAN EMAIL
	targetAccount, err := s.adminRepo.FindAccountByEmail(input.Email)
	if err != nil {
		return errors.New("user with this email not found in ToenToen system")
	}

	targetAccountID := targetAccount.AccountID // Ini ID yang akan kita pakai

	// Cegah owner mengundang dirinya sendiri
	if ownerID == targetAccountID {
		return errors.New("you cannot invite yourself as an admin")
	}

	// Cek apakah undangan sudah pernah dikirim sebelumnya
	existing, err := s.adminRepo.GetAssignmentByAdminAndBusiness(targetAccountID, input.BusinessID)
	if err == nil && existing != nil {
		if existing.Status == "pending" {
			return errors.New("an invitation is already pending for this user")
		} else if existing.Status == "active" {
			return errors.New("this user is already an active admin for this business")
		}
	}

	// Pastikan profil admin ada (Silent create jika belum ada)
	if err := s.adminRepo.CreateAdminProfileIfNotExist(targetAccountID); err != nil {
		return errors.New("failed to initialize admin profile")
	}

	// Buat Undangan
	assignment := models.BusinessAdminAssignment{
		AssignmentID: utils.GenerateUUID("ASN"),
		AdminID:      targetAccountID, // 🪄 Gunakan ID yang didapat dari database
		BusinessID:   input.BusinessID,
		InvitedBy:    ownerID,
		Role:         "business_admin", 
		Status:       "pending",        
		ShiftStart:   input.ShiftStart,
		ShiftEnd:     input.ShiftEnd,
	}

	if input.Role != "" {
		assignment.Role = input.Role
	}

	if err := s.adminRepo.CreateAssignment(&assignment); err != nil {
		return errors.New("failed to send invitation")
	}

	return nil
}

// 2. CALON ADMIN MERESPONS UNDANGAN
func (s *businessAdminService) RespondToInvitation(adminID string, assignmentID string, input models.RespondInvitationInput) error {
	// Cek apakah status respons valid
	if input.Status != "active" && input.Status != "rejected" {
		return errors.New("invalid status response. Use 'active' or 'rejected'")
	}

	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("invitation not found")
	}

	// 🚨 VALIDASI KRUSIAL: Pastikan yang memencet tombol Terima/Tolak adalah akun yang benar-benar diundang!
	if assignment.AdminID != adminID {
		return errors.New("forbidden: this invitation does not belong to you")
	}

	if assignment.Status != "pending" {
		return errors.New("this invitation has already been processed")
	}

	assignment.Status = input.Status
	if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
		return errors.New("failed to update invitation status")
	}

	return nil
}

// 3. OWNER MENGUBAH / MEMECAT ADMIN
func (s *businessAdminService) UpdateAdminAssignment(ownerID string, assignmentID string, input models.UpdateAssignmentInput) error {
	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("assignment not found")
	}

	// Validasi: Pastikan yang mengubah adalah Owner asli dari toko tersebut
	_, err = s.bizRepo.FindBusinessByIDAndOwner(assignment.BusinessID, ownerID)
	if err != nil {
		return errors.New("forbidden: you do not own this business")
	}

	// Update field (HANYA operasional toko, data pribadi admin tidak disentuh)
	if input.Role != nil {
		assignment.Role = *input.Role
	}
	if input.ShiftStart != nil {
		assignment.ShiftStart = input.ShiftStart
	}
	if input.ShiftEnd != nil {
		assignment.ShiftEnd = input.ShiftEnd
	}
	
	// Jika Owner ingin memecat/memberhentikan, dia mengirim status 'revoked'
	if input.Status != nil {
		if *input.Status == "revoked" || *input.Status == "active" {
			assignment.Status = *input.Status
		} else {
			return errors.New("invalid status update. Use 'revoked' or 'active'")
		}
	}

	if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
		return errors.New("failed to update admin assignment")
	}

	return nil
}

// 4. OWNER MELIHAT DAFTAR ADMIN DI TOKONYA
func (s *businessAdminService) GetAdminsByBusiness(ownerID, businessID string) ([]models.BusinessAdminAssignment, error) {
	// Validasi kepemilikan toko
	_, err := s.bizRepo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return nil, errors.New("forbidden: you do not own this business")
	}

	return s.adminRepo.GetAssignmentsByBusinessID(businessID)
}

// 5. ADMIN MELIHAT PEKERJAAN / UNDANGANNYA SENDIRI
func (s *businessAdminService) GetMyAssignments(adminID string) ([]models.BusinessAdminAssignment, error) {
	// Tidak perlu validasi owner, karena admin berhak melihat daftar tempat dia bekerja/diundang
	return s.adminRepo.GetAssignmentsByAdminID(adminID)
}