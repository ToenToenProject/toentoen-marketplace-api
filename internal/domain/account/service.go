package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"os"
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
	// Hapus karakter pemisah umum seperti spasi, strip, tanda kurung
	sanitized := strings.ReplaceAll(clean, " ", "")
	sanitized = strings.ReplaceAll(sanitized, "-", "")
	sanitized = strings.ReplaceAll(sanitized, "(", "")
	sanitized = strings.ReplaceAll(sanitized, ")", "")

	// Standar internasional E.164 & lokal: opsional diawali '+' lalu 7 sampai 15 digit angka
	// Mendukung +628..., 08..., 628..., +1..., +44..., +65..., dan seluruh kode negara di dunia
	phonePattern := regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	if !phonePattern.MatchString(sanitized) {
		return errors.New("Nomor telepon tidak valid (masukkan nomor valid dengan format lokal seperti 08... atau internasional seperti +62... / kode negara lainnya)")
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
	if owner.SelfiePhotoUrl, err = crypto.EncryptPointer(owner.SelfiePhotoUrl); err != nil {
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
	owner.SelfiePhotoUrl, _ = crypto.DecryptPointer(owner.SelfiePhotoUrl)
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
	if admin.SelfiePhotoUrl, err = crypto.EncryptPointer(admin.SelfiePhotoUrl); err != nil {
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
	admin.SelfiePhotoUrl, _ = crypto.DecryptPointer(admin.SelfiePhotoUrl)
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
	GetKYCStatus(accountID string) (*models.KYCStatusResponse, error)
	SubmitKYC(accountID string, input models.SubmitKYCInput) error
	UpdateConsent(accountID string, input models.ConsentInput) (*models.Account, error)
	RequestDeleteOTP(accountID string) error
	VerifyDeleteOTPAndExecute(accountID string, otp string) error
	ForceDeleteAccount(identifier string) error
	VerifyOperationsAuthorization(authHeader string, cookieToken string, workerID string, opsUserID string, internalSecret string) (*models.OperationsUser, bool, error)
	SetNotificationService(notif notification.NotificationService)
}

type profileService struct {
	repo         ProfileRepository
	notifService notification.NotificationService
}

func (s *profileService) SetNotificationService(notif notification.NotificationService) {
	s.notifService = notif
}

func NewProfileService(repo ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

func isAccountDataComplete(account *models.Account) string {
	if account == nil {
		return "inactive"
	}

	// Jangan mengubah status jika akun sedang ditangguhkan atau diblokir
	if account.AccountStatus == "suspended" || account.AccountStatus == "banned" {
		return account.AccountStatus
	}

	// Karena verifikasi via nomor telepon belum tersedia, status akun aktif/verified
	// hanya ditentukan jika pengguna sudah memiliki email terdaftar dan terverifikasi
	if account.Email != nil && strings.TrimSpace(*account.Email) != "" {
		return "active"
	}

	return "inactive"
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
	if err := validatePhoneNumber(input.PhoneNumber); err != nil {
		return nil, err
	}
	if err := validateDate(input.BirthDate, "tanggal lahir"); err != nil {
		return nil, err
	}

	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Akun tidak ditemukan")
		}
		return nil, err
	}

	if input.FirstName != nil {
		cleanFirst := strings.TrimSpace(*input.FirstName)
		if cleanFirst == "" {
			return nil, errors.New("Nama depan wajib diisi dan tidak boleh kosong")
		}
		account.FirstName = cleanFirst
	}
	if input.LastName != nil {
		account.LastName = strings.TrimSpace(*input.LastName)
	}
	if input.Username != nil && strings.TrimSpace(*input.Username) != "" {
		cleanUsername := strings.TrimSpace(*input.Username)
		exists, err := s.repo.CheckUsernameExists(cleanUsername, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("Username sudah digunakan")
		}
		account.Username = cleanUsername
	}
	if input.PhoneNumber != nil && *input.PhoneNumber != "" {
		exists, err := s.repo.CheckPhoneNumberExists(*input.PhoneNumber, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("Nomor telepon sudah digunakan")
		}
		account.PhoneNumber = input.PhoneNumber
	}
	if input.Gender != nil {
		cleanGender := strings.TrimSpace(*input.Gender)
		if cleanGender == "" {
			account.Gender = nil
		} else {
			account.Gender = &cleanGender
		}
	}

	customer, err := s.repo.GetCustomerByAccountID(accountID)
	if err != nil {
		return nil, errors.New("Data profil pengguna tidak ditemukan")
	}

	if input.Bio != nil {
		cleanBio := strings.TrimSpace(*input.Bio)
		if cleanBio == "" {
			customer.Bio = nil
		} else {
			customer.Bio = &cleanBio
		}
	}
	if input.ProfilePictureUrl != nil {
		cleanPic := strings.TrimSpace(*input.ProfilePictureUrl)
		if cleanPic == "" {
			account.ProfilePictureUrl = nil
		} else {
			account.ProfilePictureUrl = &cleanPic
		}
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
		return nil, errors.New("Gagal menyimpan data profil pengguna")
	}

	return account, nil
}

func (s *profileService) RequestDeleteOTP(accountID string) error {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	if account.Email == nil || strings.TrimSpace(*account.Email) == "" {
		if account.PhoneNumber != nil && strings.TrimSpace(*account.PhoneNumber) != "" {
			return errors.New("Layanan verifikasi OTP via nomor telepon sedang dalam tahap pengembangan. Akun Anda belum memiliki email untuk menerima kode OTP penghapusan akun.")
		}
		return errors.New("Akun Anda tidak memiliki email terdaftar untuk menerima kode OTP penghapusan akun")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("delete_otp:%s", accountID)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode OTP")
	}

	if err := utils.SendEmailOTP(*account.Email, otp, "Kode Verifikasi Penghapusan Akun ToenToen"); err != nil {
		config.RDB.Del(config.Ctx, redisKey)
		return fmt.Errorf("Gagal mengirim kode OTP ke email: %w", err)
	}

	return nil
}

func (s *profileService) VerifyDeleteOTPAndExecute(accountID string, otp string) error {
	redisKey := fmt.Sprintf("delete_otp:%s", accountID)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()
	cleanOTP := strings.TrimSpace(otp)
	if err != nil || storedOTP != cleanOTP {
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
	cleanIdentifier := strings.TrimSpace(identifier)
	account, err := s.repo.GetAccountByIdentifier(cleanIdentifier)
	if err != nil {
		return errors.New("akun tidak ditemukan")
	}

	// Execute deletion in a transaction
	deletedBusinessIDs, deletedProductIDs, err := s.repo.DeleteAccountTx(account.AccountID, account.Role)
	if err != nil {
		return errors.New("Gagal menghapus akun: " + err.Error())
	}

	// Clean up Redis keys
	if account.Email != nil && *account.Email != "" {
		config.RDB.Del(config.Ctx, fmt.Sprintf("login_otp:%s", *account.Email))
		config.RDB.Del(config.Ctx, fmt.Sprintf("register_otp:%s", *account.Email))
		config.RDB.Del(config.Ctx, fmt.Sprintf("pwd_otp:%s", *account.Email))
	}
	config.RDB.Del(config.Ctx, fmt.Sprintf("delete_otp:%s", account.AccountID))

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

// IsAllowedOperationsRole memeriksa apakah role dari skema operations berhak melakukan tindakan administratif tinggi seperti Force Delete
func IsAllowedOperationsRole(role string) bool {
	normalized := strings.ToLower(strings.TrimSpace(role))
	switch normalized {
	case "cto", "c_level", "c-level", "managerial", "admin_spv", "admin", "cs", "admin/cs", "cs/admin", "customer_service", "customer service", "customer_support", "support", "operations_admin", "super_admin", "superadmin":
		return true
	default:
		return false
	}
}

// VerifyOperationsAuthorization memvalidasi kredensial pemanggil dari skema operations (CTO, Admin, CS)
func (s *profileService) VerifyOperationsAuthorization(authHeader string, cookieToken string, workerID string, opsUserID string, internalSecret string) (*models.OperationsUser, bool, error) {
	// 1. Otorisasi via Internal Secret (service-to-service)
	if internalSecret != "" && internalSecret == config.Envs.JWTSecret {
		return &models.OperationsUser{
			WorkerID: "INTERNAL_SERVICE",
			Name:     "Internal Service",
			Role:     "cto",
			IsActive: true,
		}, true, nil
	}

	// 2. Otorisasi via JWT Bearer Token / Cookie
	tokenString := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	} else if authHeader != "" {
		tokenString = strings.TrimSpace(authHeader)
	}
	if tokenString == "" && cookieToken != "" {
		tokenString = cookieToken
	}

	if tokenString != "" {
		// Kumpulkan kemungkinan JWT Secret: secret webapi-go dan secret operations
		secrets := [][]byte{}
		if config.Envs.JWTSecret != "" {
			secrets = append(secrets, []byte(config.Envs.JWTSecret))
		}
		if opsSecret := os.Getenv("OPERATIONS_JWT_SECRET"); opsSecret != "" && opsSecret != config.Envs.JWTSecret {
			secrets = append(secrets, []byte(opsSecret))
		}
		// Fallback ke known operations JWT secret jika berbeda
		knownOpsSecret := "9cc90577eeb04dedcbadf38bb6abc6d98bb953129c59d42518c46cd073eecef5"
		if config.Envs.JWTSecret != knownOpsSecret {
			secrets = append(secrets, []byte(knownOpsSecret))
		}

		var claims jwt.MapClaims
		var tokenValid bool

		for _, sec := range secrets {
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("metode enkripsi token tidak valid")
				}
				return sec, nil
			})
			if err == nil && token != nil && token.Valid {
				if cl, ok := token.Claims.(jwt.MapClaims); ok {
					claims = cl
					tokenValid = true
					break
				}
			}
		}

		if tokenValid && claims != nil {
			roleFromClaim, _ := claims["role"].(string)
			accountID, _ := claims["account_id"].(string)
			if accountID == "" {
				accountID, _ = claims["sub"].(string)
			}
			if accountID == "" {
				accountID, _ = claims["id"].(string)
			}
			claimWorkerID, _ := claims["worker_id"].(string)
			if claimWorkerID == "" {
				claimWorkerID, _ = claims["workerId"].(string)
			}

			idToCheck := claimWorkerID
			if idToCheck == "" {
				idToCheck = accountID
			}

			// Jika ada identifier pengguna di claim, verifikasi ke database operations.users
			if idToCheck != "" {
				if opsUser, err := s.repo.FindOperationsUser(idToCheck); err == nil && opsUser != nil {
					if IsAllowedOperationsRole(opsUser.Role) {
						return opsUser, true, nil
					}
					return nil, false, fmt.Errorf("role Anda (%s) pada operations tidak memiliki izin force delete", opsUser.Role)
				}
			}

			// Jika tidak ditemukan di database atau idToCheck kosong, tapi role pada token valid
			if IsAllowedOperationsRole(roleFromClaim) {
				return &models.OperationsUser{
					ID:       accountID,
					WorkerID: claimWorkerID,
					Name:     "Operations User",
					Role:     roleFromClaim,
					IsActive: true,
				}, true, nil
			}
		}
	}

	// 3. Otorisasi via Header X-Worker-ID
	if workerID != "" {
		opsUser, err := s.repo.FindOperationsUser(workerID)
		if err == nil && opsUser != nil {
			if IsAllowedOperationsRole(opsUser.Role) {
				return opsUser, true, nil
			}
			return nil, false, fmt.Errorf("role Anda (%s) tidak memiliki izin force delete", opsUser.Role)
		}
	}

	// 4. Otorisasi via Header X-Operations-User-ID
	if opsUserID != "" {
		opsUser, err := s.repo.FindOperationsUser(opsUserID)
		if err == nil && opsUser != nil {
			if IsAllowedOperationsRole(opsUser.Role) {
				return opsUser, true, nil
			}
			return nil, false, fmt.Errorf("role Anda (%s) tidak memiliki izin force delete", opsUser.Role)
		}
	}

	// 5. Bypass untuk environment non-produksi (development / local testing)
	if config.Envs.AppEnv != "production" {
		return &models.OperationsUser{
			WorkerID: "DEV_ENV",
			Name:     "Development Tester",
			Role:     "cto",
			IsActive: true,
		}, true, nil
	}

	return nil, false, errors.New("akses ditolak: endpoint force delete hanya dapat diakses oleh role CTO, Admin, atau CS pada skema operations")
}

func (s *profileService) UpdateOwnerProfile(accountID string, input models.UpdateProfileOwnerInput) (*models.Account, error) {
	if err := validatePhoneNumber(input.PhoneNumber); err != nil {
		return nil, err
	}

	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Akun tidak ditemukan")
		}
		return nil, err
	}

	if input.FirstName != nil {
		cleanFirst := strings.TrimSpace(*input.FirstName)
		if cleanFirst == "" {
			return nil, errors.New("Nama depan wajib diisi dan tidak boleh kosong")
		}
		account.FirstName = cleanFirst
	}
	if input.LastName != nil {
		account.LastName = strings.TrimSpace(*input.LastName)
	}
	if input.ProfilePictureUrl != nil {
		cleanPic := strings.TrimSpace(*input.ProfilePictureUrl)
		if cleanPic == "" {
			account.ProfilePictureUrl = nil
		} else {
			account.ProfilePictureUrl = &cleanPic
		}
	}
	if input.Username != nil && strings.TrimSpace(*input.Username) != "" {
		cleanUsername := strings.TrimSpace(*input.Username)
		exists, err := s.repo.CheckUsernameExists(cleanUsername, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("Username sudah digunakan")
		}
		account.Username = cleanUsername
	}
	if input.PhoneNumber != nil && *input.PhoneNumber != "" {
		exists, err := s.repo.CheckPhoneNumberExists(*input.PhoneNumber, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("Nomor telepon sudah digunakan")
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
		return nil, errors.New("Data profil pengguna tidak ditemukan")
	}

	if input.ReferralCode != nil {
		cleanInput := strings.TrimSpace(*input.ReferralCode)
		hasExistingCode := owner.ReferralCode != nil && strings.TrimSpace(*owner.ReferralCode) != ""

		if hasExistingCode {
			if cleanInput == "" || !strings.EqualFold(cleanInput, strings.TrimSpace(*owner.ReferralCode)) {
				return nil, errors.New("Kode referral sudah terisi dan tidak dapat diubah kembali")
			}
		} else if cleanInput != "" {
			ref, err := s.repo.FindReferralCode(cleanInput)
			if err != nil || !ref.IsActive || ref.ReferralCode == nil || *ref.ReferralCode == "" {
				return nil, errors.New("Kode referral tidak ditemukan")
			}
			owner.ReferralCode = ref.ReferralCode
		}
	}
	if input.Country != nil {
		owner.Country = input.Country
	}

	account.Owner = owner
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveOwnerProfile(account, owner); err != nil {
		return nil, errors.New("Gagal menyimpan data profil pemilik")
	}
	return account, nil
}

func (s *profileService) UpdateBusinessAdminProfile(accountID string, input models.UpdateBusinessAdminKYCInput) (*models.Account, error) {
	if err := validatePhoneNumber(input.PhoneNumber); err != nil {
		return nil, err
	}

	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Akun tidak ditemukan")
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
			return nil, errors.New("Username sudah digunakan")
		}
		account.Username = cleanUsername
	}
	if input.PhoneNumber != nil && *input.PhoneNumber != "" {
		exists, err := s.repo.CheckPhoneNumberExists(*input.PhoneNumber, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("Nomor telepon sudah digunakan")
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

	account.BusinessAdmin = admin
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveBusinessAdminProfile(account, admin); err != nil {
		return nil, errors.New("Gagal menyimpan data profil admin bisnis")
	}

	return account, nil
}

func (s *profileService) GetKYCStatus(accountID string) (*models.KYCStatusResponse, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Akun tidak ditemukan")
		}
		return nil, err
	}

	res := &models.KYCStatusResponse{
		Role:      account.Role,
		KycStatus: "unverified",
	}

	switch account.Role {
	case "owner":
		owner, err := s.repo.GetOwnerByAccountID(accountID)
		if err == nil && owner != nil {
			if owner.KycStatus != "" {
				res.KycStatus = owner.KycStatus
			}
			res.RejectionReason = owner.KycRejectionReason
			res.SubmittedAt = owner.KycSubmittedAt
			res.VerifiedAt = owner.KycVerifiedAt
		}
	case "business_admin":
		admin, err := s.repo.GetBusinessAdminByAccountID(accountID)
		if err == nil && admin != nil {
			if admin.KycStatus != "" {
				res.KycStatus = admin.KycStatus
			}
			res.RejectionReason = admin.KycRejectionReason
			res.SubmittedAt = admin.KycSubmittedAt
			res.VerifiedAt = admin.KycVerifiedAt
		}
	default:
		res.KycStatus = "verified"
	}

	res.IsVerified = res.KycStatus == "verified"
	res.AcceptedTermsAt = account.AcceptedTermsAt
	res.TermsVersion = account.TermsVersion
	res.AcceptedPrivacyAt = account.AcceptedPrivacyAt
	res.PrivacyVersion = account.PrivacyVersion
	res.HasAcceptedTerms = account.AcceptedTermsAt != nil
	res.HasAcceptedPrivacy = account.AcceptedPrivacyAt != nil
	return res, nil
}

func (s *profileService) SubmitKYC(accountID string, input models.SubmitKYCInput) error {
	input.Normalize()

	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("Akun tidak ditemukan")
		}
		return err
	}

	if account.Role != "owner" && account.Role != "business_admin" {
		return errors.New("Pengajuan verifikasi identitas (KYC) hanya berlaku untuk merchant owner dan business admin")
	}

	// Periksa status pengajuan KYC saat ini: jika pending atau verified, jangan izinkan submit ulang
	switch account.Role {
	case "owner":
		owner, err := s.repo.GetOwnerByAccountID(accountID)
		if err == nil && owner != nil {
			if owner.KycStatus == "pending" {
				return errors.New("Dokumen identitas Anda sedang dalam proses verifikasi. Harap menunggu hingga proses verifikasi selesai.")
			}
			if owner.KycStatus == "verified" {
				return errors.New("Akun Anda sudah terverifikasi KYC")
			}
		}
	case "business_admin":
		admin, err := s.repo.GetBusinessAdminByAccountID(accountID)
		if err == nil && admin != nil {
			if admin.KycStatus == "pending" {
				return errors.New("Dokumen identitas Anda sedang dalam proses verifikasi. Harap menunggu hingga proses verifikasi selesai.")
			}
			if admin.KycStatus == "verified" {
				return errors.New("Akun Anda sudah terverifikasi KYC")
			}
		}
	}

	cleanKtp := strings.TrimSpace(input.KtpNumber)
	if err := validateKtpNumber(&cleanKtp); err != nil {
		return err
	}

	cleanPhoto := strings.TrimSpace(input.KtpPhotoUrl)
	if cleanPhoto == "" {
		return errors.New("Foto KTP wajib diunggah")
	}

	cleanSelfie := strings.TrimSpace(input.SelfiePhotoUrl)
	if cleanSelfie == "" {
		return errors.New("Foto selfie wajib diunggah")
	}

	if err := validateDate(input.KtpBirthDate, "tanggal lahir KTP"); err != nil {
		return err
	}

	exists, err := s.repo.CheckKtpNumberExists(cleanKtp, accountID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("Nomor KTP sudah terdaftar pada akun lain")
	}

	now := time.Now()
	newStatus := "pending"

	if input.AcceptedTerms != nil && *input.AcceptedTerms {
		account.AcceptedTermsAt = &now
		tVer := "v1.0"
		if input.TermsVersion != nil && strings.TrimSpace(*input.TermsVersion) != "" {
			tVer = strings.TrimSpace(*input.TermsVersion)
		}
		account.TermsVersion = &tVer
	}
	if input.AcceptedPrivacy != nil && *input.AcceptedPrivacy {
		account.AcceptedPrivacyAt = &now
		pVer := "v1.0"
		if input.PrivacyVersion != nil && strings.TrimSpace(*input.PrivacyVersion) != "" {
			pVer = strings.TrimSpace(*input.PrivacyVersion)
		}
		account.PrivacyVersion = &pVer
	}

	switch account.Role {
	case "owner":
		owner, err := s.repo.GetOwnerByAccountID(accountID)
		if err != nil {
			owner = &models.Owner{AccountID: accountID}
		}

		owner.KtpNumber = &cleanKtp
		owner.KtpPhotoUrl = &cleanPhoto
		owner.SelfiePhotoUrl = &cleanSelfie
		owner.KycStatus = newStatus
		owner.KycSubmittedAt = &now
		owner.KycVerifiedAt = nil
		owner.KycRejectionReason = nil

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

		ownerToSave := *owner
		if err := encryptOwnerKTP(&ownerToSave); err != nil {
			return errors.New("Gagal mengenkripsi data KTP")
		}

		if err := s.repo.SaveOwnerProfile(account, &ownerToSave); err != nil {
			return errors.New("Gagal menyimpan pengajuan verifikasi KYC")
		}

		// 🎁 Poin 8-1: Program Sambutan Mitra Baru saat Pemilik Toko selesai registrasi data diri / KYC
		if s.notifService != nil {
			go s.notifService.SendWelcomeMerchantNotification(accountID)
		}

	case "business_admin":
		admin, err := s.repo.GetBusinessAdminByAccountID(accountID)
		if err != nil {
			admin = &models.BusinessAdmin{AccountID: accountID}
		}

		admin.KtpNumber = &cleanKtp
		admin.KtpPhotoUrl = &cleanPhoto
		admin.SelfiePhotoUrl = &cleanSelfie
		admin.KycStatus = newStatus
		admin.KycSubmittedAt = &now
		admin.KycVerifiedAt = nil
		admin.KycRejectionReason = nil

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

		adminToSave := *admin
		if err := encryptBusinessAdminKTP(&adminToSave); err != nil {
			return errors.New("Gagal mengenkripsi data KTP")
		}

		if err := s.repo.SaveBusinessAdminProfile(account, &adminToSave); err != nil {
			return errors.New("Gagal menyimpan pengajuan verifikasi KYC")
		}
	}

	return nil
}

func (s *profileService) UpdateConsent(accountID string, input models.ConsentInput) (*models.Account, error) {
	input.Normalize()

	if input.AcceptedTerms == nil || !*input.AcceptedTerms {
		return nil, errors.New("Anda wajib menyetujui Syarat & Ketentuan untuk melanjutkan")
	}
	if input.AcceptedPrivacy == nil || !*input.AcceptedPrivacy {
		return nil, errors.New("Anda wajib menyetujui Kebijakan Privasi untuk melanjutkan")
	}

	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Akun tidak ditemukan")
		}
		return nil, err
	}

	now := time.Now()
	tVer := "v1.0"
	if input.TermsVersion != nil && strings.TrimSpace(*input.TermsVersion) != "" {
		tVer = strings.TrimSpace(*input.TermsVersion)
	}

	pVer := "v1.0"
	if input.PrivacyVersion != nil && strings.TrimSpace(*input.PrivacyVersion) != "" {
		pVer = strings.TrimSpace(*input.PrivacyVersion)
	}

	if err := s.repo.UpdateConsent(accountID, &now, &tVer, &now, &pVer); err != nil {
		return nil, errors.New("Gagal menyimpan persetujuan dokumen")
	}

	account.AcceptedTermsAt = &now
	account.TermsVersion = &tVer
	account.AcceptedPrivacyAt = &now
	account.PrivacyVersion = &pVer
	account.PasswordHash = nil

	return account, nil
}

