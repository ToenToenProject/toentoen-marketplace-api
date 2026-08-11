package meta

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type MetaRepository interface {
	GetAllCategories() ([]models.Category, error)
	GetAllTags() ([]models.Tag, error)
}

type metaRepository struct {
	db *gorm.DB
}

func NewMetaRepository(db *gorm.DB) MetaRepository {
	return &metaRepository{db}
}

func (r *metaRepository) GetAllCategories() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Preload("Children").Preload("Children.Children").Preload("Children.Children.Children").Where("parent_id IS NULL").Find(&categories).Error
	return categories, err
}

func (r *metaRepository) GetAllTags() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Preload("Children").Preload("Children.Children").Preload("Children.Children.Children").Where("parent_id IS NULL").Find(&tags).Error
	return tags, err
}
