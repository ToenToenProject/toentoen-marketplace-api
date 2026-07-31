package meta

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type MetaRepository interface {
	GetAllCategories() ([]models.Category, error)
	GetAllTagGroups() ([]models.TagGroup, error)
}

type metaRepository struct {
	db *gorm.DB
}

func NewMetaRepository(db *gorm.DB) MetaRepository {
	return &metaRepository{db}
}

func (r *metaRepository) GetAllCategories() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Preload("Subcategories").Where("parent_id IS NULL").Find(&categories).Error
	return categories, err
}

func (r *metaRepository) GetAllTagGroups() ([]models.TagGroup, error) {
	var tagGroups []models.TagGroup
	err := r.db.Preload("Tags").Find(&tagGroups).Error
	return tagGroups, err
}
