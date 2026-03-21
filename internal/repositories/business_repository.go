package repositories

import (
	"toentoen-backend/internal/models"
	"gorm.io/gorm"
)

// Pastikan interface BusinessRepository di-update menjadi:
type BusinessRepository interface {
	CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour) error
	
	// Tambahan baru:
	FindBusinessesByOwnerID(ownerID int64) ([]models.Business, error)
	FindBusinessByIDAndOwner(businessID, ownerID int64) (*models.Business, error)
	CheckHandleExists(handle string, excludeBusinessID int64) bool
	UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64) error
	DeleteBusiness(business *models.Business) error
	FindBusinessByIDAndOwnerWithPreload(businessID, ownerID int64) (*models.Business, error)
}

type businessRepository struct {
	db *gorm.DB
}

func NewBusinessRepository(db *gorm.DB) BusinessRepository {
	return &businessRepository{db}
}

func (r *businessRepository) CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Insert Data Utama (Tabel businesses)
		if err := tx.Create(business).Error; err != nil {
			return err
		}

		// 2. Insert Jam Operasional (Bulk Insert)
		if len(hours) > 0 {
			for i := range hours {
				hours[i].BusinessID = business.BusinessID // Pasang ID bisnis yang baru jadi
			}
			if err := tx.Create(&hours).Error; err != nil {
				return err
			}
		}

		// 3. Insert Kategori Pivot (Tabel business_categories)
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

		// 4. Insert Tag Pivot (Tabel business_tags)
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

		return nil // Commit transaksi jika semua lancar
	})
}

// === IMPLEMENTASI FUNGSI BARU ===

func (r *businessRepository) FindBusinessesByOwnerID(ownerID int64) ([]models.Business, error) {
	var businesses []models.Business
	// Preload menarik data relasi (Jam, Kategori, Tag) agar JSON response lengkap
	err := r.db.
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		Where("account_id = ?", ownerID).
		Find(&businesses).Error
	return businesses, err
}

func (r *businessRepository) FindBusinessByIDAndOwner(businessID, ownerID int64) (*models.Business, error) {
	var business models.Business
	err := r.db.Where("business_id = ? AND account_id = ?", businessID, ownerID).First(&business).Error
	return &business, err
}

func (r *businessRepository) CheckHandleExists(handle string, excludeBusinessID int64) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("handle = ? AND business_id != ?", handle, excludeBusinessID).Count(&count)
	return count > 0 // Return true jika handle sudah dipakai toko lain
}

// Ubah implementasi UpdateBusiness di file repository Anda menjadi seperti ini:

func (r *businessRepository) UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update tabel utama (businesses)
		if err := tx.Save(business).Error; err != nil {
			return err
		}

		// 2. Sync Jam Operasional (Jika dikirim dari Frontend)
		if newHours != nil {
			// Hapus semua jam lama
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessHour{}).Error; err != nil {
				return err
			}
			// Insert jam baru (jika array tidak kosong)
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

		// 3. Sync Kategori (Jika dikirim dari Frontend)
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

		// 4. Sync Tags (Jika dikirim dari Frontend)
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

func (r *businessRepository) FindBusinessByIDAndOwnerWithPreload(businessID, ownerID int64) (*models.Business, error) {
	var business models.Business
	err := r.db.
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		Where("business_id = ? AND account_id = ?", businessID, ownerID).
		First(&business).Error

	return &business, err
}