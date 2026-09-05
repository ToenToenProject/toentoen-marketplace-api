package report

import (
	"errors"
	"fmt"
	"strings"

	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

// Request payload struct legacy
type SubmitReportRequest struct {
	Reason       string   `json:"reason"`
	Description  *string  `json:"description"`
	EvidenceURLs []string `json:"evidence_urls"`
}

type Service interface {
	SubmitReport(accountID, businessID string, req SubmitReportRequest) error
	CreateUniversalReport(accountID string, input models.CreateReportInput) error
	GetReports(filter models.ReportFilterQuery) ([]models.ReportResponse, int64, error)
	GetUserReports(accountID string, page, limit int) ([]models.ReportResponse, int64, error)
	GetReportDetail(reportID string) (*models.ReportResponse, error)
	DeleteReport(reportID, requesterAccountID, role string) error
	UpdateReportStatus(reportID string, input models.UpdateReportStatusInput, resolvedBy string) error
}

type service struct {
	repo         Repository
	notifService notification.NotificationService
}

func NewService(r Repository, notifService notification.NotificationService) Service {
	return &service{
		repo:         r,
		notifService: notifService,
	}
}

func (s *service) SubmitReport(accountID, businessID string, req SubmitReportRequest) error {
	exists, err := s.repo.CheckBusinessExists(businessID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("bisnis tidak ditemukan")
	}

	if req.Reason == "" {
		return errors.New("alasan laporan wajib diisi")
	}

	if req.Reason == "Other" && (req.Description == nil || *req.Description == "") {
		return errors.New("deskripsi wajib diisi ketika alasan adalah 'Lainnya'")
	}

	var evidences []models.BusinessReportEvidence
	for _, url := range req.EvidenceURLs {
		if url != "" {
			evidences = append(evidences, models.BusinessReportEvidence{
				FileURL: url,
			})
		}
	}

	report := models.BusinessReport{
		AccountID:   accountID,
		BusinessID:  businessID,
		Reason:      req.Reason,
		Description: req.Description,
		Status:      "pending",
		Evidence:    evidences,
	}

	return s.repo.CreateReport(&report)
}

func normalizeReasonAndPriority(rawReason string) (string, string) {
	clean := strings.TrimSpace(rawReason)
	lower := strings.ToLower(clean)

	switch {
	case strings.Contains(lower, "fake") || strings.Contains(lower, "fraud"):
		return "Fake or fraudulent post", "high"
	case strings.Contains(lower, "inaccurate") || strings.Contains(lower, "innacurate"):
		return "Inaccurate posts information", "medium"
	case strings.Contains(lower, "policy") || strings.Contains(lower, "violation"):
		return "Policy violation", "high"
	case strings.Contains(lower, "spam") || strings.Contains(lower, "misleading"):
		return "Spam or misleading information", "medium"
	case strings.Contains(lower, "other") || lower == "lainnya":
		return "Other", "low"
	default:
		// Jika format custom lainnya
		return clean, "medium"
	}
}

func (s *service) CreateUniversalReport(accountID string, input models.CreateReportInput) error {
	accountID = strings.TrimSpace(accountID)
	targetID := strings.TrimSpace(input.TargetID)
	if targetID == "" {
		return errors.New("target_id wajib diisi")
	}

	targetType := strings.ToLower(strings.TrimSpace(input.TargetType))
	if targetType == "" {
		// Auto detect dari prefix ID
		upperID := strings.ToUpper(targetID)
		if strings.HasPrefix(upperID, "REV") || strings.HasPrefix(upperID, "PRV") {
			targetType = "review"
		} else if strings.HasPrefix(upperID, "CMT") || strings.HasPrefix(upperID, "PRC") || strings.HasPrefix(upperID, "PCM") || strings.HasPrefix(upperID, "RVC") {
			targetType = "comment"
		} else if strings.HasPrefix(upperID, "PST") {
			targetType = "post"
		} else if strings.HasPrefix(upperID, "PRD") {
			targetType = "product"
		} else if strings.HasPrefix(upperID, "BUS") {
			targetType = "business"
		} else {
			targetType = "other"
		}
	}

	// 1. Cek apakah user sudah pernah melaporkan target ini sebelumnya (1 Akun 1 Report)
	hasReported, errReported := s.repo.HasUserReportedTarget(accountID, targetID)
	if errReported == nil && hasReported {
		return errors.New("Anda sudah pernah melaporkan konten ini sebelumnya")
	}

	// 2. Validasi target existence
	exists, err := s.repo.CheckTargetExists(targetType, targetID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("target yang ingin dilaporkan tidak ditemukan")
	}

	if strings.TrimSpace(input.Reason) == "" {
		return errors.New("alasan pelaporan (reason) wajib diisi")
	}

	standardReason, priority := normalizeReasonAndPriority(input.Reason)
	if standardReason == "Other" && (input.Description == nil || strings.TrimSpace(*input.Description) == "") {
		return errors.New("deskripsi wajib diisi ketika memilih alasan 'Other'")
	}

	evidenceList := input.Evidence
	if evidenceList == nil {
		evidenceList = []string{}
	}

	report := &models.Report{
		ReportID:    utils.GenerateUUID("RPT"),
		AccountID:   accountID,
		TargetType:  targetType,
		TargetID:    targetID,
		Reason:      standardReason,
		Description: input.Description,
		Evidence:    evidenceList,
		Status:      "pending",
		Priority:    priority,
	}

	return s.repo.CreateUniversalReport(report)
}

func (s *service) toReportResponse(r models.Report) models.ReportResponse {
	reporterName := "Pengguna"
	var reporterAvatar *string
	if r.Account != nil {
		reporterName = strings.TrimSpace(r.Account.FirstName + " " + r.Account.LastName)
		if reporterName == "" {
			reporterName = "Pengguna"
		}
		reporterAvatar = r.Account.ProfilePictureUrl
	}

	evidences := r.Evidence
	if evidences == nil {
		evidences = []string{}
	}

	priority := r.Priority
	if priority == "" {
		priority = "medium"
	}

	return models.ReportResponse{
		ReportID:       r.ReportID,
		AccountID:      r.AccountID,
		ReporterName:   reporterName,
		ReporterAvatar: reporterAvatar,
		TargetType:     r.TargetType,
		TargetID:       r.TargetID,
		Reason:         r.Reason,
		Description:    r.Description,
		Evidence:       evidences,
		Status:         r.Status,
		Priority:       priority,
		AdminNote:      r.AdminNote,
		ResolvedAt:     r.ResolvedAt,
		ResolvedBy:     r.ResolvedBy,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func (s *service) GetReports(filter models.ReportFilterQuery) ([]models.ReportResponse, int64, error) {
	reports, total, err := s.repo.GetReportsPaginated(filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.ReportResponse, 0, len(reports))
	for _, r := range reports {
		responses = append(responses, s.toReportResponse(r))
	}
	return responses, total, nil
}

func (s *service) GetUserReports(accountID string, page, limit int) ([]models.ReportResponse, int64, error) {
	reports, total, err := s.repo.GetUserReports(accountID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.ReportResponse, 0, len(reports))
	for _, r := range reports {
		responses = append(responses, s.toReportResponse(r))
	}
	return responses, total, nil
}

func (s *service) GetReportDetail(reportID string) (*models.ReportResponse, error) {
	report, err := s.repo.GetReportByID(reportID)
	if err != nil {
		return nil, errors.New("laporan tidak ditemukan")
	}

	resp := s.toReportResponse(*report)
	return &resp, nil
}

func (s *service) DeleteReport(reportID, requesterAccountID, role string) error {
	report, err := s.repo.GetReportByID(reportID)
	if err != nil {
		return errors.New("laporan tidak ditemukan")
	}

	isOwner := strings.EqualFold(strings.TrimSpace(report.AccountID), strings.TrimSpace(requesterAccountID))
	isAdmin := strings.EqualFold(role, "admin") || strings.EqualFold(role, "platform_admin") || strings.EqualFold(role, "superadmin")

	if !isOwner && !isAdmin {
		return errors.New("tidak memiliki izin untuk menghapus laporan ini")
	}

	return s.repo.DeleteReport(reportID)
}

func (s *service) UpdateReportStatus(reportID string, input models.UpdateReportStatusInput, resolvedBy string) error {
	report, err := s.repo.GetReportByID(reportID)
	if err != nil {
		return errors.New("laporan tidak ditemukan")
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"in_review": true,
		"resolved":  true,
		"rejected":  true,
		"dismissed": true,
	}

	cleanStatus := strings.ToLower(strings.TrimSpace(input.Status))
	if !validStatuses[cleanStatus] {
		return errors.New("status laporan tidak valid (pending, in_review, resolved, rejected, dismissed)")
	}

	adminNote := ""
	if input.AdminNote != nil {
		adminNote = *input.AdminNote
	}

	if err := s.repo.UpdateReportResolution(reportID, cleanStatus, adminNote, resolvedBy); err != nil {
		return err
	}

	// 1. Eksekusi Aksi Takedown Konten jika status Resolved (atau jika action == delete_target)
	if cleanStatus == "resolved" {
		shouldDelete := true
		if input.Action != nil && *input.Action == "none" {
			shouldDelete = false
		}
		if shouldDelete {
			_ = s.repo.TakedownTargetContent(report.TargetType, report.TargetID)
		}
	}

	// 2. Kirim Notifikasi Resolusi ke Pelapor (Resolution Loop)
	if s.notifService != nil && report.AccountID != "" {
		go func() {
			notifData := map[string]interface{}{
				"report_id":   reportID,
				"target_type": report.TargetType,
				"target_id":   report.TargetID,
				"status":      cleanStatus,
			}

			if cleanStatus == "resolved" {
				s.notifService.SendNotification(
					[]string{report.AccountID},
					resolvedBy,
					"report_resolved",
					reportID,
					"Laporan Anda Telah Ditindaklanjuti",
					fmt.Sprintf("Terima kasih atas kontribusi Anda. Konten yang Anda laporkan (%s) telah kami tindaklanjuti dan dihapus sesuai Pedoman Komunitas.", report.TargetID),
					notifData,
					nil,
				)
			} else if cleanStatus == "rejected" || cleanStatus == "dismissed" {
				s.notifService.SendNotification(
					[]string{report.AccountID},
					resolvedBy,
					"report_rejected",
					reportID,
					"Laporan Anda Selesai Ditinjau",
					fmt.Sprintf("Kami telah memeriksa konten yang Anda laporkan (%s). Saat ini kami tidak menemukan pelanggaran Pedoman Komunitas.", report.TargetID),
					notifData,
					nil,
				)
			}
		}()
	}

	return nil
}
