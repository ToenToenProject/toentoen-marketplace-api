package meta

import (
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type MetaRepository interface {
	GetAllCategories() ([]models.Category, error)
	GetAllTags(categoryID *int64, scope string) ([]models.Tag, error)
	GetAllOccupations() ([]models.Occupation, error)
	GetProvinces(search string) ([]models.Province, error)
	GetCities(provinceCode, search string) ([]models.City, error)
}

type metaRepository struct {
	db *gorm.DB
}

func NewMetaRepository(db *gorm.DB) MetaRepository {
	return &metaRepository{db}
}

func (r *metaRepository) GetAllCategories() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Preload("Children", "is_active = ?", true).
		Preload("Children.Children", "is_active = ?", true).
		Where("parent_id IS NULL AND is_active = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

func (r *metaRepository) GetAllTags(categoryID *int64, scope string) ([]models.Tag, error) {
	var groups []models.TagGroup
	if err := r.db.Where("is_active = ?", true).Order("sort_order ASC, group_id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	// Fallback for unseeded/legacy test databases
	if len(groups) == 0 {
		var tags []models.Tag
		query := r.db.Preload("Children").Preload("Children.Children").Where("parent_id IS NULL AND is_active = ?", true)
		if categoryID != nil {
			query = query.Where("is_global = ? OR category_id = ?", true, *categoryID)
		}
		err := query.Find(&tags).Error
		return tags, err
	}

	var flatTags []models.Tag
	tagQuery := r.db.Model(&models.Tag{}).Where("business_tags.is_active = ?", true)
	if categoryID != nil {
		tagQuery = tagQuery.
			Joins("JOIN category_tag_rules ctr ON ctr.tag_id = business_tags.tag_id AND ctr.is_active = true").
			Where("ctr.category_id = ?", *categoryID)
	}
	if err := tagQuery.Order("business_tags.sort_order ASC, business_tags.tag_id ASC").Find(&flatTags).Error; err != nil {
		return nil, err
	}

	groupSlugByID := make(map[int64]string, len(groups))
	for _, g := range groups {
		groupSlugByID[g.GroupID] = g.Slug
	}

	childrenByParent := make(map[int64][]models.Tag)
	topTagsByGroup := make(map[int64][]models.Tag)
	for i := range flatTags {
		if flatTags[i].GroupID != nil {
			flatTags[i].GroupSlug = groupSlugByID[*flatTags[i].GroupID]
		}
		if flatTags[i].ParentID != nil {
			childrenByParent[*flatTags[i].ParentID] = append(childrenByParent[*flatTags[i].ParentID], flatTags[i])
		}
	}
	for i := range flatTags {
		if flatTags[i].ParentID == nil && flatTags[i].GroupID != nil {
			t := flatTags[i]
			t.Children = childrenByParent[t.TagID]
			topTagsByGroup[*t.GroupID] = append(topTagsByGroup[*t.GroupID], t)
		}
	}

	result := make([]models.Tag, 0, len(groups))
	for _, g := range groups {
		items := topTagsByGroup[g.GroupID]
		if len(items) == 0 {
			continue
		}
		gid := g.GroupID
		result = append(result, models.Tag{
			TagID:     -g.GroupID,
			Slug:      g.Slug,
			Name:      g.Name,
			TagType:   g.TagType,
			GroupID:   &gid,
			GroupSlug: g.Slug,
			IsGlobal:  true,
			SortOrder: g.SortOrder,
			IsActive:  true,
			Children:  items,
		})
	}

	_ = scope
	return result, nil
}

func (r *metaRepository) GetAllOccupations() ([]models.Occupation, error) {
	var occupations []models.Occupation
	err := r.db.Where("is_active = ?", true).Order("label ASC").Find(&occupations).Error
	return occupations, err
}

func (r *metaRepository) GetProvinces(search string) ([]models.Province, error) {
	var provinces []models.Province
	query := r.db.Model(&models.Province{})
	if s := strings.TrimSpace(search); s != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(s)+"%")
	}
	err := query.Order("name ASC").Find(&provinces).Error
	return provinces, err
}

func (r *metaRepository) GetCities(provinceCode, search string) ([]models.City, error) {
	var cities []models.City
	query := r.db.Model(&models.City{})
	if pc := strings.TrimSpace(provinceCode); pc != "" {
		query = query.Where("province_code = ?", pc)
	}
	if s := strings.TrimSpace(search); s != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(s)+"%")
	}
	err := query.Order("name ASC").Find(&cities).Error
	return cities, err
}

