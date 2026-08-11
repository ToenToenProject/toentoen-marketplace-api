package meta

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type MetaRepository interface {
	GetAllCategories() ([]models.Category, error)
}

type metaRepository struct {
	db *gorm.DB
}

func NewMetaRepository(db *gorm.DB) MetaRepository {
	return &metaRepository{db}
}

func (r *metaRepository) GetAllCategories() ([]models.Category, error) {
	var categories []models.Category
	// We don't filter by parent_id since we dropped the recursive Category-to-Category tree for this feature
	err := r.db.Preload("Subcategories").Find(&categories).Error
	return categories, err
}
