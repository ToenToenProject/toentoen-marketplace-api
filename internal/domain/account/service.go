package account

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

type ProfileService interface {
	UpdateCustomerProfile(accountID string, input models.UpdateProfileCustomerInput) (*models.Account, error)
	UpdateOwnerProfile(accountID string, input models.UpdateProfileOwnerInput) (*models.Account, error)
	UpdateBusinessAdminProfile(accountID string, input models.UpdateBusinessAdminKYCInput) (*models.Account, error)
	GetMyProfile(accountID string) (*models.Account, error)
	RequestDeleteOTP(accountID string) error
	VerifyDeleteOTPAndExecute(accountID string, otp string) error
	ForceDeleteAccount(identifier string) error
}

type profileService struct {
	repo ProfileRepository
}

func NewProfileService(repo ProfileRepository) ProfileService {
	return &profileService{repo}
}

func isAccountDataComplete(account *models.Account) string {
	if account == nil {
		return "inactive"
	}

	switch account.Role {
	case "owner":
		if account.Email != nil &&
			account.PhoneNumber != nil &&
			account.FirstName != "" &&
			account.Username != "" &&
			account.PasswordHash != nil &&
			account.Gender != nil &&
			account.Owner != nil &&
			account.Owner.KtpNumber != nil &&
			*account.Owner.KtpNumber != "" &&
			account.Owner.KtpPhotoUrl != nil &&
			*account.Owner.KtpPhotoUrl != "" {
			return "active"
		}
		return "inactive"

	case "business_admin":
		if account.BusinessAdmin != nil &&
			account.BusinessAdmin.KtpNumber != nil && *account.BusinessAdmin.KtpNumber != "" &&
			account.BusinessAdmin.KtpPhotoUrl != nil && *account.BusinessAdmin.KtpPhotoUrl != "" {
			return "active"
		}
		return "inactive"

	case "customer":
		// Akun customer sudah aktif sejak OTP, jangan sampai ditimpa jadi inactive
		return "active"

	default:
		return "inactive"
	}
}

func (s *profileService) GetMyProfile(accountID string) (*models.Account, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("akun tidak ditemukan")
		}
		return nil, err
	}

	if account.Role == "customer" {
		customer, _ := s.repo.GetCustomerByAccountID(accountID)
		account.Customer = customer
	} else if account.Role == "owner" {
		owner, _ := s.repo.GetOwnerByAccountID(accountID)
		account.Owner = owner
	} else if account.Role == "business_admin" {
		admin, _ := s.repo.GetBusinessAdminByAccountID(accountID)
		account.BusinessAdmin = admin
	}

	return account, nil
}

func (s *profileService) UpdateCustomerProfile(accountID string, input models.UpdateProfileCustomerInput) (*models.Account, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	if input.FirstName != nil {
		account.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		account.LastName = *input.LastName
	}
	if input.Username != nil && strings.TrimSpace(*input.Username) != "" {
		cleanUsername := strings.TrimSpace(*input.Username)
		exists, err := s.repo.CheckUsernameExists(cleanUsername, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("username already in use")
		}
		account.Username = cleanUsername
	}
	if input.Email != nil && *input.Email != "" {
		exists, err := s.repo.CheckEmailExists(*input.Email, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("email already in use")
		}
		account.Email = input.Email
	}
	if input.PhoneNumber != nil && *input.PhoneNumber != "" {
		exists, err := s.repo.CheckPhoneNumberExists(*input.PhoneNumber, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("phone number already in use")
		}
		account.PhoneNumber = input.PhoneNumber
	}
	if input.Gender != nil {
		account.Gender = input.Gender
	}

	customer, err := s.repo.GetCustomerByAccountID(accountID)
	if err != nil {
		return nil, errors.New("user specific profile data not found")
	}

	if input.Bio != nil {
		customer.Bio = input.Bio
	}
	if input.ProfilePictureUrl != nil {
		account.ProfilePictureUrl = input.ProfilePictureUrl
	}
	if input.BirthDate != nil {
		if *input.BirthDate == "" {
			customer.BirthDate = nil
		} else {
			customer.BirthDate = input.BirthDate
		}
	}

	account.Customer = customer
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveCustomerProfile(account, customer); err != nil {
		return nil, errors.New("failed to save user profile data")
	}

	return account, nil
}

func (s *profileService) RequestDeleteOTP(accountID string) error {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("delete_otp:%s", accountID)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode OTP")
	}

	// Send OTP via Email or WhatsApp depending on which one is available
	if account.Email != nil && *account.Email != "" {
		go utils.SendEmailOTP(*account.Email, otp)
	} else if account.PhoneNumber != nil && *account.PhoneNumber != "" {
		go utils.SendWAOTP(*account.PhoneNumber, otp)
	} else {
		return errors.New("Tidak ada Email/No. HP untuk mengirim OTP")
	}

	return nil
}

func (s *profileService) VerifyDeleteOTPAndExecute(accountID string, otp string) error {
	redisKey := fmt.Sprintf("delete_otp:%s", accountID)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()
	if err != nil || storedOTP != otp {
		return errors.New("OTP salah atau kedaluwarsa")
	}

	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return errors.New("akun tidak ditemukan")
	}

	// Execute deletion in a transaction
	deletedBusinessIDs, deletedProductIDs, err := s.repo.DeleteAccountTx(accountID, account.Role)
	if err != nil {
		return errors.New("Gagal menghapus akun: " + err.Error())
	}

	// Clean up Redis
	config.RDB.Del(config.Ctx, redisKey)

	// Clean up Meilisearch
	if len(deletedBusinessIDs) > 0 || len(deletedProductIDs) > 0 {
		go func() {
			for _, bID := range deletedBusinessIDs {
				meilisearch.Client.Index("businesses").DeleteDocument(bID, nil)
			}
			for _, pID := range deletedProductIDs {
				meilisearch.Client.Index("products").DeleteDocument(pID, nil)
			}
		}()
	}

	return nil
}

func (s *profileService) ForceDeleteAccount(identifier string) error {
	account, err := s.repo.GetAccountByIdentifier(identifier)
	if err != nil {
		return errors.New("akun tidak ditemukan")
	}

	// Execute deletion in a transaction
	deletedBusinessIDs, deletedProductIDs, err := s.repo.DeleteAccountTx(account.AccountID, account.Role)
	if err != nil {
		return errors.New("Gagal menghapus akun: " + err.Error())
	}

	// Clean up Meilisearch
	if len(deletedBusinessIDs) > 0 || len(deletedProductIDs) > 0 {
		go func() {
			for _, bID := range deletedBusinessIDs {
				meilisearch.Client.Index("businesses").DeleteDocument(bID, nil)
			}
			for _, pID := range deletedProductIDs {
				meilisearch.Client.Index("products").DeleteDocument(pID, nil)
			}
		}()
	}

	return nil
}

func (s *profileService) UpdateOwnerProfile(accountID string, input models.UpdateProfileOwnerInput) (*models.Account, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	if input.FirstName != nil {
		account.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		account.LastName = *input.LastName
	}
	if input.ProfilePictureUrl != nil {
		account.ProfilePictureUrl = input.ProfilePictureUrl
	}
	if input.Username != nil && strings.TrimSpace(*input.Username) != "" {
		cleanUsername := strings.TrimSpace(*input.Username)
		exists, err := s.repo.CheckUsernameExists(cleanUsername, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("username already in use")
		}
		account.Username = cleanUsername
	}
	if input.Email != nil && *input.Email != "" {
		exists, err := s.repo.CheckEmailExists(*input.Email, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("email already in use")
		}
		account.Email = input.Email
	}
	if input.PhoneNumber != nil && *input.PhoneNumber != "" {
		exists, err := s.repo.CheckPhoneNumberExists(*input.PhoneNumber, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("phone number already in use")
		}
		account.PhoneNumber = input.PhoneNumber
	}
	if input.Title != nil {
		title := strings.ToLower(*input.Title)
		var gender string
		if title == "mr" || title == "mr." {
			gender = "male"
		} else if title == "mrs" || title == "mrs." || title == "ms" || title == "ms." || title == "miss" {
			gender = "female"
		} else {
			gender = "other"
		}
		account.Gender = &gender
	}

	owner, err := s.repo.GetOwnerByAccountID(accountID)
	if err != nil {
		return nil, errors.New("user specific profile data not found")
	}

	if input.KtpNumber != nil {
		if *input.KtpNumber == "" {
			owner.KtpNumber = nil
		} else {
			exists, err := s.repo.CheckKtpNumberExists(*input.KtpNumber, accountID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, errors.New("ktp number already in use")
			}
			owner.KtpNumber = input.KtpNumber
		}
	}
	if input.KtpPhotoUrl != nil {
		if *input.KtpPhotoUrl == "" {
			owner.KtpPhotoUrl = nil
		} else {
			owner.KtpPhotoUrl = input.KtpPhotoUrl
		}
	}

	if input.ReferralCode != nil {
		if *input.ReferralCode == "" {
			owner.ReferralCode = nil
		} else {
			owner.ReferralCode = input.ReferralCode
		}
	}
	if input.Country != nil {
		owner.Country = input.Country
	}
	if input.KtpName != nil {
		owner.KtpName = input.KtpName
	}
	if input.KtpBirthPlace != nil {
		owner.KtpBirthPlace = input.KtpBirthPlace
	}
	if input.KtpBirthDate != nil {
		owner.KtpBirthDate = input.KtpBirthDate
	}
	if input.KtpGender != nil {
		owner.KtpGender = input.KtpGender
	}
	if input.KtpAddress != nil {
		owner.KtpAddress = input.KtpAddress
	}
	if input.KtpRtRw != nil {
		owner.KtpRtRw = input.KtpRtRw
	}
	if input.KtpVillage != nil {
		owner.KtpVillage = input.KtpVillage
	}
	if input.KtpDistrict != nil {
		owner.KtpDistrict = input.KtpDistrict
	}
	if input.KtpReligion != nil {
		owner.KtpReligion = input.KtpReligion
	}
	if input.KtpOccupation != nil {
		owner.KtpOccupation = input.KtpOccupation
	}
	if input.KtpCitizenship != nil {
		owner.KtpCitizenship = input.KtpCitizenship
	}

	account.Owner = owner
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveOwnerProfile(account, owner); err != nil {
		return nil, errors.New("failed to save owner profile data")
	}
	return account, nil
}

func (s *profileService) UpdateBusinessAdminProfile(accountID string, input models.UpdateBusinessAdminKYCInput) (*models.Account, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	if input.FirstName != nil && *input.FirstName != "" {
		account.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		account.LastName = *input.LastName
	}
	if input.ProfilePictureUrl != nil {
		account.ProfilePictureUrl = input.ProfilePictureUrl
	}
	if input.Username != nil && strings.TrimSpace(*input.Username) != "" {
		cleanUsername := strings.TrimSpace(*input.Username)
		exists, err := s.repo.CheckUsernameExists(cleanUsername, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("username already in use")
		}
		account.Username = cleanUsername
	}
	if input.Email != nil && *input.Email != "" {
		exists, err := s.repo.CheckEmailExists(*input.Email, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("email already in use")
		}
		account.Email = input.Email
	}
	if input.PhoneNumber != nil && *input.PhoneNumber != "" {
		exists, err := s.repo.CheckPhoneNumberExists(*input.PhoneNumber, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("phone number already in use")
		}
		account.PhoneNumber = input.PhoneNumber
	}
	if input.Title != nil {
		title := strings.ToLower(*input.Title)
		var gender string
		if title == "mr" || title == "mr." {
			gender = "male"
		} else if title == "mrs" || title == "mrs." || title == "ms" || title == "ms." || title == "miss" {
			gender = "female"
		} else {
			gender = "other"
		}
		account.Gender = &gender
	}

	admin, err := s.repo.GetBusinessAdminByAccountID(accountID)
	if err != nil {
		admin = &models.BusinessAdmin{AccountID: accountID}
	}

	if input.KtpNumber != nil {
		if *input.KtpNumber == "" {
			admin.KtpNumber = nil
		} else {
			exists, err := s.repo.CheckKtpNumberExists(*input.KtpNumber, accountID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, errors.New("ktp number already in use")
			}
			admin.KtpNumber = input.KtpNumber
		}
	}
	if input.KtpPhotoUrl != nil {
		if *input.KtpPhotoUrl == "" {
			admin.KtpPhotoUrl = nil
		} else {
			admin.KtpPhotoUrl = input.KtpPhotoUrl
		}
	}
	if input.Country != nil {
		// Country disimpan di tabel owners, untuk admin kita skip atau bisa disimpan di account level jika ada kolom
		// Sementara diabaikan karena tidak ada kolom country di business_admins
		_ = input.Country
	}
	if input.KtpName != nil {
		admin.KtpName = input.KtpName
	}
	if input.KtpBirthPlace != nil {
		admin.KtpBirthPlace = input.KtpBirthPlace
	}
	if input.KtpBirthDate != nil {
		admin.KtpBirthDate = input.KtpBirthDate
	}
	if input.KtpGender != nil {
		admin.KtpGender = input.KtpGender
	}
	if input.KtpAddress != nil {
		admin.KtpAddress = input.KtpAddress
	}
	if input.KtpRtRw != nil {
		admin.KtpRtRw = input.KtpRtRw
	}
	if input.KtpVillage != nil {
		admin.KtpVillage = input.KtpVillage
	}
	if input.KtpDistrict != nil {
		admin.KtpDistrict = input.KtpDistrict
	}
	if input.KtpReligion != nil {
		admin.KtpReligion = input.KtpReligion
	}
	if input.KtpOccupation != nil {
		admin.KtpOccupation = input.KtpOccupation
	}
	if input.KtpCitizenship != nil {
		admin.KtpCitizenship = input.KtpCitizenship
	}

	account.BusinessAdmin = admin
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveBusinessAdminProfile(account, admin); err != nil {
		return nil, errors.New("failed to save business admin profile data")
	}

	return account, nil
}


