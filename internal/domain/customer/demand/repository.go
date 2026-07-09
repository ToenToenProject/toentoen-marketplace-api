package demand

import (
	"fmt"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type DemandRepository interface {
	VerifyBusinessAccess(accountID, role, businessID string) bool
	CreateSignal(signal *models.DemandSignal) error
	GetOpenSignalsByCategory(category string) ([]models.DemandSignal, error)
	GetSignalByID(signalID string) (*models.DemandSignal, error)
	
	CreateResponse(response *models.DemandResponse) error
	AcceptResponse(signalID, responseID string) error
	GetOwnerTokensByCategory(category string) ([]string, error) // Tambahkan baris ini
	GetActiveSignalsForOwner(ownerLat, ownerLng float64, categories []string) ([]models.DemandSignal, error)
	
}

type demandRepository struct {
	db *gorm.DB
}

func NewDemandRepository(db *gorm.DB) DemandRepository {
	return &demandRepository{db}
}

func (r *demandRepository) VerifyBusinessAccess(accountID, role, businessID string) bool {
	var count int64
	if role == "owner" {
		r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count)
	} else if role == "business_admin" {
		r.db.Model(&models.BusinessAdminAssignment{}).Where("business_id = ? AND admin_id = ? AND is_active = ?", businessID, accountID, true).Count(&count)
	}
	return count > 0
}

func (r *demandRepository) CreateSignal(signal *models.DemandSignal) error {
	return r.db.Create(signal).Error
}

func (r *demandRepository) GetOpenSignalsByCategory(category string) ([]models.DemandSignal, error) {
	var signals []models.DemandSignal
	err := r.db.Preload("Account").
		Where("category = ? AND status = ? AND expires_at > ?", category, "open", time.Now()).
		Order("created_at DESC").
		Find(&signals).Error
	return signals, err
}

func (r *demandRepository) GetSignalByID(signalID string) (*models.DemandSignal, error) {
	var signal models.DemandSignal

	// Tip: Jika di struct DemandSignal Anda punya relasi ke data Pembeli (misal: Account),
	// Anda bisa menambahkan .Preload("Account") sebelum .Where
	err := r.db.Where("signal_id = ?", signalID).First(&signal).Error
	if err != nil {
		return nil, err
	}

	return &signal, nil
}

func (r *demandRepository) CreateResponse(response *models.DemandResponse) error {
	return r.db.Create(response).Error
}

// Transaction: Menerima tawaran sekaligus menutup signal
func (r *demandRepository) AcceptResponse(signalID, responseID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update Response menjadi accepted
		if err := tx.Model(&models.DemandResponse{}).Where("response_id = ?", responseID).Update("is_accepted", true).Error; err != nil {
			return err
		}

		// 2. Update Signal menjadi processing & terpenuhi
		now := time.Now()
		updates := map[string]interface{}{
			"status":       "processing",
			"is_fulfilled": true,
			"fulfilled_at": now,
		}
		if err := tx.Model(&models.DemandSignal{}).Where("signal_id = ?", signalID).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *demandRepository) GetOwnerTokensByCategory(categoryName string) ([]string, error) {
	var tokens []string
	
	// Kita melakukan 3 kali JOIN sekarang:
	// 1. Join ke device_tokens untuk mendapatkan alamat FCM HP Owner
	// 2. Join ke business_categories (Tabel Jembatan / Pivot Table)
	// 3. Join ke categories untuk mencocokkan namanya
	err := r.db.Table("businesses").
		Select("DISTINCT device_tokens.fcm_token").
		Joins("JOIN device_tokens ON device_tokens.account_id = businesses.account_id").
		Joins("JOIN business_categories ON business_categories.business_id = businesses.business_id").
		Joins("JOIN categories ON categories.category_id = business_categories.category_id").
		Where("categories.name = ?", categoryName).
		Pluck("device_tokens.fcm_token", &tokens).Error

	return tokens, err
}

func (r *demandRepository) GetActiveSignalsForOwner(ownerLat, ownerLng float64, categories []string) ([]models.DemandSignal, error) {
	var signals []models.DemandSignal

	// Jika owner belum punya kategori, langsung kembalikan array kosong agar tidak error di SQL
	if len(categories) == 0 {
		return signals, nil
	}

	// Rumus Haversine dalam raw SQL. 
	// Menghitung jarak (dalam KM) antara koordinat Sinyal (latitude/longitude di tabel) 
	// dengan koordinat Owner (yang dikirim sebagai parameter ?).
	distanceQuery := fmt.Sprintf(
		"(6371 * acos(cos(radians(%f)) * cos(radians(latitude)) * cos(radians(longitude) - radians(%f)) + sin(radians(%f)) * sin(radians(latitude))))",
		ownerLat, ownerLng, ownerLat,
	)

	err := r.db.
		Where("status = ?", "open").
		Where("expires_at > ?", time.Now()).
		Where("category IN ?", categories).
		// Karena distanceQuery sudah berisi angka asli (bukan ? lagi), kita tidak perlu melempar variabel di belakangnya
		Where(distanceQuery + " <= radius_km"). 
		Order(distanceQuery + " ASC"). // Order by sekarang berisi string rumus yang valid tanpa tanda tanya
		Find(&signals).Error

	return signals, err
}