package report

import (
	"errors"
	"strings"

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
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{
		repo: r,
	}
}

func (s *service) SubmitReport(accountID, businessID string, req SubmitReportRequest) error {
	exists, err := s.repo.CheckBusinessExists(businessID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("business not found")
	}

	if req.Reason == "" {
		return errors.New("reason is required")
	}

	if req.Reason == "Other" && (req.Description == nil || *req.Description == "") {
		return errors.New("description is required when reason is 'Other'")
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

func (s *service) CreateUniversalReport(accountID string, input models.CreateReportInput) error {
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

	// Validasi target existence
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

	evidenceList := input.Evidence
	if evidenceList == nil {
		evidenceList = []string{}
	}

	report := &models.Report{
		ReportID:    utils.GenerateUUID("RPT"),
		AccountID:   accountID,
		TargetType:  targetType,
		TargetID:    targetID,
		Reason:      input.Reason,
		Description: input.Description,
		Evidence:    evidenceList,
		Status:      "pending",
	}

	return s.repo.CreateUniversalReport(report)
}
