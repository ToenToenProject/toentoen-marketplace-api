package repositories

import (
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
)

type ProfileRepository interface {
	GetAccountByID(accountID int64) (*models.Account, error)
	GetUserByAccountID(accountID int64) (*models.User, error)
	GetOwnerByAccountID(accountID int64) (*models.Owner, error)
	GetBusinessAdminByAccountID(accountID int64) (*models.BusinessAdmin, error)
	SaveUserProfile(account *models.Account, user *models.User) error
	SaveOwnerProfile(account *models.Account, owner *models.Owner) error
	SaveBusinessAdminProfile(account *models.Account, businessAdmin *models.BusinessAdmin) error
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository{
	return &profileRepository{db}
}

func (r *profileRepository) GetAccountByID(accountID int64) (*models.Account, error) {
	var account models.Account
	err := r.db.First(&account, accountID).Error
	return &account, err
}

func (r *profileRepository) GetUserByAccountID(accountID int64) (*models.User, error) {
	var user models.User
	err := r.db.Where("account_id = ?", accountID).First(&user).Error
	return &user, err
}

func (r *profileRepository) GetOwnerByAccountID(accountID int64) (*models.Owner, error) {
	var owner models.Owner
	err := r.db.Where("account_id = ?", accountID).First(&owner).Error
	return &owner, err
}

func (r *profileRepository) GetBusinessAdminByAccountID(accountID int64) (*models.BusinessAdmin, error) {
	var businessAdmin models.BusinessAdmin
	err := r.db.Where("account_id = ?", accountID).First(&businessAdmin).Error
	return &businessAdmin, err
}

func (r *profileRepository) SaveUserProfile(account *models.Account, user *models.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil{
			return err
		}

		if err := tx.Save(user).Error; err != nil {
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