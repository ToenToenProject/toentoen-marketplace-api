package transfer

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/redis/go-redis/v9"
)

type BusinessTransferService interface {
	CheckTarget(ownerID, businessID, targetIdentifier string) (*models.TransferTargetPreviewResponse, error)
	InitiateTransfer(ownerID, businessID string, input models.InitiateTransferInput) (map[string]interface{}, error)
	VerifyOTPAndCreateTransfer(ownerID, businessID string, input models.VerifyTransferOTPInput) (*models.BusinessTransferRequest, error)
	CancelTransfer(ownerID, businessID string) error
	GetIncomingTransfers(targetAccountID string) ([]models.IncomingTransferListItem, error)
	RespondTransfer(targetAccountID, transferID string, input models.RespondTransferInput) error
}

type businessTransferService struct {
	repo         BusinessTransferRepository
	notifService notification.NotificationService
}

func NewBusinessTransferService(repo BusinessTransferRepository, notifService notification.NotificationService) BusinessTransferService {
	return &businessTransferService{
		repo:         repo,
		notifService: notifService,
	}
}

func (s *businessTransferService) CheckTarget(ownerID, businessID, targetIdentifier string) (*models.TransferTargetPreviewResponse, error) {
	businessID = strings.TrimSpace(businessID)
	targetIdentifier = strings.TrimSpace(targetIdentifier)

	if businessID == "" {
		return nil, errors.New("business_id wajib diisi")
	}
	if targetIdentifier == "" {
		return nil, errors.New("Email atau nomor telepon target wajib diisi")
	}

	// 1. Validasi kepemilikan bisnis
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil || business == nil {
		return nil, errors.New("Bisnis/Toko tidak ditemukan")
	}
	if business.AccountID != ownerID {
		return nil, errors.New("Anda bukan pemilik sah dari bisnis ini")
	}

	// 2. Cari akun target
	target, err := s.repo.FindAccountByIdentifier(targetIdentifier)
	if err != nil || target == nil {
		return nil, errors.New("Akun tujuan dengan identitas tersebut tidak ditemukan")
	}

	// 3. Validasi akun target
	if target.AccountID == ownerID {
		return nil, errors.New("Anda tidak dapat mentransfer bisnis kepada akun Anda sendiri")
	}
	if target.AccountStatus != "active" {
		return nil, errors.New("Akun tujuan sedang tidak aktif atau belum diverifikasi")
	}
	if target.Role != "owner" {
		return nil, errors.New("Transfer bisnis hanya dapat dilakukan kepada akun yang berstatus sebagai Owner")
	}

	targetName := target.FirstName
	if target.LastName != "" {
		targetName += " " + target.LastName
	}
	if strings.TrimSpace(targetName) == "" {
		targetName = target.Username
	}

	return &models.TransferTargetPreviewResponse{
		TargetAccountID: target.AccountID,
		TargetName:      targetName,
		TargetEmail:     target.Email,
		TargetPhone:     target.PhoneNumber,
		TargetAvatar:    target.ProfilePictureUrl,
		TargetRole:      target.Role,
	}, nil
}

func (s *businessTransferService) InitiateTransfer(ownerID, businessID string, input models.InitiateTransferInput) (map[string]interface{}, error) {
	businessID = strings.TrimSpace(businessID)
	targetIdentifier := strings.TrimSpace(input.TargetIdentifier)

	if businessID == "" {
		return nil, errors.New("business_id wajib diisi")
	}
	if targetIdentifier == "" {
		return nil, errors.New("target_identifier wajib diisi")
	}

	// 1. Validasi target
	_, err := s.CheckTarget(ownerID, businessID, targetIdentifier)
	if err != nil {
		return nil, err
	}

	// 2. Cek apakah ada transfer yang masih aktif
	existing, _ := s.repo.GetActiveTransferByBusiness(businessID)
	if existing != nil {
		return nil, errors.New("Bisnis ini sudah memiliki permohonan transfer yang sedang menunggu persetujuan. Silakan batalkan permohonan lama terlebih dahulu.")
	}

	// 3. Ambil data akun Owner saat ini untuk tujuan pengiriman OTP
	currentOwner, err := s.repo.FindAccountByID(ownerID)
	if err != nil || currentOwner == nil {
		return nil, errors.New("Data akun pemilik tidak ditemukan")
	}

	// 4. Generate OTP
	otp := utils.GenerateOTP()
	otpKey := fmt.Sprintf("biz_transfer_otp:%s:%s", businessID, ownerID)
	targetKey := fmt.Sprintf("biz_transfer_target:%s:%s", businessID, ownerID)

	// Simpan ke Redis (TTL 5 menit)
	if err := config.RDB.Set(config.Ctx, otpKey, otp, 5*time.Minute).Err(); err != nil {
		return nil, errors.New("Gagal memproses OTP pada sistem")
	}
	_ = config.RDB.Set(config.Ctx, targetKey, targetIdentifier, 5*time.Minute).Err()

	// 5. Kirimkan OTP ke kontak Owner Lama
	destination := ""
	channel := strings.ToLower(input.Channel)
	if channel == "whatsapp" && currentOwner.PhoneNumber != nil && *currentOwner.PhoneNumber != "" {
		go utils.SendWhatsAppOTP(*currentOwner.PhoneNumber, otp)
		destination = utils.MaskPhoneNumber(*currentOwner.PhoneNumber)
	} else if currentOwner.Email != nil && *currentOwner.Email != "" {
		channel = "email"
		go utils.SendEmailOTP(*currentOwner.Email, otp)
		destination = utils.MaskEmail(*currentOwner.Email)
	} else if currentOwner.PhoneNumber != nil && *currentOwner.PhoneNumber != "" {
		channel = "whatsapp"
		go utils.SendWhatsAppOTP(*currentOwner.PhoneNumber, otp)
		destination = utils.MaskPhoneNumber(*currentOwner.PhoneNumber)
	} else {
		return nil, errors.New("Akun Anda tidak memiliki email atau nomor telepon yang valid untuk verifikasi OTP")
	}

	return map[string]interface{}{
		"channel":            channel,
		"destination":        destination,
		"expires_in_seconds": 300,
	}, nil
}

func (s *businessTransferService) VerifyOTPAndCreateTransfer(ownerID, businessID string, input models.VerifyTransferOTPInput) (*models.BusinessTransferRequest, error) {
	businessID = strings.TrimSpace(businessID)
	targetIdentifier := strings.TrimSpace(input.TargetIdentifier)
	inputOTP := strings.TrimSpace(input.OTP)

	if businessID == "" {
		return nil, errors.New("business_id wajib diisi")
	}
	if targetIdentifier == "" {
		return nil, errors.New("target_identifier wajib diisi")
	}
	if inputOTP == "" {
		return nil, errors.New("OTP wajib diisi")
	}

	// 1. Verifikasi OTP dari Redis
	otpKey := fmt.Sprintf("biz_transfer_otp:%s:%s", businessID, ownerID)
	storedOTP, err := config.RDB.Get(config.Ctx, otpKey).Result()
	if err == redis.Nil || storedOTP != inputOTP {
		return nil, errors.New("Kode OTP tidak valid atau sudah kadaluarsa")
	}

	// Hapus OTP dari Redis
	config.RDB.Del(config.Ctx, otpKey)
	config.RDB.Del(config.Ctx, fmt.Sprintf("biz_transfer_target:%s:%s", businessID, ownerID))

	// 2. Validasi target akun
	target, err := s.repo.FindAccountByIdentifier(targetIdentifier)
	if err != nil || target == nil {
		return nil, errors.New("Akun tujuan tidak ditemukan")
	}
	if target.Role != "owner" {
		return nil, errors.New("Transfer bisnis hanya dapat dilakukan kepada akun yang berstatus sebagai Owner")
	}

	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil || business == nil {
		return nil, errors.New("Bisnis tidak ditemukan")
	}
	if business.AccountID != ownerID {
		return nil, errors.New("Anda bukan pemilik sah dari bisnis ini")
	}
	if target.AccountID == ownerID {
		return nil, errors.New("Anda tidak dapat mentransfer bisnis kepada akun Anda sendiri")
	}

	// 3. Buat transfer request
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 hari masa berlaku

	req := &models.BusinessTransferRequest{
		TransferID:       utils.GenerateUUID("BTF"),
		BusinessID:       businessID,
		CurrentOwnerID:   ownerID,
		TargetAccountID:  target.AccountID,
		TargetIdentifier: targetIdentifier,
		Status:           "pending_acceptance",
		OTPVerifiedAt:    &now,
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateTransferRequest(req); err != nil {
		return nil, errors.New("Gagal membuat permohonan transfer bisnis")
	}

	req.Business = business
	req.TargetAccount = target

	// 4. Kirim notifikasi ke Target Owner
	bizName := "Toko"
	if business.BusinessName != nil && *business.BusinessName != "" {
		bizName = *business.BusinessName
	}

	if s.notifService != nil {
		title := "Undangan Transfer Kepemilikan Toko"
		body := fmt.Sprintf("Anda menerima undangan pengalihan kepemilikan toko '%s'. Silakan periksa dan beri persetujuan.", bizName)
		data := map[string]interface{}{
			"transfer_id": req.TransferID,
			"business_id": businessID,
			"type":        "business_transfer_invitation",
		}
		s.notifService.SendNotification([]string{target.AccountID}, ownerID, "business_transfer_invitation", req.TransferID, title, body, data, nil)
	}

	return req, nil
}

func (s *businessTransferService) CancelTransfer(ownerID, businessID string) error {
	activeTransfer, err := s.repo.GetActiveTransferByBusiness(businessID)
	if err != nil || activeTransfer == nil {
		return errors.New("Tidak ada permohonan transfer aktif yang dapat dibatalkan")
	}
	if activeTransfer.CurrentOwnerID != ownerID {
		return errors.New("Anda tidak memiliki hak untuk membatalkan transfer ini")
	}

	if err := s.repo.CancelTransfer(activeTransfer.TransferID); err != nil {
		return errors.New("Gagal membatalkan permohonan transfer")
	}

	// Kirim notifikasi pembatalan ke target
	if s.notifService != nil {
		title := "Permohonan Transfer Toko Dibatalkan"
		body := "Permohonan pengalihan kepemilikan toko sebelumnya telah dibatalkan oleh pemilik toko."
		data := map[string]interface{}{
			"transfer_id": activeTransfer.TransferID,
			"business_id": businessID,
		}
		s.notifService.SendNotification([]string{activeTransfer.TargetAccountID}, ownerID, "business_transfer_cancelled", activeTransfer.TransferID, title, body, data, nil)
	}

	return nil
}

func (s *businessTransferService) GetIncomingTransfers(targetAccountID string) ([]models.IncomingTransferListItem, error) {
	reqs, err := s.repo.GetIncomingTransfers(targetAccountID)
	if err != nil {
		return nil, err
	}

	var results []models.IncomingTransferListItem
	for _, r := range reqs {
		var bizName string
		var bizLogo *string
		if r.Business != nil {
			if r.Business.BusinessName != nil {
				bizName = *r.Business.BusinessName
			}
			bizLogo = r.Business.LogoURL
		}

		var ownerName string
		if r.CurrentOwner != nil {
			ownerName = r.CurrentOwner.FirstName
			if r.CurrentOwner.LastName != "" {
				ownerName += " " + r.CurrentOwner.LastName
			}
			if strings.TrimSpace(ownerName) == "" {
				ownerName = r.CurrentOwner.Username
			}
		}

		results = append(results, models.IncomingTransferListItem{
			TransferID:       r.TransferID,
			BusinessID:       r.BusinessID,
			BusinessName:     bizName,
			BusinessLogo:     bizLogo,
			CurrentOwnerID:   r.CurrentOwnerID,
			CurrentOwnerName: ownerName,
			Status:           r.Status,
			ExpiresAt:        r.ExpiresAt,
			CreatedAt:        r.CreatedAt,
		})
	}

	return results, nil
}

func (s *businessTransferService) RespondTransfer(targetAccountID, transferID string, input models.RespondTransferInput) error {
	transferID = strings.TrimSpace(transferID)
	action := strings.ToLower(strings.TrimSpace(input.Action))

	if transferID == "" {
		return errors.New("transfer_id wajib diisi")
	}
	if action != "accept" && action != "reject" {
		return errors.New("Aksi harus 'accept' atau 'reject'")
	}

	transfer, err := s.repo.GetTransferByID(transferID)
	if err != nil || transfer == nil {
		return errors.New("Permohonan transfer bisnis tidak ditemukan")
	}

	if transfer.TargetAccountID != targetAccountID {
		return errors.New("Anda tidak memiliki hak untuk merespon permohonan transfer ini")
	}

	if transfer.Status != "pending_acceptance" {
		return fmt.Errorf("Permohonan transfer sudah berstatus '%s'", transfer.Status)
	}

	if time.Now().After(transfer.ExpiresAt) {
		return errors.New("Permohonan transfer sudah kadaluarsa")
	}

	bizName := "Toko"
	if transfer.Business != nil && transfer.Business.BusinessName != nil && *transfer.Business.BusinessName != "" {
		bizName = *transfer.Business.BusinessName
	}

	if action == "reject" {
		if err := s.repo.RejectTransfer(transferID); err != nil {
			return errors.New("Gagal menolak transfer bisnis")
		}

		// Notifikasi ke Owner Lama
		if s.notifService != nil {
			title := "Transfer Toko Ditolak"
			body := fmt.Sprintf("Calon pemilik baru telah menolak pengalihan kepemilikan toko '%s'.", bizName)
			s.notifService.SendNotification([]string{transfer.CurrentOwnerID}, targetAccountID, "business_transfer_rejected", transferID, title, body, nil, nil)
		}

		return nil
	}

	// Action == "accept"
	// Pastikan target masih berstatus owner
	targetAcc, err := s.repo.FindAccountByID(targetAccountID)
	if err != nil || targetAcc == nil || targetAcc.Role != "owner" {
		return errors.New("Akun Anda harus berstatus sebagai Owner untuk menerima transfer bisnis")
	}

	if err := s.repo.AcceptTransfer(transferID, transfer.BusinessID, targetAccountID); err != nil {
		return errors.New("Gagal memproses persetujuan transfer bisnis: " + err.Error())
	}

	// Notifikasi Sukses ke Kedua Belah Pihak
	if s.notifService != nil {
		// Ke Owner Lama
		titleOld := "Transfer Toko Selesai"
		bodyOld := fmt.Sprintf("Kepemilikan toko '%s' telah berhasil dialihkan kepada pemilik baru.", bizName)
		s.notifService.SendNotification([]string{transfer.CurrentOwnerID}, targetAccountID, "business_transfer_completed", transferID, titleOld, bodyOld, nil, nil)

		// Ke Owner Baru
		titleNew := "Selamat! Kepemilikan Toko Aktif"
		bodyNew := fmt.Sprintf("Anda sekarang resmi menjadi pemilik toko '%s'. Toko telah ditambahkan ke dashboard Anda.", bizName)
		s.notifService.SendNotification([]string{targetAccountID}, transfer.CurrentOwnerID, "business_transfer_accepted", transferID, titleNew, bodyNew, nil, nil)
	}

	return nil
}
