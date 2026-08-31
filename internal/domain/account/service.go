package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
)

var (
	nikRegex   = regexp.MustCompile(`^[0-9]{16}$`)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^[0-9+]{9,16}$`)
)

func validateKtpNumber(ktp *string) error {
	if ktp == nil || strings.TrimSpace(*ktp) == "" {
		return nil
	}
	clean := strings.TrimSpace(*ktp)
	if !nikRegex.MatchString(clean) {
		return errors.New("Nomor KTP/NIK harus terdiri dari 16 digit angka")
	}
	return nil
}

func validateEmail(email *string) error {
	if email == nil || strings.TrimSpace(*email) == "" {
		return nil
	}
	clean := strings.TrimSpace(*email)
	if !emailRegex.MatchString(clean) {
		return errors.New("Format email tidak valid")
	}
	return nil
}

func validatePhoneNumber(phone *string) error {
	if phone == nil || strings.TrimSpace(*phone) == "" {
		return nil
	}
	clean := strings.TrimSpace(*phone)
	if !phoneRegex.MatchString(clean) {
		return errors.New("Nomor telepon tidak valid (minimal 9 digit angka)")
	}
	return nil
}

func validateDate(dateStr *string, fieldName string) error {
	if dateStr == nil || strings.TrimSpace(*dateStr) == "" {
		return nil
	}
	clean := strings.TrimSpace(*dateStr)
	if _, err := time.Parse("2006-01-02", clean); err != nil {
		return fmt.Errorf("Format %s tidak valid (gunakan format YYYY-MM-DD)", fieldName)
	}
	return nil
}

func encryptOwnerKTP(owner *models.Owner) error {
	if owner == nil {
		return nil
	}
	var err error
	if owner.KtpNumber, err = crypto.EncryptPointer(owner.KtpNumber); err != nil {
		return err
	}
	if owner.KtpPhotoUrl, err = crypto.EncryptPointer(owner.KtpPhotoUrl); err != nil {
		return err
	}
	if owner.KtpName, err = crypto.EncryptPointer(owner.KtpName); err != nil {
		return err
	}
	if owner.KtpBirthPlace, err = crypto.EncryptPointer(owner.KtpBirthPlace); err != nil {
		return err
	}
	if owner.KtpGender, err = crypto.EncryptPointer(owner.KtpGender); err != nil {
		return err
	}
	if owner.KtpAddress, err = crypto.EncryptPointer(owner.KtpAddress); err != nil {
		return err
	}
	if owner.KtpRtRw, err = crypto.EncryptPointer(owner.KtpRtRw); err != nil {
		return err
	}
	if owner.KtpVillage, err = crypto.EncryptPointer(owner.KtpVillage); err != nil {
		return err
	}
	if owner.KtpDistrict, err = crypto.EncryptPointer(owner.KtpDistrict); err != nil {
		return err
	}
	if owner.KtpReligion, err = crypto.EncryptPointer(owner.KtpReligion); err != nil {
		return err
	}
	if owner.KtpOccupation, err = crypto.EncryptPointer(owner.KtpOccupation); err != nil {
		return err
	}
	if owner.KtpCitizenship, err = crypto.EncryptPointer(owner.KtpCitizenship); err != nil {
		return err
	}
	return nil
}

func decryptOwnerKTP(owner *models.Owner) {
	if owner == nil {
		return
	}
	owner.KtpNumber, _ = crypto.DecryptPointer(owner.KtpNumber)
	owner.KtpPhotoUrl, _ = crypto.DecryptPointer(owner.KtpPhotoUrl)
	owner.KtpName, _ = crypto.DecryptPointer(owner.KtpName)
	owner.KtpBirthPlace, _ = crypto.DecryptPointer(owner.KtpBirthPlace)
	owner.KtpGender, _ = crypto.DecryptPointer(owner.KtpGender)
	owner.KtpAddress, _ = crypto.DecryptPointer(owner.KtpAddress)
	owner.KtpRtRw, _ = crypto.DecryptPointer(owner.KtpRtRw)
	owner.KtpVillage, _ = crypto.DecryptPointer(owner.KtpVillage)
	owner.KtpDistrict, _ = crypto.DecryptPointer(owner.KtpDistrict)
	owner.KtpReligion, _ = crypto.DecryptPointer(owner.KtpReligion)
	owner.KtpOccupation, _ = crypto.DecryptPointer(owner.KtpOccupation)
	owner.KtpCitizenship, _ = crypto.DecryptPointer(owner.KtpCitizenship)
}

func encryptBusinessAdminKTP(admin *models.BusinessAdmin) error {
	if admin == nil {
		return nil
	}
	var err error
	if admin.KtpNumber, err = crypto.EncryptPointer(admin.KtpNumber); err != nil {
		return err
	}
	if admin.KtpPhotoUrl, err = crypto.EncryptPointer(admin.KtpPhotoUrl); err != nil {
		return err
	}
	if admin.KtpName, err = crypto.EncryptPointer(admin.KtpName); err != nil {
		return err
	}
	if admin.KtpBirthPlace, err = crypto.EncryptPointer(admin.KtpBirthPlace); err != nil {
		return err
	}
	if admin.KtpGender, err = crypto.EncryptPointer(admin.KtpGender); err != nil {
		return err
	}
	if admin.KtpAddress, err = crypto.EncryptPointer(admin.KtpAddress); err != nil {
		return err
	}
	if admin.KtpRtRw, err = crypto.EncryptPointer(admin.KtpRtRw); err != nil {
		return err
	}
	if admin.KtpVillage, err = crypto.EncryptPointer(admin.KtpVillage); err != nil {
		return err
	}
	if admin.KtpDistrict, err = crypto.EncryptPointer(admin.KtpDistrict); err != nil {
		return err
	}
	if admin.KtpReligion, err = crypto.EncryptPointer(admin.KtpReligion); err != nil {
		return err
	}
	if admin.KtpOccupation, err = crypto.EncryptPointer(admin.KtpOccupation); err != nil {
		return err
	}
	if admin.KtpCitizenship, err = crypto.EncryptPointer(admin.KtpCitizenship); err != nil {
		return err
	}
	return nil
}

func decryptBusinessAdminKTP(admin *models.BusinessAdmin) {
	if admin == nil {
		return
	}
	admin.KtpNumber, _ = crypto.DecryptPointer(admin.KtpNumber)
	admin.KtpPhotoUrl, _ = crypto.DecryptPointer(admin.KtpPhotoUrl)
	admin.KtpName, _ = crypto.DecryptPointer(admin.KtpName)
	admin.KtpBirthPlace, _ = crypto.DecryptPointer(admin.KtpBirthPlace)
	admin.KtpGender, _ = crypto.DecryptPointer(admin.KtpGender)
	admin.KtpAddress, _ = crypto.DecryptPointer(admin.KtpAddress)
	admin.KtpRtRw, _ = crypto.DecryptPointer(admin.KtpRtRw)
	admin.KtpVillage, _ = crypto.DecryptPointer(admin.KtpVillage)
	admin.KtpDistrict, _ = crypto.DecryptPointer(admin.KtpDistrict)
	admin.KtpReligion, _ = crypto.DecryptPointer(admin.KtpReligion)
	admin.KtpOccupation, _ = crypto.DecryptPointer(admin.KtpOccupation)
	admin.KtpCitizenship, _ = crypto.DecryptPointer(admin.KtpCitizenship)
}

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
		decryptOwnerKTP(owner)
		account.Owner = owner
	} else if account.Role == "business_admin" {
		admin, _ := s.repo.GetBusinessAdminByAccountID(accountID)
		decryptBusinessAdminKTP(admin)
		account.BusinessAdmin = admin
	}

	return account, nil
}

func (s *profileService) UpdateCustomerProfile(accountID string, input models.UpdateProfileCustomerInput) (*models.Account, error) {
	if err := validateEmail(input.Email); err != nil {
		return nil, err
	}
	if err := validatePhoneNumber(input.PhoneNumber); err != nil {
		return nil, err
	}
	if err := validateDate(input.BirthDate, "tanggal lahir"); err != nil {
		return nil, err
	}

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
	if err := validateEmail(input.Email); err != nil {
		return nil, err
	}
	if err := validatePhoneNumber(input.PhoneNumber); err != nil {
		return nil, err
	}
	if err := validateKtpNumber(input.KtpNumber); err != nil {
		return nil, err
	}
	if err := validateDate(input.KtpBirthDate, "tanggal lahir"); err != nil {
		return nil, err
	}

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
	decryptOwnerKTP(owner)

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

	ownerToSave := *owner
	if err := encryptOwnerKTP(&ownerToSave); err != nil {
		return nil, errors.New("failed to encrypt sensitive owner profile data")
	}

	if err := s.repo.SaveOwnerProfile(account, &ownerToSave); err != nil {
		return nil, errors.New("failed to save owner profile data")
	}
	return account, nil
}

func (s *profileService) UpdateBusinessAdminProfile(accountID string, input models.UpdateBusinessAdminKYCInput) (*models.Account, error) {
	if err := validateEmail(input.Email); err != nil {
		return nil, err
	}
	if err := validatePhoneNumber(input.PhoneNumber); err != nil {
		return nil, err
	}
	if err := validateKtpNumber(input.KtpNumber); err != nil {
		return nil, err
	}
	if err := validateDate(input.KtpBirthDate, "tanggal lahir"); err != nil {
		return nil, err
	}

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
	decryptBusinessAdminKTP(admin)

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

	adminToSave := *admin
	if err := encryptBusinessAdminKTP(&adminToSave); err != nil {
		return nil, errors.New("failed to encrypt sensitive business admin profile data")
	}

	if err := s.repo.SaveBusinessAdminProfile(account, &adminToSave); err != nil {
		return nil, errors.New("failed to save business admin profile data")
	}

	return account, nil
}


