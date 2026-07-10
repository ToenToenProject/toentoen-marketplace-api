package report

import (
	"errors"

	"toentoen-backend/internal/domain/shared/models"
)

// Request payload struct
type SubmitReportRequest struct {
	Reason       string   `json:"reason"`
	Description  *string  `json:"description"`
	EvidenceURLs []string `json:"evidence_urls"`
}

type Service interface {
	SubmitReport(accountID, businessID string, req SubmitReportRequest) error
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
	// Validasi apakah BusinessID tersebut benar-benar ada
	exists, err := s.repo.CheckBusinessExists(businessID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("business not found")
	}

	// Validasi payload
	if req.Reason == "" {
		return errors.New("reason is required")
	}

	// Jika Reason adalah "Other", sebaiknya description tidak kosong (opsional sesuai bisnis logik)
	if req.Reason == "Other" && (req.Description == nil || *req.Description == "") {
		return errors.New("description is required when reason is 'Other'")
	}

	// Buat Evidence slice
	var evidences []models.BusinessReportEvidence
	for _, url := range req.EvidenceURLs {
		if url != "" {
			evidences = append(evidences, models.BusinessReportEvidence{
				FileURL: url,
			})
		}
	}

	// Buat Model Report
	report := models.BusinessReport{
		AccountID:   accountID,
		BusinessID:  businessID,
		Reason:      req.Reason,
		Description: req.Description,
		Status:      "pending",
		Evidence:    evidences,
	}

	// Simpan
	return s.repo.CreateReport(&report)
}
