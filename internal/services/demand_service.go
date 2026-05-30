package services

import (
	"errors"
	"fmt"
	"log"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
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
	demandRepo   repositories.DemandRepository   // Sebelumnya mungkin namanya cuma 'repo'
	businessRepo repositories.BusinessRepository // 👈 TAMBAHKAN INI
}

func NewDemandService(demandRepo repositories.DemandRepository, businessRepo repositories.BusinessRepository) DemandService {
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
	// Menggunakan goroutine (go func) agar API merespons dengan cepat ke pembeli 
	// tanpa harus menunggu proses pengiriman notifikasi dari server Google selesai.
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
            "type":      "new_demand_signal",
            "signal_id": signal.SignalID,
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

// 2. Owner mencari orderan yang terbuka
func (s *demandService) ExploreOpenSignals(category string) ([]models.DemandSignal, error) {
	return s.demandRepo.GetOpenSignalsByCategory(category)
}

// 3. Owner menawarkan jasanya
func (s *demandService) SubmitResponse(accountID, role, signalID string, input models.CreateResponseInput) (*models.DemandResponse, error) {
	if !s.demandRepo.VerifyBusinessAccess(accountID, role, input.BusinessID) {
		return nil, errors.New("forbidden: anda tidak memiliki akses ke bisnis ini")
	}

	// Cek apakah signalnya masih open
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

// 4. Pembeli menerima tawaran
func (s *demandService) AcceptOffer(accountID, signalID, responseID string) error {
	signal, err := s.demandRepo.GetSignalByID(signalID)
	if err != nil {
		return errors.New("signal tidak ditemukan")
	}

	// Pastikan hanya pembuat signal yang bisa memencet tombol "Terima"
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

	// 1. Tentukan Bisnis Mana yang Mau Dipakai
	if businessID != "" {
		// Jika Frontend mengirim business_id, cari bisnis spesifik tersebut
		activeBusiness, err = s.businessRepo.GetSpecificBusiness(accountID, businessID)
	} else {
		// Jika kosong (mungkin Owner baru login), ambil bisnis pertama sebagai default
		businesses, errRepo := s.businessRepo.FindBusinessesByOwnerID(accountID)
		if errRepo == nil && len(businesses) > 0 {
			activeBusiness = &businesses[0] // Ambil index 0
		} else {
			err = fmt.Errorf("tidak ada bisnis yang ditemukan")
		}
	}

	// Jika masih error atau gagal dapat bisnis, hentikan
	if err != nil || activeBusiness == nil {
		return nil, fmt.Errorf("akses bisnis ditolak atau tidak ditemukan: %v", err)
	}

	// 2. Ekstrak Kategori (Sama seperti sebelumnya)
	var categories []string
	for _, cat := range activeBusiness.Categories {
		categories = append(categories, cat.Name)
	}

	// 3. Fallback Koordinat (Sama seperti sebelumnya)
	// 3. Fallback Koordinat (Sangat Penting!)
	if currentLat == 0 && currentLng == 0 {
		// CEK DULU: Pastikan data di database tidak kosong (bukan NULL/nil)
		if activeBusiness.Latitude != nil && activeBusiness.Longitude != nil {
			// Gunakan tanda * di depan variabel untuk "membongkar" isi pointernya
			currentLat = *activeBusiness.Latitude
			currentLng = *activeBusiness.Longitude
		}
	}

	// 4. Lanjutkan ke perhitungan Haversine
	return s.demandRepo.GetActiveSignalsForOwner(currentLat, currentLng, categories)
}

func (s *demandService) GetSignalDetail(signalID string) (*models.DemandSignal, error) {
	signal, err := s.demandRepo.GetSignalByID(signalID)
	if err != nil {
		// Jika err dari GORM adalah "record not found", kita tangkap di sini
		return nil, fmt.Errorf("sinyal dengan ID %s tidak ditemukan", signalID)
	}

	return signal, nil
}