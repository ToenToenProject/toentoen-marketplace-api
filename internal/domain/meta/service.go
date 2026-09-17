package meta

import (
	"toentoen-backend/internal/domain/shared/models"
)

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
	return s.metaRepo.GetAllCategories()
}

func (s *metaService) GetTags(categoryID *int64, scope string) ([]models.Tag, error) {
	return s.metaRepo.GetAllTags(categoryID, scope)
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
