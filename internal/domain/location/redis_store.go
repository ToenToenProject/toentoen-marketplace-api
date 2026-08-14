package location

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLocationStore mengelola penyimpanan lokasi merchant di Redis GEO
type RedisLocationStore struct {
	rdb *redis.Client
	ctx context.Context
}

// NewRedisLocationStore membuat instance baru RedisLocationStore
func NewRedisLocationStore(rdb *redis.Client) *RedisLocationStore {
	return &RedisLocationStore{
		rdb: rdb,
		ctx: context.Background(),
	}
}

// ================================================================
// MERCHANT LOCATION (Redis GEO)
// ================================================================

// UpsertMerchantLocation menyimpan atau memperbarui lokasi merchant di Redis GEO
func (s *RedisLocationStore) UpsertMerchantLocation(businessID string, lat, lng float64) error {
	return s.rdb.GeoAdd(s.ctx, KeyMerchantLocations, &redis.GeoLocation{
		Name:      businessID,
		Latitude:  lat,
		Longitude: lng,
	}).Err()
}

// RemoveMerchantLocation menghapus lokasi merchant dari Redis GEO
func (s *RedisLocationStore) RemoveMerchantLocation(businessID string) error {
	return s.rdb.ZRem(s.ctx, KeyMerchantLocations, businessID).Err()
}

// GetNearbyMerchants mengambil daftar merchant dalam radius tertentu dari titik customer
func (s *RedisLocationStore) GetNearbyMerchants(lat, lng, radiusKm float64, limit int) ([]redis.GeoLocation, error) {
	if limit <= 0 {
		limit = 50
	}

	results, err := s.rdb.GeoSearch(s.ctx, KeyMerchantLocations, &redis.GeoSearchQuery{
		Longitude:  lng,
		Latitude:   lat,
		Radius:     radiusKm,
		RadiusUnit: "km",
		Sort:       "ASC", // Terdekat dulu
		Count:      limit,
	}).Result()

	if err != nil {
		return nil, err
	}

	// GeoSearch hanya mengembalikan nama member, kita perlu posisi juga
	var locations []redis.GeoLocation
	for _, name := range results {
		pos, err := s.rdb.GeoPos(s.ctx, KeyMerchantLocations, name).Result()
		if err == nil && len(pos) > 0 && pos[0] != nil {
			dist, _ := s.rdb.GeoDist(s.ctx, KeyMerchantLocations, name, "__query_point__", "km").Result()
			_ = dist // kita hitung manual saja
			locations = append(locations, redis.GeoLocation{
				Name:      name,
				Latitude:  pos[0].Latitude,
				Longitude: pos[0].Longitude,
			})
		}
	}

	return locations, nil
}

// GetNearbyMerchantsWithDist mengambil merchant terdekat beserta jaraknya
func (s *RedisLocationStore) GetNearbyMerchantsWithDist(lat, lng, radiusKm float64, limit int) ([]redis.GeoLocation, error) {
	if limit <= 0 {
		limit = 50
	}

	return s.rdb.GeoSearchLocation(s.ctx, KeyMerchantLocations, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     radiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      limit,
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
}

// GetMerchantLocation mengambil posisi satu merchant tertentu
func (s *RedisLocationStore) GetMerchantLocation(businessID string) (*redis.GeoPos, error) {
	positions, err := s.rdb.GeoPos(s.ctx, KeyMerchantLocations, businessID).Result()
	if err != nil {
		return nil, err
	}
	if len(positions) == 0 || positions[0] == nil {
		return nil, fmt.Errorf("merchant %s tidak ditemukan di live locations", businessID)
	}
	return positions[0], nil
}

// GetTotalActiveMerchants mengembalikan jumlah merchant yang aktif di Redis GEO
func (s *RedisLocationStore) GetTotalActiveMerchants() (int64, error) {
	return s.rdb.ZCard(s.ctx, KeyMerchantLocations).Result()
}

// ================================================================
// MERCHANT METADATA (Redis Hash)
// ================================================================

// SetMerchantMeta menyimpan metadata merchant di Redis Hash
func (s *RedisLocationStore) SetMerchantMeta(businessID, businessName, logoURL, storeType string, isOpen bool) error {
	key := KeyMerchantMetaPrefix + businessID
	pipe := s.rdb.Pipeline()

	pipe.HSet(s.ctx, key, map[string]interface{}{
		"business_name": businessName,
		"logo_url":      logoURL,
		"store_type":    storeType,
		"is_open":       strconv.FormatBool(isOpen),
		"updated_at":    time.Now().Unix(),
	})
	pipe.Expire(s.ctx, key, MerchantMetaTTL)

	_, err := pipe.Exec(s.ctx)
	return err
}

// GetMerchantMeta mengambil metadata merchant dari Redis Hash
func (s *RedisLocationStore) GetMerchantMeta(businessID string) (map[string]string, error) {
	key := KeyMerchantMetaPrefix + businessID
	return s.rdb.HGetAll(s.ctx, key).Result()
}

// DeleteMerchantMeta menghapus metadata merchant dari Redis
func (s *RedisLocationStore) DeleteMerchantMeta(businessID string) error {
	key := KeyMerchantMetaPrefix + businessID
	return s.rdb.Del(s.ctx, key).Err()
}

// ================================================================
// BATCH OPERATIONS (untuk sync dari PostgreSQL)
// ================================================================

// BatchUpsertMerchantLocations menyimpan banyak lokasi merchant sekaligus menggunakan pipeline
func (s *RedisLocationStore) BatchUpsertMerchantLocations(merchants []MerchantGeoData) error {
	if len(merchants) == 0 {
		return nil
	}

	pipe := s.rdb.Pipeline()
	for _, m := range merchants {
		pipe.GeoAdd(s.ctx, KeyMerchantLocations, &redis.GeoLocation{
			Name:      m.BusinessID,
			Latitude:  m.Lat,
			Longitude: m.Lng,
		})
		// Simpan metadata
		key := KeyMerchantMetaPrefix + m.BusinessID
		pipe.HSet(s.ctx, key, map[string]interface{}{
			"business_name": m.BusinessName,
			"logo_url":      m.LogoURL,
			"store_type":    m.StoreType,
			"is_open":       strconv.FormatBool(m.IsOpen),
			"updated_at":    time.Now().Unix(),
		})
		pipe.Expire(s.ctx, key, MerchantMetaTTL)
	}

	_, err := pipe.Exec(s.ctx)
	return err
}

// MerchantGeoData adalah struct perantara untuk batch sync dari DB ke Redis
type MerchantGeoData struct {
	BusinessID   string
	BusinessName string
	LogoURL      string
	StoreType    string
	IsOpen       bool
	Lat          float64
	Lng          float64
}
