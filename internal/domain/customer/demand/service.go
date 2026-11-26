package demand

import (
	"toentoen-backend/internal/domain/merchant/business"
	"errors"
	"fmt"
	"log"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type DemandService interface {
	CreateSignal(accountID string, input models.CreateSignalInput) (*models.DemandSignal, error)
	ExploreOpenSignals(category string) ([]models.DemandSignal, error)
	SubmitResponse(accountID, role, signalID string, input models.CreateResponseInput) (*models.DemandResponse, error)
	AcceptOffer(accountID, signalID, responseID string) error
	GetActiveFeed(accountID string, businessID string, currentLat, currentLng float64) ([]models.DemandSignal, error)
	GetSignalDetail(signalID string) (*models.DemandSignal, error)
}

type demandService struct {
	demandRepo   DemandRepository   // Sebelumnya mungkin namanya cuma 'repo'
	businessRepo business.BusinessRepository // 👈 TAMBAHKAN INI
}

func NewDemandService(demandRepo DemandRepository, businessRepo business.BusinessRepository) DemandService {
	return &demandService{
		demandRepo:   demandRepo,
		businessRepo: businessRepo,
	}
}

func (s *demandService) CreateSignal(accountID string, input models.CreateSignalInput) (*models.DemandSignal, error) {
	signal := models.DemandSignal{
		SignalID:    utils.GenerateUUID("SIG"),
		AccountID:   accountID,
		Category:    input.Category,
		Description: input.Description,
		MediaUrls:   input.MediaUrls,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		RadiusKm:    input.RadiusKm,
		Status:      "open",
		ExpiresAt:   time.Now().Add(24 * time.Hour), 
	}

	if err := s.demandRepo.CreateSignal(&signal); err != nil {
		return nil, errors.New("gagal membuat demand signal")
	}

	// === TRIGGER NOTIFIKASI REAL-TIME FCM ===
	// Dispatch real-time push notification asynchronously.
	go func() {
    log.Println("🔍 Memulai proses pencarian token...") // Tambahkan ini
    
    tokens, err := s.demandRepo.GetOwnerTokensByCategory(input.Category)
    if err != nil {
        log.Printf("❌ Error saat ambil token: %v\n", err)
        return
    }

    log.Printf("🔍 Jumlah token ditemukan: %d\n", len(tokens)) // Tambahkan ini

    if len(tokens) > 0 {
        title := "🚨 Sinyal Bantuan Masuk!"
        body := "Seseorang membutuhkan bantuan " + input.Category + " di sekitar Anda!"
        data := map[string]string{
            "type":        "signal_post",
            "signal_id":   signal.SignalID,
            "post_id":     signal.SignalID,
            "target_type": "signal",
            "target_id":   signal.SignalID,
        }

        err := config.SendPushNotification(tokens, title, body, data)
        if err != nil {
            log.Printf("❌ Error saat kirim FCM: %v\n", err)
        }
    } else {
        log.Println("⚠️ Tidak ada token ditemukan untuk kategori ini.")
    }
}()
	// ==========================================

	return &signal, nil
}

func (s *demandService) ExploreOpenSignals(category string) ([]models.DemandSignal, error) {
	return s.demandRepo.GetOpenSignalsByCategory(category)
}

func (s *demandService) SubmitResponse(accountID, role, signalID string, input models.CreateResponseInput) (*models.DemandResponse, error) {
	if !s.demandRepo.VerifyBusinessAccess(accountID, role, input.BusinessID) {
		return nil, errors.New("forbidden: anda tidak memiliki akses ke bisnis ini")
	}

	signal, err := s.demandRepo.GetSignalByID(signalID)
	if err != nil || signal.Status != "open" {
		return nil, errors.New("signal ini sudah tidak tersedia atau sudah diambil orang lain")
	}

	response := models.DemandResponse{
		ResponseID: utils.GenerateUUID("RSP"),
		SignalID:   signalID,
		BusinessID: input.BusinessID,
		Message:    input.Message,
		MediaUrls:  input.MediaUrls,
	}

	if err := s.demandRepo.CreateResponse(&response); err != nil {
		return nil, errors.New("gagal mengirimkan penawaran")
	}
	return &response, nil
}

func (s *demandService) AcceptOffer(accountID, signalID, responseID string) error {
	signal, err := s.demandRepo.GetSignalByID(signalID)
	if err != nil {
		return errors.New("signal tidak ditemukan")
	}

	// Ensure only signal creator can accept merchant responses.
	if signal.AccountID != accountID {
		return errors.New("forbidden: anda bukan pemilik signal ini")
	}

	if signal.Status != "open" {
		return errors.New("signal ini sudah tidak dalam status open")
	}

	if err := s.demandRepo.AcceptResponse(signalID, responseID); err != nil {
		return errors.New("gagal memproses penerimaan tawaran")
	}

	return nil
}

// Jangan lupa update interfacenya juga agar menerima parameter businessID string
func (s *demandService) GetActiveFeed(accountID string, businessID string, currentLat, currentLng float64) ([]models.DemandSignal, error) {
	
	var activeBusiness *models.Business
	var err error

	// Resolve target merchant business.
	if businessID != "" {
		// Target specific business if business_id is provided.
		activeBusiness, err = s.businessRepo.GetSpecificBusiness(accountID, businessID)
	} else {
		businesses, errRepo := s.businessRepo.FindBusinessesByOwnerID(accountID)
		if errRepo == nil && len(businesses) > 0 {
			activeBusiness = &businesses[0] // Ambil index 0
		} else {
			err = fmt.Errorf("tidak ada bisnis yang ditemukan")
		}
	}

	if err != nil || activeBusiness == nil {
		return nil, fmt.Errorf("akses bisnis ditolak atau tidak ditemukan: %v", err)
	}

	var categories []string
	for _, cat := range activeBusiness.Categories {
		categories = append(categories, cat.Name)
	}

	if currentLat == 0 && currentLng == 0 {
		if activeBusiness.Latitude != nil && activeBusiness.Longitude != nil {
			currentLat = *activeBusiness.Latitude
			currentLng = *activeBusiness.Longitude
		}
	}

	return s.demandRepo.GetActiveSignalsForOwner(currentLat, currentLng, categories)
}

func (s *demandService) GetSignalDetail(signalID string) (*models.DemandSignal, error) {
	signal, err := s.demandRepo.GetSignalByID(signalID)
	if err != nil {
		return nil, fmt.Errorf("sinyal dengan ID %s tidak ditemukan", signalID)
	}

	return signal, nil
}