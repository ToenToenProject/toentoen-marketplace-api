package meta

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
)

const metaCacheTTL = 30 * time.Minute

type MetaService interface {
	GetCategories() ([]models.Category, error)
	GetTags(categoryID *int64, scope string) ([]models.Tag, error)
	GetStoreTypes() []map[string]string
	GetOccupations() ([]models.Occupation, error)
}

type metaService struct {
	metaRepo MetaRepository
}

func NewMetaService(metaRepo MetaRepository) MetaService {
	return &metaService{metaRepo}
}

func (s *metaService) GetCategories() ([]models.Category, error) {
	const cacheKey = "meta:categories:v1"
	if config.RDB != nil {
		if cached, err := config.RDB.Get(config.Ctx, cacheKey).Result(); err == nil && cached != "" {
			var res []models.Category
			if json.Unmarshal([]byte(cached), &res) == nil {
				return res, nil
			}
		}
	}

	res, err := s.metaRepo.GetAllCategories()
	if err != nil {
		return nil, err
	}
	if config.RDB != nil {
		if b, errM := json.Marshal(res); errM == nil {
			config.RDB.Set(config.Ctx, cacheKey, string(b), metaCacheTTL)
		}
	}
	return res, nil
}

func (s *metaService) GetTags(categoryID *int64, scope string) ([]models.Tag, error) {
	catKey := "all"
	if categoryID != nil {
		catKey = fmt.Sprintf("%d", *categoryID)
	}
	scopeKey := strings.ToLower(strings.TrimSpace(scope))
	if scopeKey == "" {
		scopeKey = "all"
	}
	cacheKey := fmt.Sprintf("meta:tags:v1:%s:%s", catKey, scopeKey)

	if config.RDB != nil {
		if cached, err := config.RDB.Get(config.Ctx, cacheKey).Result(); err == nil && cached != "" {
			var res []models.Tag
			if json.Unmarshal([]byte(cached), &res) == nil {
				return res, nil
			}
		}
	}

	res, err := s.metaRepo.GetAllTags(categoryID, scope)
	if err != nil {
		return nil, err
	}
	if config.RDB != nil {
		if b, errM := json.Marshal(res); errM == nil {
			config.RDB.Set(config.Ctx, cacheKey, string(b), metaCacheTTL)
		}
	}
	return res, nil
}

func (s *metaService) GetStoreTypes() []map[string]string {
	return []map[string]string{
		{"label": "Rumah", "label_en": "Home", "value": "home"},
		{"label": "Keliling", "label_en": "Mobile", "value": "mobile"},
		{"label": "Gerobak", "label_en": "Stand", "value": "stand"},
		{"label": "Toko", "label_en": "Store", "value": "store"},
		{"label": "Ruko", "label_en": "Shophouse", "value": "shophouse"},
		{"label": "Pasar", "label_en": "Market", "value": "market"},
		{"label": "Alun - alun", "label_en": "Plaza", "value": "plaza"},
		{"label": "Pusat Belanja", "label_en": "Mall", "value": "mall"},
		{"label": "Acara/Musiman", "label_en": "Event/Seasonal", "value": "event"},
	}
}

func (s *metaService) GetOccupations() ([]models.Occupation, error) {
	return s.metaRepo.GetAllOccupations()
}
