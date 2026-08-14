package meta

import (
	"toentoen-backend/internal/domain/shared/models"
)

type MetaService interface {
	GetCategories() ([]models.Category, error)
	GetTags(categoryID *int64) ([]models.Tag, error)
	GetStoreTypes() []map[string]string
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

func (s *metaService) GetTags(categoryID *int64) ([]models.Tag, error) {
	return s.metaRepo.GetAllTags(categoryID)
}

func (s *metaService) GetStoreTypes() []map[string]string {
	return []map[string]string{
		{"label": "Rumah", "value": "home"},
		{"label": "Keliling", "value": "mobile"},
		{"label": "Gerobak", "value": "stand"},
		{"label": "Toko", "value": "store"},
		{"label": "Ruko", "value": "shophouse"},
		{"label": "Pasar", "value": "market"},
		{"label": "Alun - alun", "value": "plaza"},
		{"label": "Pusat Belanja", "value": "mall"},
		{"label": "Acara/Musiman", "value": "event"},
	}
}
