package repositories

import (
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
)

type ProfileRepository interface {
	GetAccountByID(accountID string) (*models.Account, error)
	GetCustomerByAccountID(accountID string) (*models.Customer, error)
	GetOwnerByAccountID(accountID string) (*models.Owner, error)
	GetBusinessAdminByAccountID(accountID string) (*models.BusinessAdmin, error)

	SaveCustomerProfile(account *models.Account, customer *models.Customer) error
	SaveOwnerProfile(account *models.Account, owner *models.Owner) error
	SaveBusinessAdminProfile(account *models.Account, businessAdmin *models.BusinessAdmin) error
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db}
}

func (r *profileRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	return &account, err
}

func (r *profileRepository) GetCustomerByAccountID(accountID string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("account_id = ?", accountID).First(&customer).Error
	return &customer, err
}

func (r *profileRepository) GetOwnerByAccountID(accountID string) (*models.Owner, error) {
	var owner models.Owner
	err := r.db.Where("account_id = ?", accountID).First(&owner).Error
	return &owner, err
}

func (r *profileRepository) GetBusinessAdminByAccountID(accountID string) (*models.BusinessAdmin, error) {
	var businessAdmin models.BusinessAdmin
	err := r.db.Where("account_id = ?", accountID).First(&businessAdmin).Error
	return &businessAdmin, err
}

func (r *profileRepository) SaveCustomerProfile(account *models.Account, customer *models.Customer) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(customer).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *profileRepository) SaveOwnerProfile(account *models.Account, owner *models.Owner) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(owner).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *profileRepository) SaveBusinessAdminProfile(account *models.Account, businessAdmin *models.BusinessAdmin) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(businessAdmin).Error; err != nil {
			return err
		}
		return nil
	})
}
