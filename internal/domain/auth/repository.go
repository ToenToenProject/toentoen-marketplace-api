package auth

import (
	"strings"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

type AuthRepository interface {
	CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error
	UpdateAccountStatus(identifier string, status string) error
	FindByIdentifier(identifier string) (*models.Account, error)
	FindByID(id string) (*models.Account, error)
	Update(account *models.Account) error
	UpdateEmail(accountID string, newEmail string) error
	UpdatePhoneNumber(accountID string, newPhone string) error
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
		case "customer":
			if err := tx.Create(&models.Customer{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "owner":
			if err := tx.Create(&models.Owner{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "business_admin":
			inviteCode := "ADM-" + utils.GenerateRandomString(6)
			if err := tx.Create(&models.BusinessAdmin{
				AccountID:      account.AccountID,
				InvitationCode: &inviteCode,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *authRepository) UpdateAccountStatus(identifier string, status string) error {
	return r.db.Model(&models.Account{}).
		Where("email = ? OR phone_number = ?", identifier, identifier).
		Update("account_status", status).Error
}

func (r *authRepository) FindByIdentifier(identifier string) (*models.Account, error) {
	var account models.Account
	identifier = strings.TrimSpace(identifier)

	if strings.Contains(identifier, "@") {
		err := r.db.Where("LOWER(email) = LOWER(?)", identifier).First(&account).Error
		return &account, err
	}

	// Buat variasi format nomor telepon (08xxx, 628xxx, +628xxx)
	clean := strings.TrimPrefix(identifier, "+")
	var phones []string
	if strings.HasPrefix(clean, "08") {
		phones = []string{clean, "62" + clean[1:], "+62" + clean[1:]}
	} else if strings.HasPrefix(clean, "628") {
		phones = []string{clean, "0" + clean[2:], "+" + clean}
	} else {
		phones = []string{identifier, "+" + identifier, clean}
	}

	err := r.db.Where("phone_number IN ? OR LOWER(username) = LOWER(?)", phones, identifier).First(&account).Error
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

func (r *authRepository) UpdateEmail(accountID string, newEmail string) error {
	return r.db.Model(&models.Account{}).
		Where("account_id = ?", accountID).
		Update("email", newEmail).Error
}

func (r *authRepository) UpdatePhoneNumber(accountID string, newPhone string) error {
	return r.db.Model(&models.Account{}).
		Where("account_id = ?", accountID).
		Update("phone_number", newPhone).Error
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