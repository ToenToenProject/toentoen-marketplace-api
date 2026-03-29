package repositories

import (
	"toentoen-backend/internal/models"
	"gorm.io/gorm"
)

type BusinessRepository interface {
	CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour, websites []models.BusinessWebsite, images []models.BusinessImage) error
	
	FindBusinessesByOwnerID(ownerID string) ([]models.Business, error)
	FindBusinessByIDAndOwner(businessID, ownerID string) (*models.Business, error)
	CheckHandleExists(handle, excludeBusinessID string) bool
	UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64, newWebsites *[]models.BusinessWebsite, newImages *[]models.BusinessImage) error
	DeleteBusiness(business *models.Business) error
	FindBusinessByIDAndOwnerWithPreload(businessID, ownerID string) (*models.Business, error)
	CheckBusinessNameExists(ownerID, name string) bool
}

type businessRepository struct {
	db *gorm.DB
}

func NewBusinessRepository(db *gorm.DB) BusinessRepository {
	return &businessRepository{db}
}

func (r *businessRepository) CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour, websites []models.BusinessWebsite, images []models.BusinessImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(business).Error; err != nil {
			return err
		}

		if len(hours) > 0 {
			for i := range hours {
				hours[i].BusinessID = business.BusinessID 
			}
			if err := tx.Create(&hours).Error; err != nil {
				return err
			}
		}

		if len(websites) > 0 {
			for i := range websites {
				websites[i].BusinessID = business.BusinessID 
			}
			if err := tx.Create(&websites).Error; err != nil {
				return err
			}
		}

		if len(images) > 0 {
			for i := range images {
				images[i].BusinessID = business.BusinessID 
			}
			if err := tx.Create(&images).Error; err != nil {
				return err
			}
		}

		if len(categoryIDs) > 0 {
			var businessCategories []models.BusinessCategory
			for _, catID := range categoryIDs {
				businessCategories = append(businessCategories, models.BusinessCategory{
					BusinessID: business.BusinessID,
					CategoryID: catID,
				})
			}
			if err := tx.Create(&businessCategories).Error; err != nil {
				return err
			}
		}

		if len(tagIDs) > 0 {
			var businessTags []models.BusinessTag
			for _, tagID := range tagIDs {
				businessTags = append(businessTags, models.BusinessTag{
					BusinessID: business.BusinessID,
					TagID:      tagID,
				})
			}
			if err := tx.Create(&businessTags).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *businessRepository) FindBusinessesByOwnerID(ownerID string) ([]models.Business, error) {
	var businesses []models.Business
	err := r.db.
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		Preload("Websites").
		Preload("Images").
		Where("account_id = ?", ownerID).
		Find(&businesses).Error
	return businesses, err
}

func (r *businessRepository) FindBusinessByIDAndOwner(businessID, ownerID string) (*models.Business, error) {
	var business models.Business
	err := r.db.Where("business_id = ? AND account_id = ?", businessID, ownerID).First(&business).Error
	return &business, err
}

func (r *businessRepository) FindBusinessByIDAndOwnerWithPreload(businessID, ownerID string) (*models.Business, error) {
	var business models.Business
	err := r.db.
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		Preload("Websites").
		Preload("Images").
		Where("business_id = ? AND account_id = ?", businessID, ownerID).
		First(&business).Error

	return &business, err
}

func (r *businessRepository) CheckHandleExists(handle ,excludeBusinessID string) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("handle = ? AND business_id != ?", handle, excludeBusinessID).Count(&count)
	return count > 0 
}

func (r *businessRepository) UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64, newWebsites *[]models.BusinessWebsite, newImages *[]models.BusinessImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(business).Error; err != nil {
			return err
		}

		if newHours != nil {
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessHour{}).Error; err != nil {
				return err
			}
			if len(*newHours) > 0 {
				hours := *newHours
				for i := range hours {
					hours[i].BusinessID = business.BusinessID
				}
				if err := tx.Create(&hours).Error; err != nil {
					return err
				}
			}
		}

		if newWebsites != nil {
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessWebsite{}).Error; err != nil {
				return err
			}
			if len(*newWebsites) > 0 {
				websites := *newWebsites
				for i := range websites {
					websites[i].BusinessID = business.BusinessID
				}
				if err := tx.Create(&websites).Error; err != nil {
					return err
				}
			}
		}

		if newImages != nil {
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessImage{}).Error; err != nil {
				return err
			}
			if len(*newImages) > 0 {
				images := *newImages
				for i := range images {
					images[i].BusinessID = business.BusinessID
				}
				if err := tx.Create(&images).Error; err != nil {
					return err
				}
			}
		}

		if newCategoryIDs != nil {
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessCategory{}).Error; err != nil {
				return err
			}
			if len(*newCategoryIDs) > 0 {
				var businessCategories []models.BusinessCategory
				for _, catID := range *newCategoryIDs {
					businessCategories = append(businessCategories, models.BusinessCategory{
						BusinessID: business.BusinessID,
						CategoryID: catID,
					})
				}
				if err := tx.Create(&businessCategories).Error; err != nil {
					return err
				}
			}
		}

		if newTagIDs != nil {
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessTag{}).Error; err != nil {
				return err
			}
			if len(*newTagIDs) > 0 {
				var businessTags []models.BusinessTag
				for _, tagID := range *newTagIDs {
					businessTags = append(businessTags, models.BusinessTag{
						BusinessID: business.BusinessID,
						TagID:      tagID,
					})
				}
				if err := tx.Create(&businessTags).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (r *businessRepository) DeleteBusiness(business *models.Business) error {
	return r.db.Delete(business).Error
}

func (r *businessRepository) CheckBusinessNameExists(ownerID, name string) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("account_id = ? AND name = ?", ownerID, name).Count(&count)
	return count > 0 
}