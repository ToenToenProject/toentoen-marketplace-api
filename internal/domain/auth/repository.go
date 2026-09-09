package auth

import (
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

type AuthRepository interface {
	CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error
	UpdateAccountStatus(identifier string, status string) error
	UpdateEmailVerified(identifier string, verifiedAt time.Time, newStatus string) error
	UpdatePhoneVerified(identifier string, verifiedAt time.Time, newStatus string) error
	FindByIdentifier(identifier string) (*models.Account, error)
	FindByID(id string) (*models.Account, error)
	Update(account *models.Account) error
	UpdateEmail(accountID string, newEmail string) error
	UpdatePhoneNumber(accountID string, newPhone string) error
	LoadAssociations(account *models.Account) error
	FindReferralCode(code string) (*models.ReferralUser, error)
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db}
}

func (r *authRepository) CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Owner", "Customer", "BusinessAdmin").Create(account).Error; err != nil {
			return err
		}

		switch role {
		case "customer":
			if err := tx.Create(&models.Customer{AccountID: account.AccountID}).Error; err != nil {
				return err
			}

		case "owner":
			ownerRecord := models.Owner{AccountID: account.AccountID}
			if account.Owner != nil && account.Owner.ReferralCode != nil {
				ownerRecord.ReferralCode = account.Owner.ReferralCode
			}
			if err := tx.Create(&ownerRecord).Error; err != nil {
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
		Where("LOWER(email) = LOWER(?) OR phone_number = ?", identifier, identifier).
		Update("account_status", status).Error
}

func (r *authRepository) UpdateEmailVerified(identifier string, verifiedAt time.Time, newStatus string) error {
	updates := map[string]interface{}{
		"email_verified_at": verifiedAt,
	}
	if newStatus != "" {
		updates["account_status"] = newStatus
	}
	return r.db.Model(&models.Account{}).
		Where("LOWER(email) = LOWER(?) OR phone_number = ?", identifier, identifier).
		Updates(updates).Error
}

func (r *authRepository) UpdatePhoneVerified(identifier string, verifiedAt time.Time, newStatus string) error {
	updates := map[string]interface{}{
		"phone_verified_at": verifiedAt,
	}
	if newStatus != "" {
		updates["account_status"] = newStatus
	}
	return r.db.Model(&models.Account{}).
		Where("LOWER(email) = LOWER(?) OR phone_number = ?", identifier, identifier).
		Updates(updates).Error
}

func (r *authRepository) FindByIdentifier(identifier string) (*models.Account, error) {
	var account models.Account
	identifier = strings.TrimSpace(identifier)

	if strings.Contains(identifier, "@") {
		err := r.db.Where("LOWER(email) = LOWER(?)", identifier).First(&account).Error
		if err != nil {
			return nil, err
		}
		return &account, nil
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
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *authRepository) FindByID(id string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", id).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *authRepository) Update(account *models.Account) error {
	return r.db.Save(account).Error
}

func (r *authRepository) UpdateEmail(accountID string, newEmail string) error {
	now := time.Now()
	return r.db.Model(&models.Account{}).
		Where("account_id = ?", accountID).
		Updates(map[string]interface{}{
			"email":             newEmail,
			"email_verified_at": &now,
			"account_status":    "active",
		}).Error
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
	}
	return nil
}

func (r *authRepository) FindReferralCode(code string) (*models.ReferralUser, error) {
	var user models.ReferralUser
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, gorm.ErrRecordNotFound
	}
	err := r.db.Table("operations.users").
		Where("LOWER(referral_code) = LOWER(?) AND is_active = true", cleanCode).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}