package report

import (
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"gorm.io/gorm"
)

type Repository interface {
	CreateReport(report *models.BusinessReport) error
	CheckBusinessExists(businessID string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository() Repository {
	return &repository{
		db: database.DB,
	}
}

func (r *repository) CreateReport(report *models.BusinessReport) error {
	// Menggunakan transaksi agar jika insert report gagal, semuanya dibatalkan
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(report).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) CheckBusinessExists(businessID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Business{}).Where("business_id = ?", businessID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
