package transfer

import (
	"errors"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type BusinessTransferRepository interface {
	FindAccountByIdentifier(identifier string) (*models.Account, error)
	FindAccountByID(accountID string) (*models.Account, error)
	FindBusinessByID(businessID string) (*models.Business, error)
	GetActiveTransferByBusiness(businessID string) (*models.BusinessTransferRequest, error)
	CreateTransferRequest(req *models.BusinessTransferRequest) error
	GetTransferByID(transferID string) (*models.BusinessTransferRequest, error)
	GetIncomingTransfers(targetAccountID string) ([]models.BusinessTransferRequest, error)
	CancelTransfer(transferID string) error
	RejectTransfer(transferID string) error
	AcceptTransfer(transferID string, businessID string, newOwnerID string) error
}

type businessTransferRepository struct {
	db *gorm.DB
}

func NewBusinessTransferRepository(db *gorm.DB) BusinessTransferRepository {
	return &businessTransferRepository{db: db}
}

func (r *businessTransferRepository) FindAccountByIdentifier(identifier string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("email = ? OR phone_number = ? OR username = ?", identifier, identifier, identifier).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *businessTransferRepository) FindAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *businessTransferRepository) FindBusinessByID(businessID string) (*models.Business, error) {
	var business models.Business
	err := r.db.Where("business_id = ?", businessID).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *businessTransferRepository) GetActiveTransferByBusiness(businessID string) (*models.BusinessTransferRequest, error) {
	var req models.BusinessTransferRequest
	err := r.db.
		Where("business_id = ? AND status = 'pending_acceptance' AND expires_at > ?", businessID, time.Now()).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *businessTransferRepository) CreateTransferRequest(req *models.BusinessTransferRequest) error {
	return r.db.Create(req).Error
}

func (r *businessTransferRepository) GetTransferByID(transferID string) (*models.BusinessTransferRequest, error) {
	var req models.BusinessTransferRequest
	err := r.db.
		Preload("Business").
		Preload("CurrentOwner").
		Preload("TargetAccount").
		Where("transfer_id = ?", transferID).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *businessTransferRepository) GetIncomingTransfers(targetAccountID string) ([]models.BusinessTransferRequest, error) {
	var reqs []models.BusinessTransferRequest
	err := r.db.
		Preload("Business").
		Preload("CurrentOwner").
		Where("target_account_id = ? AND status = 'pending_acceptance' AND expires_at > ?", targetAccountID, time.Now()).
		Order("created_at DESC").
		Find(&reqs).Error
	if err != nil {
		return nil, err
	}
	return reqs, nil
}

func (r *businessTransferRepository) CancelTransfer(transferID string) error {
	now := time.Now()
	return r.db.Model(&models.BusinessTransferRequest{}).
		Where("transfer_id = ?", transferID).
		Updates(map[string]interface{}{
			"status":       "cancelled",
			"cancelled_at": &now,
			"updated_at":   now,
		}).Error
}

func (r *businessTransferRepository) RejectTransfer(transferID string) error {
	now := time.Now()
	return r.db.Model(&models.BusinessTransferRequest{}).
		Where("transfer_id = ?", transferID).
		Updates(map[string]interface{}{
			"status":      "rejected",
			"rejected_at": &now,
			"updated_at":  now,
		}).Error
}

func (r *businessTransferRepository) AcceptTransfer(transferID string, businessID string, newOwnerID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 1. Update status transfer request
		if err := tx.Model(&models.BusinessTransferRequest{}).
			Where("transfer_id = ?", transferID).
			Updates(map[string]interface{}{
				"status":      "accepted",
				"accepted_at": &now,
				"updated_at":  now,
			}).Error; err != nil {
			return err
		}

		// 2. Pindahkan kepemilikan bisnis ke owner baru
		res := tx.Model(&models.Business{}).
			Where("business_id = ?", businessID).
			Update("account_id", newOwnerID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("Bisnis tidak ditemukan atau gagal diperbarui")
		}

		// 3. Pastikan record Owner di tabel owners tersedia untuk owner baru
		var ownerCount int64
		if err := tx.Model(&models.Owner{}).Where("account_id = ?", newOwnerID).Count(&ownerCount).Error; err == nil && ownerCount == 0 {
			newOwnerRecord := models.Owner{
				AccountID: newOwnerID,
				CreatedAt: now,
			}
			_ = tx.Create(&newOwnerRecord).Error
		}

		return nil
	})
}
