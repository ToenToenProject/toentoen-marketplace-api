package auth

import (
	"errors"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/kychelper"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

// ErrAmbiguousAccount dikembalikan ketika identifier yang sama terdaftar di lebih dari
// satu role (customer, owner, business_admin) dan request tidak menyertakan role spesifik.
var ErrAmbiguousAccount = errors.New("Email/nomor ini terdaftar di lebih dari satu akun (Pelanggan, Pemilik Toko, atau Admin). Sertakan parameter role (customer, owner, atau business_admin) untuk melanjutkan.")

type AuthRepository interface {
	CreateAccountWithProfile(account *models.Account, role string, ownerID ...string) error
	UpdateAccountStatus(accountID string, status string) error
	UpdateEmailVerified(accountID string, verifiedAt time.Time, newStatus string) error
	UpdatePhoneVerified(accountID string, verifiedAt time.Time, newStatus string) error
	// FindByIdentifier mencari akun berdasarkan email/telepon/username di dalam kelompok role.
	// roleGroup kosong = cari di semua kelompok; jika ditemukan >1 akun, mengembalikan ErrAmbiguousAccount.
	FindByIdentifier(identifier string, roleGroup string) (*models.Account, error)
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
			// Wariskan data KYC terverifikasi jika pengguna sudah memiliki akun Admin yang terverifikasi
			kychelper.InheritVerifiedKYCOnCreate(tx, account, "owner", &ownerRecord, nil)
			if err := tx.Create(&ownerRecord).Error; err != nil {
				return err
			}

		case "business_admin":
			inviteCode := "ADM-" + utils.GenerateRandomString(6)
			adminRecord := models.BusinessAdmin{
				AccountID:      account.AccountID,
				InvitationCode: &inviteCode,
			}
			// Wariskan data KYC terverifikasi jika pengguna sudah memiliki akun Owner yang terverifikasi
			kychelper.InheritVerifiedKYCOnCreate(tx, account, "business_admin", nil, &adminRecord)
			if err := tx.Create(&adminRecord).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *authRepository) UpdateAccountStatus(accountID string, status string) error {
	return r.db.Model(&models.Account{}).
		Where("account_id = ?", accountID).
		Update("account_status", status).Error
}

func (r *authRepository) UpdateEmailVerified(accountID string, verifiedAt time.Time, newStatus string) error {
	updates := map[string]interface{}{
		"email_verified_at": verifiedAt,
	}
	if newStatus != "" {
		updates["account_status"] = newStatus
	}
	return r.db.Model(&models.Account{}).
		Where("account_id = ?", accountID).
		Updates(updates).Error
}

func (r *authRepository) UpdatePhoneVerified(accountID string, verifiedAt time.Time, newStatus string) error {
	updates := map[string]interface{}{
		"phone_verified_at": verifiedAt,
	}
	if newStatus != "" {
		updates["account_status"] = newStatus
	}
	return r.db.Model(&models.Account{}).
		Where("account_id = ?", accountID).
		Updates(updates).Error
}

// PhoneVariants membuat variasi format nomor telepon (08xxx, 628xxx, +628xxx)
func PhoneVariants(identifier string) []string {
	clean := strings.TrimPrefix(identifier, "+")
	if strings.HasPrefix(clean, "08") {
		return []string{clean, "62" + clean[1:], "+62" + clean[1:]}
	} else if strings.HasPrefix(clean, "628") {
		return []string{clean, "0" + clean[2:], "+" + clean}
	}
	return []string{identifier, "+" + identifier, clean}
}

func (r *authRepository) FindByIdentifier(identifier string, roleGroup string) (*models.Account, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, gorm.ErrRecordNotFound
	}

	query := r.db.Model(&models.Account{})
	if strings.Contains(identifier, "@") {
		query = query.Where("LOWER(email) = LOWER(?)", identifier)
	} else {
		query = query.Where("(phone_number IN ? OR LOWER(username) = LOWER(?))", PhoneVariants(identifier), identifier)
	}

	if roles := models.RolesInGroup(roleGroup); len(roles) > 0 {
		query = query.Where("role IN ?", roles)
	}

	var accounts []models.Account
	if err := query.Order("created_at ASC").Limit(2).Find(&accounts).Error; err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(accounts) > 1 {
		return nil, ErrAmbiguousAccount
	}
	return &accounts[0], nil
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