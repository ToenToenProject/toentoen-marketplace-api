package repositories

import (
	"time"
	"toentoen-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthRepository interface {
	CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error
	UpdateAccountStatus(email string, status string) error
	FindByIdentifier(identifier string) (*models.Account, error)
	FindByID(id string) (*models.Account, error) 
	Update(account *models.Account) error
	LoadAssociations(account *models.Account) error
	SaveDeviceToken(accountID, fcmToken, deviceType string) error
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
		case "customer":
			if err := tx.Create(&models.Customer{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "owner":
			if err := tx.Create(&models.Owner{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "business_admin":
			if err := tx.Create(&models.BusinessAdmin{AccountID: account.AccountID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *authRepository) UpdateAccountStatus(email string, status string) error {
	// Query: UPDATE accounts SET account_status = 'active' WHERE email = 'xxx'
	return r.db.Model(&models.Account{}).
		Where("email = ?", email).
		Update("account_status", status).Error
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
	case "customer":
		r.db.Model(account).Association("Customer").Find(&account.Customer)
	case "owner":
		r.db.Model(account).Association("Owner").Find(&account.Owner)
	case "business_admin":
		r.db.Model(account).Association("BusinessAdmin").Find(&account.BusinessAdmin)
	case "platform_admin":
		r.db.Model(account).Association("PlatformAdmin").Find(&account.PlatformAdmin)
	}
	return nil
}

func (r *authRepository) SaveDeviceToken(accountID, fcmToken, deviceType string) error {
	// Asumsi Anda menaruh struct DeviceToken di folder models
	token := models.DeviceToken{
		AccountID:  accountID,
		FCMToken:   fcmToken,
		DeviceType: deviceType,
		UpdatedAt:  time.Now(),
	}

	// r.db adalah instance GORM bawaan dari struct authRepository Anda
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fcm_token"}},
		DoUpdates: clause.AssignmentColumns([]string{"account_id", "updated_at"}),
	}).Create(&token).Error
}