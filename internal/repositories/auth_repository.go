package repositories

import (
	"errors"
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
)

type AuthRepository interface {
	CreateAccountWithProfile(account *models.Account, role string, ownerID ...int64) error
	FindByIdentifier(identifier string) (*models.Account, error)
	FindByID(id int64) (*models.Account, error)
	Update(account *models.Account) error
	LoadAssociations(account *models.Account) error
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db}
}

func (r *authRepository) CreateAccountWithProfile(account *models.Account, role string, ownerID ...int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return err
		}

		switch role {
		case "user":
			if err := tx.Create(&models.User{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "owner":
			if err := tx.Create(&models.Owner{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "business_admin":
			var bossID int64
			if len(ownerID) > 0 {
				bossID = ownerID[0]
			} else {
				return errors.New("business_admin creation requires owner_id")
			}

			if err := tx.Create(&models.BusinessAdmin{
				AccountID: account.AccountID,
				OwnerID:   bossID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *authRepository) FindByIdentifier(identifier string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("email = ?", identifier).Or("phone_number = ?", identifier).First(&account).Error
	return &account, err
}

func (r *authRepository) FindByID(id int64) (*models.Account, error) {
	var account models.Account
	err := r.db.First(&account, id).Error
	return &account, err
}

func (r *authRepository) Update(account *models.Account) error {
	return r.db.Save(account).Error
}

func (r *authRepository) LoadAssociations(account *models.Account) error {
	switch account.Role {
	case "user":
		r.db.Model(account).Association("User").Find(&account.User)
	case "owner":
		r.db.Model(account).Association("Owner").Find(&account.Owner)
	case "business_admin":
		r.db.Model(account).Association("BusinessAdmin").Find(&account.BusinessAdmin)
	case "platform_admin":
		r.db.Model(account).Association("PlatformAdmin").Find(&account.PlatformAdmin)

	}
	return nil
}
