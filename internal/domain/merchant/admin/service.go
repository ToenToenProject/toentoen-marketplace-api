package admin

import (
	"errors"
	"fmt"
	"time"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type BusinessAdminService interface {
	// Undangan & Persetujuan
	InviteAdmin(requesterID, businessID string, input models.InviteAdminInput) error
	RespondOwnerApproval(ownerID, businessID, assignmentID string, input models.RespondOwnerApprovalInput) error
	RespondToInvitation(adminID, assignmentID string, input models.RespondInvitationInput) error

	// Update Penugasan & Demosi
	UpdateAdminAssignment(ownerID, businessID, assignmentID string, input models.UpdateAssignmentInput) error

	// Grace Period 3 Hari (Pemecatan & Resign)
	TerminateAdmin(requesterID, businessID, assignmentID string) error
	ResignFromBusiness(adminID, assignmentID string) error
	CancelTermination(requesterID, assignmentID string) error

	GetAdminsByOwner(ownerID string) ([]models.AdminTreeResponse, error)
	GetMyAssignments(adminID string) ([]models.AdminAssignmentListItem, error)
	GetAdminTree(requesterID, businessID string) (*models.AdminTreeResponse, error)

	// Permission Management
	GetPermissions(requesterID, businessID, targetAssignmentID string) (*models.PermissionResponse, error)
	UpdatePermissions(requesterID, businessID, targetAssignmentID string, input models.UpdatePermissionsInput) (*models.PermissionResponse, error)
}

type businessAdminService struct {
	adminRepo    BusinessAdminRepository
	bizRepo      business.BusinessRepository
	notifService notification.NotificationService
}

func NewBusinessAdminService(adminRepo BusinessAdminRepository, bizRepo business.BusinessRepository, notifService notification.NotificationService) BusinessAdminService {
	return &businessAdminService{
		adminRepo:    adminRepo,
		bizRepo:      bizRepo,
		notifService: notifService,
	}
}

// 1. REKRUT / INVITE ADMIN (Owner Langsung atau Supervisor Butuh Approval Owner)
func (s *businessAdminService) InviteAdmin(requesterID, businessID string, input models.InviteAdminInput) error {
	biz, err := s.bizRepo.FindBusinessByID(businessID)
	if err != nil {
		return errors.New("toko tidak ditemukan")
	}

	isOwner := biz.AccountID == requesterID
	isSupervisor := false

	if !isOwner {
		requesterAssignment, errA := s.adminRepo.GetActiveAssignment(requesterID, businessID)
		if errA == nil && requesterAssignment != nil && requesterAssignment.Role == "supervisor" {
			isSupervisor = true
		}
	}

	if !isOwner && !isSupervisor {
		return errors.New("forbidden: Anda tidak memiliki izin untuk merekrut admin di toko ini")
	}

	// Supervisor HANYA boleh meng-invite role 'staff'
	if isSupervisor {
		if input.Role != "staff" {
			return errors.New("supervisor hanya dapat mengundang staf/admin biasa")
		}
	}



	// Cari Akun Calon Admin
	targetAccount, err := s.adminRepo.FindAccountByInvitationCode(input.InvitationCode)
	if err != nil {
		return errors.New("akun dengan Invitation Code tersebut tidak ditemukan")
	}

	if targetAccount.Role != "business_admin" {
		return errors.New("akun tersebut bukan terdaftar sebagai business admin")
	}

	if targetAccount.AccountID == requesterID {
		return errors.New("Anda tidak dapat mengundang diri Anda sendiri")
	}

	// Cek apakah target sudah memiliki assignment di toko ini
	existing, _ := s.adminRepo.GetAssignmentByAdminAndBusiness(targetAccount.AccountID, businessID)
	if existing != nil {
		switch existing.Status {
		case "active":
			return errors.New("admin ini sudah aktif bekerja di toko Anda")
		case "pending":
			return errors.New("undangan untuk admin ini sedang menunggu respons")
		case "pending_owner_approval":
			return errors.New("pengajuan undangan untuk admin ini sedang menunggu persetujuan Owner")
		case "scheduled_for_deletion":
			return errors.New("admin ini sedang dalam masa transisi penghentian akses di toko ini")
		case "rejected":
			// Cek Cooldown Penolakan (3x reject -> 7 days cooldown)
			if existing.RejectionCount >= 3 && existing.LastRejectedAt != nil {
				if time.Since(*existing.LastRejectedAt) < 7*24*time.Hour {
					cooldownEnd := existing.LastRejectedAt.Add(7 * 24 * time.Hour).Format("02 Jan 2006 15:04")
					return fmt.Errorf("admin ini telah menolak undangan 3 kali berturut-turut. Anda baru dapat mengundang kembali setelah %s", cooldownEnd)
				}
				// Reset cooldown jika sudah lewat 7 hari
				existing.RejectionCount = 0
			}
		}
	}

	// Pastikan profil BusinessAdmin ada
	if err := s.adminRepo.CreateAdminProfileIfNotExist(targetAccount.AccountID); err != nil {
		return errors.New("gagal memproses profil admin")
	}

	assignmentID := utils.GenerateUUID("ASN")
	businessName := "Toko Anda"
	if biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	if isOwner {
		// ALUR OWNER: Langsung kirim undangan ke calon admin
		if existing != nil {
			existing.Role = input.Role
			existing.SupervisorID = nil // Owner langsung yang merekrut
			existing.InvitedBy = requesterID
			existing.Status = "pending"
			if err := s.adminRepo.UpdateAssignment(existing); err != nil {
				return errors.New("gagal mengirimkan undangan")
			}
			assignmentID = existing.AssignmentID
		} else {
			newAssignment := models.BusinessAdminAssignment{
				AssignmentID: assignmentID,
				AdminID:      targetAccount.AccountID,
				BusinessID:   businessID,
				InvitedBy:    requesterID,
				SupervisorID: nil, // Owner langsung yang merekrut
				Role:         input.Role,
				Status:       "pending",
			}
			if err := s.adminRepo.CreateAssignment(&newAssignment); err != nil {
				return errors.New("gagal mengirimkan undangan")
			}
		}

		// Kirim Notifikasi & FCM ke Calon Admin
		go s.notifService.SendNotification(
			[]string{targetAccount.AccountID},
			requesterID,
			"admin_invitation",
			assignmentID,
			"Undangan Admin Toko",
			fmt.Sprintf("Toko %s mengundang Anda untuk bergabung sebagai %s.", businessName, input.Role),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   businessID,
				"business_name": businessName,
				"role":          input.Role,
			},
			nil,
		)

	} else if isSupervisor {
		// ALUR SUPERVISOR: Ajukan persetujuan ke Owner terlebih dahulu
		if existing != nil {
			existing.Role = "staff"
			existing.SupervisorID = &requesterID
			existing.InvitedBy = requesterID
			existing.Status = "pending_owner_approval"
			if err := s.adminRepo.UpdateAssignment(existing); err != nil {
				return errors.New("gagal mengajukan undangan")
			}
			assignmentID = existing.AssignmentID
		} else {
			newAssignment := models.BusinessAdminAssignment{
				AssignmentID: assignmentID,
				AdminID:      targetAccount.AccountID,
				BusinessID:   businessID,
				InvitedBy:    requesterID,
				SupervisorID: &requesterID,
				Role:         "staff",
				Status:       "pending_owner_approval",
			}
			if err := s.adminRepo.CreateAssignment(&newAssignment); err != nil {
				return errors.New("gagal mengajukan undangan")
			}
		}

		// Kirim Notifikasi & FCM ke Owner untuk Approval
		go s.notifService.SendNotification(
			[]string{biz.AccountID},
			requesterID,
			"spv_admin_invitation_approval",
			assignmentID,
			"Persetujuan Penambahan Admin",
			fmt.Sprintf("Supervisor mengajukan penambahan %s sebagai staff di toko %s.", targetAccount.FirstName, businessName),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   businessID,
				"business_name": businessName,
				"admin_name":    targetAccount.FirstName,
				"role":          "staff",
			},
			nil,
		)
	}

	return nil
}

// 2. OWNER MERESPONS APPROVAL INVITASI DARI SUPERVISOR (Approve / Reject)
func (s *businessAdminService) RespondOwnerApproval(ownerID, businessID, assignmentID string, input models.RespondOwnerApprovalInput) error {
	biz, err := s.bizRepo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("forbidden: Anda bukan pemilik toko ini")
	}

	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("data penugasan tidak ditemukan")
	}

	if assignment.BusinessID != businessID || assignment.Status != "pending_owner_approval" {
		return errors.New("penugasan ini tidak dalam status menunggu persetujuan Owner")
	}

	businessName := "Toko Anda"
	if biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	adminName := "Admin"
	if assignment.Admin != nil && assignment.Admin.Account.FirstName != "" {
		adminName = assignment.Admin.Account.FirstName
	}

	if input.Action == "approve" {
		assignment.Status = "pending"
		if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
			return errors.New("gagal menyetujui pengajuan")
		}

		// Notifikasi ke Supervisor: Pengajuan disetujui
		go s.notifService.SendNotification(
			[]string{assignment.InvitedBy},
			ownerID,
			"spv_approval_result",
			assignmentID,
			"Pengajuan Admin Disetujui",
			fmt.Sprintf("Owner menyetujui penambahan %s di toko %s. Undangan telah diteruskan.", adminName, businessName),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   businessID,
				"status":        "approved",
			},
			nil,
		)

		// Notifikasi ke Calon Admin: Undangan Resmi dari Toko
		go s.notifService.SendNotification(
			[]string{assignment.AdminID},
			ownerID,
			"admin_invitation",
			assignmentID,
			"Undangan Admin Toko",
			fmt.Sprintf("Toko %s mengundang Anda untuk bergabung sebagai %s.", businessName, assignment.Role),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   businessID,
				"business_name": businessName,
				"role":          assignment.Role,
			},
			nil,
		)

	} else if input.Action == "reject" {
		assignment.Status = "rejected"
		if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
			return errors.New("gagal menolak pengajuan")
		}

		// Notifikasi ke Supervisor: Pengajuan ditolak Owner
		go s.notifService.SendNotification(
			[]string{assignment.InvitedBy},
			ownerID,
			"spv_approval_result",
			assignmentID,
			"Pengajuan Admin Ditolak",
			fmt.Sprintf("Owner menolak pengajuan penambahan %s di toko %s.", adminName, businessName),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   businessID,
				"status":        "rejected",
			},
			nil,
		)
	}

	return nil
}

// 3. CALON ADMIN MERESPONS UNDANGAN (Accept / Reject)
func (s *businessAdminService) RespondToInvitation(adminID, assignmentID string, input models.RespondInvitationInput) error {
	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("undangan tidak ditemukan")
	}

	if assignment.AdminID != adminID {
		return errors.New("forbidden: undangan ini bukan milik Anda")
	}

	if assignment.Status != "pending" {
		return errors.New("undangan ini sudah diproses atau tidak lagi valid")
	}

	biz, _ := s.bizRepo.FindBusinessByID(assignment.BusinessID)
	businessName := "Toko"
	if biz != nil && biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	adminName := "Admin"
	if assignment.Admin != nil && assignment.Admin.Account.FirstName != "" {
		adminName = assignment.Admin.Account.FirstName
	}

	recipients := []string{}
	if biz != nil {
		recipients = append(recipients, biz.AccountID) // Owner
	}
	if assignment.SupervisorID != nil && *assignment.SupervisorID != "" && *assignment.SupervisorID != biz.AccountID {
		recipients = append(recipients, *assignment.SupervisorID) // Supervisor
	}

	if input.Action == "accept" {
		assignment.Status = "active"
		assignment.RejectionCount = 0
		assignment.LastRejectedAt = nil
		if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
			return errors.New("gagal menerima undangan")
		}

		// Notifikasi ke Owner & SPV: Admin bergabung
		go s.notifService.SendNotification(
			recipients,
			adminID,
			"admin_joined",
			assignmentID,
			"Admin Baru Bergabung",
			fmt.Sprintf("%s telah menerima undangan dan resmi bergabung di toko %s sebagai %s.", adminName, businessName, assignment.Role),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   assignment.BusinessID,
				"admin_id":      adminID,
				"role":          assignment.Role,
			},
			nil,
		)

	} else if input.Action == "reject" {
		assignment.Status = "rejected"
		assignment.RejectionCount++
		now := time.Now()
		assignment.LastRejectedAt = &now

		if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
			return errors.New("gagal menolak undangan")
		}

		// Notifikasi ke Owner & SPV: Admin menolak undangan
		go s.notifService.SendNotification(
			recipients,
			adminID,
			"admin_rejected_invitation",
			assignmentID,
			"Undangan Ditolak",
			fmt.Sprintf("%s menolak undangan untuk bergabung di toko %s.", adminName, businessName),
			map[string]interface{}{
				"assignment_id": assignmentID,
				"business_id":   assignment.BusinessID,
				"admin_id":      adminID,
			},
			nil,
		)
	}

	return nil
}

// 4. OWNER MEMPERBARUI PERAN / SHIFT / SUPERVISOR ADMIN
func (s *businessAdminService) UpdateAdminAssignment(ownerID, businessID, assignmentID string, input models.UpdateAssignmentInput) error {
	biz, err := s.bizRepo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("forbidden: Anda bukan pemilik toko ini")
	}

	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("penugasan admin tidak ditemukan")
	}

	if assignment.BusinessID != businessID {
		return errors.New("admin ini tidak bekerja di toko yang ditentukan")
	}

	// 🚨 ATURAN DEMOSI SUPERVISOR -> STAFF: Seluruh bawahan otomatis dialihkan ke Owner
	if assignment.Role == "supervisor" && input.Role != nil && *input.Role == "staff" {
		_ = s.adminRepo.ReassignSubordinatesToOwner(businessID, assignment.AdminID)
	}

	if input.Role != nil {
		assignment.Role = *input.Role
	}
	if input.SupervisorID != nil {
		if *input.SupervisorID == "" {
			assignment.SupervisorID = nil
		} else {
			assignment.SupervisorID = input.SupervisorID
		}
	}
	if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
		return errors.New("gagal memperbarui penugasan admin")
	}

	if len(input.Shifts) > 0 {
		var shifts []models.BusinessAdminShift
		for _, s := range input.Shifts {
			shifts = append(shifts, models.BusinessAdminShift{
				AssignmentID: assignmentID,
				DayOfWeek:    s.DayOfWeek,
				StartTime:    &s.StartTime,
				EndTime:      &s.EndTime,
			})
		}
		s.adminRepo.ReplaceShifts(assignmentID, shifts)
	}

	businessName := "Toko"
	if biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	// Notifikasi ke Admin: Perubahan Izin & Peran
	go s.notifService.SendNotification(
		[]string{assignment.AdminID},
		ownerID,
		"admin_role_changed",
		assignmentID,
		"Perubahan Peran Toko",
		fmt.Sprintf("Peran atau jadwal kerja Anda di toko %s telah diperbarui menjadi %s.", businessName, assignment.Role),
		map[string]interface{}{
			"assignment_id": assignmentID,
			"business_id":   businessID,
			"new_role":      assignment.Role,
		},
		nil,
	)

	return nil
}

// 5. TERMINATE / PEMECAHAN ADMIN OLEH OWNER/SPV (Grace Period 3 Hari, Read-Only)
func (s *businessAdminService) TerminateAdmin(requesterID, businessID, assignmentID string) error {
	biz, err := s.bizRepo.FindBusinessByID(businessID)
	if err != nil {
		return errors.New("toko tidak ditemukan")
	}

	isOwner := biz.AccountID == requesterID
	isSupervisor := false

	if !isOwner {
		requesterAssignment, errA := s.adminRepo.GetActiveAssignment(requesterID, businessID)
		if errA == nil && requesterAssignment != nil && requesterAssignment.Role == "supervisor" {
			isSupervisor = true
		}
	}

	if !isOwner && !isSupervisor {
		return errors.New("forbidden: Anda tidak memiliki izin untuk memberhentikan admin di toko ini")
	}

	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("penugasan admin tidak ditemukan")
	}

	if assignment.BusinessID != businessID || assignment.Status != "active" {
		return errors.New("admin ini tidak dalam status aktif di toko ini")
	}

	// Supervisor hanya boleh memberhentikan staf di bawahnya
	if isSupervisor {
		if assignment.SupervisorID == nil || *assignment.SupervisorID != requesterID {
			return errors.New("forbidden: supervisor hanya dapat memberhentikan staf di bawahnya")
		}
	}

	// Jika yang diberhentikan adalah Supervisor, alihkan seluruh bawahannya ke Owner
	if assignment.Role == "supervisor" {
		_ = s.adminRepo.ReassignSubordinatesToOwner(businessID, assignment.AdminID)
	}

	scheduledAt := time.Now().Add(72 * time.Hour)
	initiatedBy := "owner"
	if isSupervisor {
		initiatedBy = "supervisor"
	}

	assignment.Status = "scheduled_for_deletion"
	assignment.DeletionScheduledAt = &scheduledAt
	assignment.DeletionInitiatedBy = &initiatedBy

	if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
		return errors.New("gagal menjadwalkan pemberhentian admin")
	}

	businessName := "Toko"
	if biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	// Notifikasi ke Admin: Notice 3 Hari
	go s.notifService.SendNotification(
		[]string{assignment.AdminID},
		requesterID,
		"admin_termination_scheduled",
		assignmentID,
		"Pemberitahuan Penghentian Akses",
		fmt.Sprintf("Akses Anda di toko %s akan dinonaktifkan dalam 3 hari (efektif %s). Selama masa ini akses Anda adalah Read-Only.", businessName, scheduledAt.Format("02 Jan 2006 15:04")),
		map[string]interface{}{
			"assignment_id": assignmentID,
			"business_id":   businessID,
			"effective_at":  scheduledAt.Format(time.RFC3339),
		},
		nil,
	)

	return nil
}

// 6. ADMIN RESIGN MANDIRI DARI TOKO (Grace Period 3 Hari, Read-Only)
func (s *businessAdminService) ResignFromBusiness(adminID, assignmentID string) error {
	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("penugasan tidak ditemukan")
	}

	if assignment.AdminID != adminID {
		return errors.New("forbidden: penugasan ini bukan milik Anda")
	}

	if assignment.Status != "active" {
		return errors.New("Anda tidak dalam status aktif di toko ini")
	}

	biz, _ := s.bizRepo.FindBusinessByID(assignment.BusinessID)
	businessName := "Toko"
	if biz != nil && biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	// Jika Supervisor resign, alihkan bawahannya ke Owner
	if assignment.Role == "supervisor" {
		_ = s.adminRepo.ReassignSubordinatesToOwner(assignment.BusinessID, adminID)
	}

	scheduledAt := time.Now().Add(72 * time.Hour)
	initiatedBy := "self"

	assignment.Status = "scheduled_for_deletion"
	assignment.DeletionScheduledAt = &scheduledAt
	assignment.DeletionInitiatedBy = &initiatedBy

	if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
		return errors.New("gagal memproses pengunduran diri")
	}

	adminName := "Admin"
	if assignment.Admin != nil && assignment.Admin.Account.FirstName != "" {
		adminName = assignment.Admin.Account.FirstName
	}

	recipients := []string{}
	if biz != nil {
		recipients = append(recipients, biz.AccountID)
	}
	if assignment.SupervisorID != nil && *assignment.SupervisorID != "" && *assignment.SupervisorID != biz.AccountID {
		recipients = append(recipients, *assignment.SupervisorID)
	}

	// Notifikasi ke Owner & SPV: Admin Resign
	go s.notifService.SendNotification(
		recipients,
		adminID,
		"admin_resignation_notice",
		assignmentID,
		"Pengunduran Diri Admin",
		fmt.Sprintf("%s mengajukan pengunduran diri dari toko %s dengan masa notice 3 hari (efektif %s).", adminName, businessName, scheduledAt.Format("02 Jan 2006 15:04")),
		map[string]interface{}{
			"assignment_id": assignmentID,
			"business_id":   assignment.BusinessID,
			"admin_name":    adminName,
			"effective_at":  scheduledAt.Format(time.RFC3339),
		},
		nil,
	)

	return nil
}

// 7. BATALKAN PENGHENTIAN AKSES (Cancel Termination / Cancel Resign)
func (s *businessAdminService) CancelTermination(requesterID, assignmentID string) error {
	assignment, err := s.adminRepo.GetAssignmentByID(assignmentID)
	if err != nil {
		return errors.New("penugasan tidak ditemukan")
	}

	if assignment.Status != "scheduled_for_deletion" {
		return errors.New("penugasan ini tidak dalam masa transisi penghentian")
	}

	biz, _ := s.bizRepo.FindBusinessByID(assignment.BusinessID)
	isOwner := biz != nil && biz.AccountID == requesterID
	isAdminSelf := assignment.AdminID == requesterID

	// Validasi izin pembatalan
	if assignment.DeletionInitiatedBy != nil && *assignment.DeletionInitiatedBy == "self" {
		// Jika resign mandiri, hanya Admin itu sendiri atau Owner yang boleh membatalkan
		if !isAdminSelf && !isOwner {
			return errors.New("forbidden: Anda tidak berhak membatalkan pengunduran diri ini")
		}
	} else {
		// Jika dipecat oleh Owner/SPV, hanya Owner atau Supervisor pengundang yang boleh membatalkan
		if !isOwner && (assignment.SupervisorID == nil || *assignment.SupervisorID != requesterID) {
			return errors.New("forbidden: Anda tidak berhak membatalkan pemberhentian ini")
		}
	}

	assignment.Status = "active"
	assignment.DeletionScheduledAt = nil
	assignment.DeletionInitiatedBy = nil

	if err := s.adminRepo.UpdateAssignment(assignment); err != nil {
		return errors.New("gagal membatalkan penghentian akses")
	}

	businessName := "Toko"
	if biz != nil && biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	// Notifikasi pemulihan status
	allRecipients := []string{assignment.AdminID}
	if biz != nil {
		allRecipients = append(allRecipients, biz.AccountID)
	}

	go s.notifService.SendNotification(
		allRecipients,
		requesterID,
		"admin_termination_cancelled",
		assignmentID,
		"Akses Toko Dipulihkan",
		fmt.Sprintf("Penghentian akses untuk toko %s telah dibatalkan. Status admin kembali aktif normal.", businessName),
		map[string]interface{}{
			"assignment_id": assignmentID,
			"business_id":   assignment.BusinessID,
		},
		nil,
	)

	return nil
}

// 8. LIHAT STRUKTUR ADMIN TREE (Owner -> Supervisors -> Staff)
func (s *businessAdminService) GetAdminTree(requesterID, businessID string) (*models.AdminTreeResponse, error) {
	biz, err := s.bizRepo.FindBusinessByID(businessID)
	if err != nil {
		return nil, errors.New("toko tidak ditemukan")
	}

	isOwner := biz.AccountID == requesterID
	if !isOwner {
		activeAssignment, errA := s.adminRepo.GetActiveAssignment(requesterID, businessID)
		if errA != nil || activeAssignment == nil {
			return nil, errors.New("forbidden: Anda tidak memiliki akses ke toko ini")
		}
	}

	ownerAccount, err := s.adminRepo.FindAccountByID(biz.AccountID)
	if err != nil {
		return nil, errors.New("data pemilik toko tidak ditemukan")
	}

	allAssignments, err := s.adminRepo.GetAssignmentsByBusinessID(businessID)
	if err != nil {
		return nil, errors.New("gagal mengambil data admin toko")
	}

	businessName := "Toko"
	if biz.BusinessName != nil {
		businessName = *biz.BusinessName
	}

	response := &models.AdminTreeResponse{
		BusinessID:   businessID,
		BusinessName: businessName,
		Owner: models.AdminTreeNode{
			AccountID: ownerAccount.AccountID,
			Name:      ownerAccount.FirstName + " " + ownerAccount.LastName,
			Username:  ownerAccount.Username,
			AvatarURL: ownerAccount.ProfilePictureUrl,
			Role:      "owner",
		},
		Supervisors: []models.AdminTreeNode{},
		DirectStaff: []models.AdminTreeNode{},
	}

	// Map supervisor nodes
	spvMap := make(map[string]*models.AdminTreeNode)
	var staffAssignments []models.BusinessAdminAssignment

	for _, a := range allAssignments {
		// Tampilkan admin yang berstatus active atau scheduled_for_deletion
		if a.Status != "active" && a.Status != "scheduled_for_deletion" {
			continue
		}

		node := models.AdminTreeNode{
			AccountID:    a.AdminID,
			AssignmentID: a.AssignmentID,
			Name:         a.Admin.Account.FirstName + " " + a.Admin.Account.LastName,
			Username:     a.Admin.Account.Username,
			AvatarURL:    a.Admin.Account.ProfilePictureUrl,
			Role:         a.Role,
			Status:       a.Status,
			Shifts:       a.Shifts,
			Subordinates: []models.AdminTreeNode{},
			// Supervisor default semua true, staff ambil dari DB
			Permissions: func() models.AdminPermissions {
				if a.Role == "supervisor" {
					return models.AdminPermissions{}.AllTrue()
				}
				if a.Permissions == nil {
					return models.AdminPermissions{}
				}
				return a.Permissions
			}(),
		}

		if a.Role == "supervisor" {
			spvNode := node
			spvMap[a.AdminID] = &spvNode
		} else {
			staffAssignments = append(staffAssignments, a)
		}
	}

	// Masukkan staff ke supervisor masing-masing atau ke DirectStaff
	for _, a := range staffAssignments {
		perms := a.Permissions
		if perms == nil {
			perms = models.AdminPermissions{}
		}
		node := models.AdminTreeNode{
			AccountID:    a.AdminID,
			AssignmentID: a.AssignmentID,
			Name:         a.Admin.Account.FirstName + " " + a.Admin.Account.LastName,
			Username:     a.Admin.Account.Username,
			AvatarURL:    a.Admin.Account.ProfilePictureUrl,
			Role:         a.Role,
			Status:       a.Status,
			Shifts:       a.Shifts,
			Permissions:  perms,
		}

		if a.SupervisorID != nil && spvMap[*a.SupervisorID] != nil {
			spvMap[*a.SupervisorID].Subordinates = append(spvMap[*a.SupervisorID].Subordinates, node)
		} else {
			response.DirectStaff = append(response.DirectStaff, node)
		}
	}

	for _, spv := range spvMap {
		response.Supervisors = append(response.Supervisors, *spv)
	}

	return response, nil
}

// 9. LIST ADMIN BERDASARKAN OWNER (ALL BUSINESSES)
func (s *businessAdminService) GetAdminsByOwner(ownerID string) ([]models.AdminTreeResponse, error) {
	// 1. Ambil semua bisnis milik owner
	businesses, err := s.bizRepo.GetBusinessListWithPreviews(ownerID, "", 0, nil, "", 1000) // Ambil cukup banyak
	if err != nil {
		return nil, errors.New("gagal mengambil daftar toko")
	}

	var allTrees []models.AdminTreeResponse
	for _, biz := range businesses {
		// 2. Untuk setiap bisnis, buat Admin Tree
		tree, err := s.GetAdminTree(ownerID, biz.BusinessID)
		if err == nil && tree != nil {
			allTrees = append(allTrees, *tree)
		}
	}

	return allTrees, nil
}

// 10. LIST TOKO TEMPAT ADMIN BEKERJA (UNTUK ADMIN)
func (s *businessAdminService) GetMyAssignments(adminID string) ([]models.AdminAssignmentListItem, error) {
	assignments, err := s.adminRepo.GetAssignmentsByAdminID(adminID)
	if err != nil {
		return nil, err
	}

	var response []models.AdminAssignmentListItem
	for _, assignment := range assignments {
		var mainImage string
		if assignment.Business != nil {
			if assignment.Business.LogoURL != nil && *assignment.Business.LogoURL != "" {
				mainImage = *assignment.Business.LogoURL
			} else if len(assignment.Business.BusinessImages) > 0 {
				mainImage = assignment.Business.BusinessImages[0]
			}

			var fullAddress string
			if assignment.Business.AddressDetail != nil {
				fullAddress += *assignment.Business.AddressDetail
			}
			if assignment.Business.City != nil {
				if fullAddress != "" {
					fullAddress += ", "
				}
				fullAddress += *assignment.Business.City
			}
			if assignment.Business.Province != nil {
				if fullAddress != "" {
					fullAddress += ", "
				}
				fullAddress += *assignment.Business.Province
			}

			bizName := ""
			if assignment.Business.BusinessName != nil {
				bizName = *assignment.Business.BusinessName
			}

			ownerName := "" // Biarkan kosong atau jika diperlukan bisa diambil dari repo

			// Permissions: supervisor default semua true, staff dari DB
			perms := assignment.Permissions
			if assignment.Role == "supervisor" {
				perms = models.AdminPermissions{}.AllTrue()
			} else if perms == nil {
				perms = models.AdminPermissions{}
			}

			item := models.AdminAssignmentListItem{
				AssignmentID: assignment.AssignmentID,
				Role:         assignment.Role,
				Status:       assignment.Status,
				Permissions:  perms,
				Business: models.BusinessListItem{
					BusinessID:          assignment.BusinessID,
					BusinessName:        bizName,
					OwnerName:           ownerName,
					LogoURL:             mainImage,
					Address:             fullAddress,
					AverageRating:       0,
					TotalProductReviews: 0,
				},
			}
			response = append(response, item)
		}
	}

	return response, nil
}

// 11. GET PERMISSIONS (Owner atau Supervisor melihat izin admin tertentu)
func (s *businessAdminService) GetPermissions(requesterID, businessID, targetAssignmentID string) (*models.PermissionResponse, error) {
	target, err := s.adminRepo.GetPermissions(targetAssignmentID)
	if err != nil {
		return nil, errors.New("assignment tidak ditemukan")
	}

	if businessID == "" {
		businessID = target.BusinessID
	}

	biz, err := s.bizRepo.FindBusinessByID(businessID)
	if err != nil {
		return nil, errors.New("toko tidak ditemukan")
	}

	isOwner := biz.AccountID == requesterID
	if !isOwner {
		requesterAssignment, errA := s.adminRepo.GetActiveAssignment(requesterID, businessID)
		if errA != nil || requesterAssignment == nil {
			return nil, errors.New("forbidden: Anda tidak memiliki akses ke toko ini")
		}
	}

	if target.BusinessID != businessID {
		return nil, errors.New("forbidden: assignment tidak termasuk dalam toko ini")
	}

	adminName := ""
	if target.Admin != nil && target.Admin.Account != nil {
		adminName = target.Admin.Account.FirstName + " " + target.Admin.Account.LastName
	}

	perms := target.Permissions
	if target.Role == "supervisor" {
		perms = models.AdminPermissions{}.AllTrue()
	} else if perms == nil {
		perms = models.AdminPermissions{}
	}

	return &models.PermissionResponse{
		AssignmentID: target.AssignmentID,
		AdminID:      target.AdminID,
		AdminName:    adminName,
		Role:         target.Role,
		Permissions:  perms,
	}, nil
}

// 12. UPDATE PERMISSIONS (Owner atau Supervisor mengatur izin admin)
func (s *businessAdminService) UpdatePermissions(requesterID, businessID, targetAssignmentID string, input models.UpdatePermissionsInput) (*models.PermissionResponse, error) {
	target, err := s.adminRepo.GetPermissions(targetAssignmentID)
	if err != nil {
		return nil, errors.New("assignment tidak ditemukan")
	}

	if businessID == "" {
		businessID = target.BusinessID
	}

	biz, err := s.bizRepo.FindBusinessByID(businessID)
	if err != nil {
		return nil, errors.New("toko tidak ditemukan")
	}

	isOwner := biz.AccountID == requesterID

	var requesterAssignment *models.BusinessAdminAssignment
	if !isOwner {
		a, errA := s.adminRepo.GetActiveAssignment(requesterID, businessID)
		if errA != nil || a == nil || a.Role != "supervisor" {
			return nil, errors.New("forbidden: hanya Owner atau Supervisor aktif yang dapat mengubah izin admin")
		}
		requesterAssignment = a
	}
	if target.BusinessID != businessID {
		return nil, errors.New("forbidden: assignment tidak termasuk dalam toko ini")
	}
	if target.Status != "active" && target.Status != "scheduled_for_deletion" {
		return nil, errors.New("admin ini tidak dalam status aktif")
	}

	// Supervisor tidak boleh mengubah permission sesama supervisor
	if !isOwner && target.Role == "supervisor" {
		return nil, errors.New("forbidden: Supervisor tidak dapat mengubah izin sesama Supervisor")
	}

	// Supervisor hanya boleh mengubah permission bawahannya langsung
	if !isOwner && requesterAssignment != nil {
		if target.SupervisorID == nil || *target.SupervisorID != requesterID {
			return nil, errors.New("forbidden: Anda hanya dapat mengubah izin staf yang langsung berada di bawah Anda")
		}
	}

	// Validasi key permission — tolak unknown keys
	validKeys := make(map[string]bool, len(models.AllPermissions))
	for _, k := range models.AllPermissions {
		validKeys[k] = true
	}
	for key := range input.Permissions {
		if !validKeys[key] {
			return nil, fmt.Errorf("permission key tidak dikenal: '%s'", key)
		}
	}

	// Simpan permission yang dikirim (partial update: key tidak dikirim tidak berubah)
	// Merge ke permissions existing
	existing := target.Permissions
	if existing == nil {
		existing = models.AdminPermissions{}
	}
	for key, val := range input.Permissions {
		existing[key] = val
	}

	if err := s.adminRepo.UpdatePermissions(targetAssignmentID, existing); err != nil {
		return nil, errors.New("gagal memperbarui izin admin")
	}

	adminName := ""
	if target.Admin != nil && target.Admin.Account != nil {
		adminName = target.Admin.Account.FirstName + " " + target.Admin.Account.LastName
	}

	return &models.PermissionResponse{
		AssignmentID: target.AssignmentID,
		AdminID:      target.AdminID,
		AdminName:    adminName,
		Role:         target.Role,
		Permissions:  existing,
	}, nil
}
