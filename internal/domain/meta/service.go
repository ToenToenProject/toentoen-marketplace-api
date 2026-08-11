package meta

import (
	"toentoen-backend/internal/domain/shared/models"
)

type MetaService interface {
	GetCategories() ([]models.Category, error)
	GetTags(categoryID *int64) ([]models.Tag, error)
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
