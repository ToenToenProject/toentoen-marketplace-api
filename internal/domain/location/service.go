package location

import (
	"log"
	"math"
	"time"

	"toentoen-backend/internal/domain/realtime"

	"gorm.io/gorm"
)

// LocationService mengelola logika bisnis fitur Live Location
type LocationService struct {
	store *RedisLocationStore
	hub   *realtime.Hub
	db    *gorm.DB
}

// NewLocationService membuat instance baru LocationService
func NewLocationService(store *RedisLocationStore, hub *realtime.Hub, db *gorm.DB) *LocationService {
	return &LocationService{
		store: store,
		hub:   hub,
		db:    db,
	}
}

// ================================================================
// CORE: NEARBY QUERY
// ================================================================

// GetNearbyMerchants mengambil daftar merchant terdekat dari posisi customer
func (s *LocationService) GetNearbyMerchants(lat, lng, radiusKm float64, limit int) ([]NearbyMerchant, error) {
	if radiusKm <= 0 {
		radiusKm = 5 // Default 5 km
	}
	if limit <= 0 {
		limit = 50
	}

	results, err := s.store.GetNearbyMerchantsWithDist(lat, lng, radiusKm, limit)
	if err != nil {
		return nil, err
	}

	var merchants []NearbyMerchant
	for _, r := range results {
		meta, err := s.store.GetMerchantMeta(r.Name)
		if err != nil || len(meta) == 0 {
			// Metadata belum ada di Redis, skip (akan di-sync oleh worker)
			continue
		}

		merchant := NearbyMerchant{
			BusinessID:   r.Name,
			BusinessName: meta["business_name"],
			LogoURL:      meta["logo_url"],
			Lat:          r.Latitude,
			Lng:          r.Longitude,
			DistanceKm:   math.Round(r.Dist*100) / 100, // Bulatkan ke 2 desimal
			IsOpen:       meta["is_open"] == "true",
			StoreType:    meta["store_type"],
		}

		merchants = append(merchants, merchant)
	}

	return merchants, nil
}

// GetMerchantPosition mengambil posisi terkini satu merchant tertentu
func (s *LocationService) GetMerchantPosition(businessID string) (*NearbyMerchant, error) {
	pos, err := s.store.GetMerchantLocation(businessID)
	if err != nil {
		return nil, err
	}

	meta, _ := s.store.GetMerchantMeta(businessID)

	merchant := &NearbyMerchant{
		BusinessID:   businessID,
		BusinessName: meta["business_name"],
		LogoURL:      meta["logo_url"],
		Lat:          pos.Latitude,
		Lng:          pos.Longitude,
		IsOpen:       meta["is_open"] == "true",
		StoreType:    meta["store_type"],
	}

	return merchant, nil
}

// ================================================================
// CUSTOMER LOCATION PROCESSING (dari WebSocket)
// ================================================================

// ProcessCustomerLocation menerima update lokasi dari customer,
// menjalankan filter, dan mengembalikan daftar merchant terdekat terbaru
func (s *LocationService) ProcessCustomerLocation(state *realtime.LocationState, update LocationUpdate) ([]NearbyMerchant, bool) {
	now := time.Now()

	// 1. Rate limiting: maksimal 1 update per detik
	if now.Sub(state.LastUpdateAt) < 1*time.Second {
		return nil, false
	}
	state.LastUpdateAt = now
	state.UpdateCount++

	// 2. GPS accuracy filter: tolak update dengan akurasi > 100 meter
	if update.Accuracy > 100 {
		return nil, false
	}

	// 3. GPS jitter filter
	if state.LastLat != 0 && state.LastLng != 0 {
		if isGPSJitter(state.LastLat, state.LastLng, update.Lat, update.Lng, update.Speed) {
			return nil, false
		}
	}

	// 4. Adaptive throttle: kurangi frekuensi jika customer diam
	if !shouldSendUpdate(state, update.Lat, update.Lng) {
		return nil, false
	}

	// 5. Update state
	state.LastLat = update.Lat
	state.LastLng = update.Lng
	state.LastSentAt = now

	// 6. Query nearby merchants
	merchants, err := s.GetNearbyMerchants(update.Lat, update.Lng, 5, 50)
	if err != nil {
		log.Printf("⚠️ Error querying nearby merchants: %v", err)
		return nil, false
	}

	return merchants, true
}

// ================================================================
// SYNC: PostgreSQL → Redis (Startup & Periodic)
// ================================================================

// SyncMerchantLocationsFromDB membaca semua merchant yang memiliki koordinat
// dari PostgreSQL dan menyimpannya ke Redis GEO
func (s *LocationService) SyncMerchantLocationsFromDB() error {
	type merchantRow struct {
		BusinessID   string  `gorm:"column:business_id"`
		BusinessName string  `gorm:"column:business_name"`
		LogoURL      *string `gorm:"column:logo_url"`
		StoreType    *string `gorm:"column:store_type"`
		Latitude     float64 `gorm:"column:latitude"`
		Longitude    float64 `gorm:"column:longitude"`
		IsActive     bool    `gorm:"column:is_active"`
		IsClosed     bool    `gorm:"column:is_manually_closed"`
	}

	var rows []merchantRow
	err := s.db.Table("businesses").
		Select("business_id, business_name, logo_url, store_type, latitude, longitude, is_active, is_manually_closed").
		Where("latitude IS NOT NULL AND longitude IS NOT NULL AND latitude != 0 AND longitude != 0").
		Where("is_active = ?", true).
		Find(&rows).Error

	if err != nil {
		return err
	}

	var batch []MerchantGeoData
	for _, r := range rows {
		logoURL := ""
		if r.LogoURL != nil {
			logoURL = *r.LogoURL
		}
		storeType := "regular"
		if r.StoreType != nil {
			storeType = *r.StoreType
		}

		batch = append(batch, MerchantGeoData{
			BusinessID:   r.BusinessID,
			BusinessName: r.BusinessName,
			LogoURL:      logoURL,
			StoreType:    storeType,
			IsOpen:       !r.IsClosed,
			Lat:          r.Latitude,
			Lng:          r.Longitude,
		})
	}

	if err := s.store.BatchUpsertMerchantLocations(batch); err != nil {
		return err
	}

	log.Printf("📍 Synced %d merchant locations dari PostgreSQL ke Redis", len(batch))
	return nil
}

// StartPeriodicSync menjalankan sinkronisasi periodik setiap interval tertentu
func (s *LocationService) StartPeriodicSync(interval time.Duration) {
	go func() {
		// Sync pertama kali saat startup
		if err := s.SyncMerchantLocationsFromDB(); err != nil {
			log.Printf("⚠️ Gagal sync awal lokasi merchant: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.SyncMerchantLocationsFromDB(); err != nil {
				log.Printf("⚠️ Gagal periodic sync lokasi merchant: %v", err)
			}
		}
	}()
}

// ================================================================
// THROTTLE LOGIC
// ================================================================

// shouldSendUpdate menentukan apakah update lokasi ini layak diteruskan (adaptive throttle)
func shouldSendUpdate(t *realtime.LocationState, lat, lng float64) bool {
	now := time.Now()

	// Jika belum pernah mengirim, langsung kirim
	if t.LastSentAt.IsZero() {
		return true
	}

	dist := haversine(t.LastLat, t.LastLng, lat, lng)

	if dist < 5 { // < 5 meter = dianggap diam
		if !t.IsStationary {
			t.IsStationary = true
			t.StationarySince = now
		}
		// Saat diam: kirim setiap 15 detik saja (hemat resource)
		return now.Sub(t.LastSentAt) >= 15*time.Second
	}

	// Bergerak: kirim setiap 3 detik
	t.IsStationary = false
	return now.Sub(t.LastSentAt) >= 3*time.Second
}

// ================================================================
// GPS UTILITY FUNCTIONS
// ================================================================

// isGPSJitter mendeteksi apakah perpindahan koordinat tidak masuk akal
// berdasarkan kecepatan yang dilaporkan device
func isGPSJitter(prevLat, prevLng, newLat, newLng, speedKmh float64) bool {
	dist := haversine(prevLat, prevLng, newLat, newLng) // dalam meter

	// Kecepatan default jika tidak tersedia (asumsi jalan kaki)
	if speedKmh <= 0 {
		speedKmh = 5
	}

	// Jarak maksimum yang masuk akal: speed (km/h) × 5 detik (interval + margin)
	maxPossibleDist := speedKmh * 5 / 3.6 // konversi ke meter

	// Jika jarak > 3× jarak logis DAN > 100 meter, anggap jitter
	return dist > math.Max(maxPossibleDist*3, 100)
}

// haversine menghitung jarak antara dua titik koordinat dalam meter
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000 // Radius bumi dalam meter

	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusM * c
}

// degreesToRadians mengkonversi derajat ke radian
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
