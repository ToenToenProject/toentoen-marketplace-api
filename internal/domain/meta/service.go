package meta

import (
	"toentoen-backend/internal/domain/shared/models"
)

type MetaService interface {
	GetCategories() ([]models.Category, error)
	GetTags() ([]models.TagGroup, error)
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

func (s *metaService) GetTags() ([]models.TagGroup, error) {
	return s.metaRepo.GetAllTagGroups()
}
