package repositories

import (
	"errors"
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
)

type AuthRepository interface {
	CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error
	FindByIdentifier(identifier string) (*models.Account, error)
	FindByID(id string) (*models.Account, error) 
	Update(account *models.Account) error
	LoadAssociations(account *models.Account) error
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db}
}

func (r *authRepository) CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error {
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
			var bossID string
			if len(ownerID) > 0 {
				bossID = ownerID[0]
			} else {
				return errors.New("pembuatan akun business_admin membutuhkan owner_id")
			}

			var existingOwner models.Owner
			if err := tx.Where("account_id = ?", bossID).First(&existingOwner).Error; err != nil {
				return errors.New("gagal membuat admin: data owner tidak valid atau tidak terdaftar")
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

func (r *authRepository) FindByID(id string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", id).First(&account).Error
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