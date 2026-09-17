package meta

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type MetaRepository interface {
	GetAllCategories() ([]models.Category, error)
	GetAllTags(categoryID *int64, scope string) ([]models.Tag, error)
	GetAllOccupations() ([]models.Occupation, error)
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

func (r *metaRepository) GetAllTags(categoryID *int64, scope string) ([]models.Tag, error) {
	var tags []models.Tag
	query := r.db.Preload("Children").Preload("Children.Children").Preload("Children.Children.Children").Where("parent_id IS NULL")

	switch scope {
	case "global":
		query = query.Where("is_global = ?", true)
	case "specific":
		query = query.Where("is_global = ?", false)
	}

	if categoryID != nil {
		categoryIDs := []int64{*categoryID}
		currID := *categoryID
		for {
			var cat models.Category
			if err := r.db.Select("category_id, parent_id").Where("category_id = ?", currID).First(&cat).Error; err != nil || cat.ParentID == nil {
				break
			}
			categoryIDs = append(categoryIDs, *cat.ParentID)
			currID = *cat.ParentID
		}

		if scope == "global" {
			query = r.db.Preload("Children").Preload("Children.Children").Preload("Children.Children.Children").
				Where("parent_id IS NULL").
				Where("is_global = ? OR category_id IN ?", true, categoryIDs)
		} else {
			query = query.Where("category_id IN ?", categoryIDs)
		}
	}

	err := query.Find(&tags).Error
	return tags, err
}

func (r *metaRepository) GetAllOccupations() ([]models.Occupation, error) {
	var occupations []models.Occupation
	err := r.db.Where("is_active = ?", true).Order("label ASC").Find(&occupations).Error
	return occupations, err
}

