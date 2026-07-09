package business_profile

import (
	"errors"

	"gorm.io/gorm"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
)

type Repository interface {
	GetBusinessDetail(businessID string) (models.Business, error)
	CheckBookmark(accountID, businessID string) (bool, error)
	ToggleBookmark(accountID, businessID string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository() Repository {
	return &repository{
		db: database.DB,
	}
}

func (r *repository) GetBusinessDetail(businessID string) (models.Business, error) {
	var business models.Business
	err := r.db.
		Preload("Categories").
		Preload("Hours").
		Where("business_id = ?", businessID).
		First(&business).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return business, errors.New("business not found")
		}
		return business, err
	}

	return business, nil
}

func (r *repository) CheckBookmark(accountID, businessID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.BusinessBookmark{}).
		Where("account_id = ? AND business_id = ?", accountID, businessID).
		Count(&count).Error
	
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) ToggleBookmark(accountID, businessID string) (bool, error) {
	var existing models.BusinessBookmark
	err := r.db.Where("account_id = ? AND business_id = ?", accountID, businessID).First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Buat baru
			newBookmark := models.BusinessBookmark{
				AccountID:  accountID,
				BusinessID: businessID,
			}
			if errCreate := r.db.Create(&newBookmark).Error; errCreate != nil {
				return false, errCreate
			}
			return true, nil // Sekarang di-bookmark
		}
		return false, err
	}

	// Hapus (Unbookmark)
	if errDelete := r.db.Delete(&existing).Error; errDelete != nil {
		return true, errDelete // gagal hapus, masih bookmarked
	}

	return false, nil // Sudah tidak di-bookmark
}
