package meta

import (
	_ "embed"
	"encoding/json"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed data/indonesia_regions.json
var indonesiaRegionsJSON []byte

type rawRegionCity struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type rawRegionProvince struct {
	Code   string          `json:"code"`
	Name   string          `json:"name"`
	Cities []rawRegionCity `json:"cities"`
}

// SeedIndonesiaRegions idempotently upserts all 38 Indonesian provinces and 514 regencies/cities.
func SeedIndonesiaRegions(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	var raw []rawRegionProvince
	if err := json.Unmarshal(indonesiaRegionsJSON, &raw); err != nil {
		return err
	}

	provinces := make([]models.Province, 0, len(raw))
	cities := make([]models.City, 0, 520)
	for _, p := range raw {
		provinces = append(provinces, models.Province{
			ProvinceCode: p.Code,
			Name:         p.Name,
		})
		for _, c := range p.Cities {
			cities = append(cities, models.City{
				CityCode:     c.Code,
				ProvinceCode: p.Code,
				Name:         c.Name,
			})
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "province_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"name"}),
		}).CreateInBatches(&provinces, 100).Error; err != nil {
			return err
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "city_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"province_code", "name"}),
		}).CreateInBatches(&cities, 100).Error; err != nil {
			return err
		}
		return nil
	})
}
