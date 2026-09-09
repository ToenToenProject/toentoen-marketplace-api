package auth

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(input models.RegisterUnifiedInput) error
	CheckReferralCode(code string) (*models.CheckReferralResponse, error)
	VerifyRegisterOTP(input models.VerifyRegisterOTPInput) error
	SendEmailOTP(email string) error
	VerifyEmailOTP(input models.VerifyEmailOTPInput) error
	SendPhoneOTP(phone string) error
	VerifyPhoneOTP(input models.VerifyPhoneOTPInput) error
	EvaluateAccountStatus(account *models.Account) string
	Login(input models.LoginUnifiedInput) error
	VerifyLoginOTP(input models.VerifyLoginOTPInput) (string, error)
	ForgotPassword(input models.ForgotPasswordInput) error
	VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error)
	ResetPassword(input models.ResetPasswordInput) error
	ChangePassword(accountID string, input models.ChangePasswordInput) error
	RequestEmailChange(accountID string, input models.RequestEmailChangeInput) error
	VerifyEmailChange(accountID string, input models.VerifyEmailChangeInput) error
	RequestPhoneChange(accountID string, input models.RequestPhoneChangeInput) error
	VerifyPhoneChange(accountID string, input models.VerifyPhoneChangeInput) error
	CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error)
}

type authService struct {
	repo AuthRepository
}

func NewAuthService(repo AuthRepository) AuthService {
	return &authService{repo}
}

func cleanIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if strings.Contains(identifier, "@") {
		return strings.ToLower(identifier)
	}
	// Bersihkan karakter spasi, tanda strip, atau simbol dari nomor telepon
	identifier = strings.ReplaceAll(identifier, " ", "")
	identifier = strings.ReplaceAll(identifier, "-", "")
	identifier = strings.ReplaceAll(identifier, "+", "")
	return identifier
}

func isEmail(identifier string) bool {
	addr, err := mail.ParseAddress(identifier)
	if err != nil {
		return false
	}
	parts := strings.Split(addr.Address, "@")
	return len(parts) == 2 && strings.Contains(parts[1], ".")
}

func (s *authService) sendOTP(identifier string, email *string, phoneNumber *string, redisPrefix string) error {
	if !isEmail(identifier) {
		return errors.New("Layanan verifikasi OTP melalui nomor telepon/WhatsApp sedang dalam tahap pengembangan. Silakan gunakan alamat email.")
	}

	if email == nil || strings.TrimSpace(*email) == "" {
		return errors.New("Alamat email tidak ditemukan untuk pengiriman kode verifikasi")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("%s:%s", redisPrefix, identifier)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode verifikasi pada sistem")
	}

	subject := "Kode Verifikasi ToenToen"
	switch redisPrefix {
	case "register_otp":
		subject = "Kode Verifikasi Registrasi Akun ToenToen"
	case "email_verify_otp":
		subject = "Kode Verifikasi Email ToenToen"
	case "login_otp":
		subject = "Kode Verifikasi Login ToenToen"
	case "forgot_otp":
		subject = "Kode Verifikasi Reset Password ToenToen"
	}

	if err := utils.SendEmailOTP(*email, otp, subject); err != nil {
		config.RDB.Del(config.Ctx, redisKey)
		return fmt.Errorf("Gagal mengirim kode verifikasi ke email: %w", err)
	}

	return nil
}

func (s *authService) Register(input models.RegisterUnifiedInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if !isEmail(input.Identifier) {
		return errors.New("Pendaftaran menggunakan nomor telepon belum tersedia saat ini karena layanan OTP nomor telepon sedang dalam pengembangan. Silakan daftar menggunakan email.")
	}

	var cleanRefCode *string
	if input.Role == "owner" && input.ReferralCode != nil && strings.TrimSpace(*input.ReferralCode) != "" {
		code := strings.TrimSpace(*input.ReferralCode)
		ref, err := s.repo.FindReferralCode(code)
		if err != nil || !ref.IsActive || ref.ReferralCode == nil || *ref.ReferralCode == "" {
			return errors.New("Kode referral tidak ditemukan")
		}
		cleanRefCode = ref.ReferralCode
	}

	existingAccount, err := s.repo.FindByIdentifier(input.Identifier)

	if err == nil {
		switch existingAccount.AccountStatus {
		case "active":
			return errors.New("Akun dengan email ini sudah terdaftar dan aktif. Silakan langsung login.")
		case "inactive":
			if err := s.sendOTP(input.Identifier, existingAccount.Email, existingAccount.PhoneNumber, "register_otp"); err != nil {
				return err
			}
			return errors.New("Akun sudah terdaftar namun belum aktif. Kami telah mengirimkan ulang kode verifikasi yang baru, silakan cek kotak masuk/spam email Anda.")
		default:
			return errors.New("Akun ini tidak dapat digunakan. Silakan hubungi layanan bantuan.")
		}
	} else if err != gorm.ErrRecordNotFound {
		return errors.New("Terjadi kesalahan pada server saat mengecek data akun")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Gagal memproses kata sandi")
	}
	pwd := string(passwordHash)

	account := models.Account{
		AccountID:     utils.GenerateUUID("ACC"),
		PasswordHash:  &pwd,
		Role:          input.Role,
		Username:      utils.GenerateRandomUsername(input.Identifier),
		FirstName:     "New",
		LastName:      "Account",
		AuthProvider:  "local",
		AccountStatus: "inactive",
		Email:         &input.Identifier,
	}

	if input.Role == "owner" && cleanRefCode != nil {
		account.Owner = &models.Owner{
			AccountID:    account.AccountID,
			ReferralCode: cleanRefCode,
		}
	}

	if err := s.repo.CreateAccountWithProfile(&account, input.Role); err != nil {
		return errors.New("Gagal menyimpan data akun baru")
	}

	return s.sendOTP(input.Identifier, account.Email, account.PhoneNumber, "register_otp")
}

func (s *authService) CheckReferralCode(code string) (*models.CheckReferralResponse, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, errors.New("Kode referral tidak boleh kosong")
	}

	ref, err := s.repo.FindReferralCode(cleanCode)
	if err != nil || !ref.IsActive || ref.ReferralCode == nil || *ref.ReferralCode == "" {
		return nil, errors.New("Kode referral tidak ditemukan")
	}

	return &models.CheckReferralResponse{
		Code:    *ref.ReferralCode,
		IsValid: true,
		Message: "Kode referral valid dan tersedia",
	}, nil
}

func (s *authService) EvaluateAccountStatus(account *models.Account) string {
	if account == nil {
		return "inactive"
	}
	if account.AccountStatus == "banned" || account.AccountStatus == "suspended" {
		return account.AccountStatus
	}

	if account.EmailVerifiedAt != nil {
		return "active"
	}
	return "inactive"
}

func (s *authService) VerifyRegisterOTP(input models.VerifyRegisterOTPInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	redisKey := fmt.Sprintf("register_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return errors.New("OTP tidak valid atau sudah kadaluarsa")
	}

	config.RDB.Del(config.Ctx, redisKey)

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	if account.AccountStatus == "banned" {
		return errors.New("Akun anda telah diblokir dari platform")
	}
	if account.AccountStatus == "suspended" {
		return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	}

	now := time.Now()
	account.EmailVerifiedAt = &now
	newStatus := s.EvaluateAccountStatus(account)

	if err := s.repo.UpdateEmailVerified(input.Identifier, now, newStatus); err != nil {
		return errors.New("Gagal mengaktifkan akun")
	}

	return nil
}

func (s *authService) SendEmailOTP(email string) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if !isEmail(cleanEmail) {
		return errors.New("Format alamat email tidak valid")
	}

	account, err := s.repo.FindByIdentifier(cleanEmail)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("Akun dengan email ini tidak ditemukan")
		}
		return errors.New("Terjadi kesalahan sistem saat mencari data akun")
	}

	if account.AccountStatus == "banned" {
		return errors.New("Akun anda telah diblokir dari platform")
	}
	if account.AccountStatus == "suspended" {
		return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	}

	return s.sendOTP(cleanEmail, account.Email, account.PhoneNumber, "email_verify_otp")
}

func (s *authService) VerifyEmailOTP(input models.VerifyEmailOTPInput) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(input.Email))
	if !isEmail(cleanEmail) {
		return errors.New("Format alamat email tidak valid")
	}

	redisKey := fmt.Sprintf("email_verify_otp:%s", cleanEmail)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()
	if err == redis.Nil || storedOTP != input.OTP {
		// Fallback check register_otp jikalau kode OTP diminta dari registrasi
		regKey := fmt.Sprintf("register_otp:%s", cleanEmail)
		storedRegOTP, regErr := config.RDB.Get(config.Ctx, regKey).Result()
		if regErr == redis.Nil || storedRegOTP != input.OTP {
			return errors.New("OTP tidak valid atau sudah kadaluarsa")
		}
		config.RDB.Del(config.Ctx, regKey)
	} else {
		config.RDB.Del(config.Ctx, redisKey)
	}

	account, err := s.repo.FindByIdentifier(cleanEmail)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	if account.AccountStatus == "banned" {
		return errors.New("Akun anda telah diblokir dari platform")
	}
	if account.AccountStatus == "suspended" {
		return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	}

	now := time.Now()
	account.EmailVerifiedAt = &now
	newStatus := s.EvaluateAccountStatus(account)

	if err := s.repo.UpdateEmailVerified(cleanEmail, now, newStatus); err != nil {
		return errors.New("Gagal memverifikasi email")
	}

	return nil
}

func (s *authService) SendPhoneOTP(phone string) error {
	return errors.New("Layanan verifikasi OTP melalui nomor telepon/WhatsApp sedang dalam tahap pengembangan. Silakan gunakan alamat email.")
}

func (s *authService) VerifyPhoneOTP(input models.VerifyPhoneOTPInput) error {
	return errors.New("Layanan verifikasi OTP melalui nomor telepon/WhatsApp sedang dalam tahap pengembangan.")
}

func (s *authService) Login(input models.LoginUnifiedInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if !isEmail(input.Identifier) {
		return errors.New("Login menggunakan nomor telepon belum tersedia saat ini karena layanan verifikasi nomor telepon sedang dalam pengembangan. Silakan login menggunakan email.")
	}

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("Email atau password salah")
		}
		return errors.New("Terjadi kesalahan sistem")
	}

	switch account.AccountStatus {
	case "inactive":
		return errors.New("Akun anda belum aktif. Silakan cek email Anda untuk OTP verifikasi.")
	case "suspended":
		return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	case "banned":
		return errors.New("Akun anda telah diblokir dari platform")
	}

	if account.PasswordHash == nil {
		return errors.New("Akun ini tidak memiliki kata sandi, silakan gunakan fitur Lupa Password untuk membuat sandi baru.")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.Password)); err != nil {
		return errors.New("Email atau password salah")
	}

	return s.sendOTP(input.Identifier, account.Email, account.PhoneNumber, "login_otp")
}

func (s *authService) VerifyLoginOTP(input models.VerifyLoginOTPInput) (string, error) {
	input.Identifier = cleanIdentifier(input.Identifier)
	redisKey := fmt.Sprintf("login_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return "", errors.New("OTP tidak valid atau sudah kadaluarsa")
	}

	config.RDB.Del(config.Ctx, redisKey)

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return "", errors.New("Gagal memuat data akun")
	}

	now := time.Now()
	account.LastLoginAt = &now
	s.repo.Update(account)

	token, err := utils.GenerateToken(account.AccountID, account.Role)
	if err != nil {
		return "", errors.New("Gagal membuat sesi pengguna")
	}

	return token, nil
}

func (s *authService) ForgotPassword(input models.ForgotPasswordInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if !isEmail(input.Identifier) {
		return errors.New("Layanan reset kata sandi menggunakan nomor telepon sedang dalam pengembangan. Silakan gunakan email terdaftar Anda.")
	}

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	// Kirim OTP verifikasi (berlaku untuk semua akun lokal maupun akun OAuth yang ingin membuat kata sandi baru)
	return s.sendOTP(input.Identifier, account.Email, account.PhoneNumber, "forgot_otp")
}

func (s *authService) VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error) {
	input.Identifier = cleanIdentifier(input.Identifier)
	redisKey := fmt.Sprintf("forgot_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil {
		return "", errors.New("Kode OTP telah kadaluarsa atau tidak ditemukan")
	}

	if storedOTP != input.OTP {
		attemptKey := fmt.Sprintf("forgot_otp_attempts:%s", input.Identifier)
		attempts, _ := config.RDB.Incr(config.Ctx, attemptKey).Result()
		config.RDB.Expire(config.Ctx, attemptKey, 5*time.Minute)
		if attempts >= 5 {
			config.RDB.Del(config.Ctx, redisKey)
			config.RDB.Del(config.Ctx, attemptKey)
			return "", errors.New("Terlalu banyak percobaan kode OTP yang salah. Silakan minta kode OTP baru.")
		}
		return "", errors.New("Kode OTP tidak valid")
	}

	config.RDB.Del(config.Ctx, redisKey)
	config.RDB.Del(config.Ctx, fmt.Sprintf("forgot_otp_attempts:%s", input.Identifier))

	// Generate secure UUID reset token
	resetToken := utils.GenerateUUID("RST")
	tokenRedisKey := fmt.Sprintf("reset_token:%s", input.Identifier)
	config.RDB.Set(config.Ctx, tokenRedisKey, resetToken, 10*time.Minute)

	return resetToken, nil
}

func (s *authService) ResetPassword(input models.ResetPasswordInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("Kata sandi baru dan konfirmasi kata sandi tidak cocok")
	}

	tokenRedisKey := fmt.Sprintf("reset_token:%s", input.Identifier)
	storedToken, err := config.RDB.Get(config.Ctx, tokenRedisKey).Result()

	if err == redis.Nil || storedToken != input.ResetToken {
		return errors.New("Sesi reset password tidak valid atau sudah kadaluarsa")
	}

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	if account.PasswordHash != nil && *account.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.NewPassword)); err == nil {
			return errors.New("Kata sandi baru tidak boleh sama dengan kata sandi lama")
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Gagal memproses kata sandi")
	}
	pwd := string(passwordHash)
	account.PasswordHash = &pwd
	now := time.Now()
	account.LastLoginAt = &now

	// Jika akun sebelumnya inactive, verifikasi OTP via reset password membuktikan kepemilikan akun -> aktifkan
	if account.AccountStatus == "inactive" {
		account.AccountStatus = "active"
	}

	if err := s.repo.Update(account); err != nil {
		return errors.New("Gagal memperbarui kata sandi")
	}

	config.RDB.Del(config.Ctx, tokenRedisKey)
	return nil
}

func (s *authService) ChangePassword(accountID string, input models.ChangePasswordInput) error {
	if len(strings.TrimSpace(input.NewPassword)) < 6 {
		return errors.New("Kata sandi baru minimal 6 karakter")
	}
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("Kata sandi baru dan konfirmasi kata sandi tidak cocok")
	}

	account, err := s.repo.FindByID(accountID)
	if err != nil {
		return errors.New("Sesi pengguna tidak valid")
	}

	// Jika akun sudah memiliki kata sandi sebelumnya, wajib verifikasi CurrentPassword
	if account.PasswordHash != nil && *account.PasswordHash != "" {
		if strings.TrimSpace(input.CurrentPassword) == "" {
			return errors.New("Kata sandi saat ini wajib diisi")
		}
		if input.CurrentPassword == input.NewPassword {
			return errors.New("Kata sandi baru tidak boleh sama dengan kata sandi saat ini")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.CurrentPassword)); err != nil {
			return errors.New("Kata sandi saat ini salah")
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Gagal memproses kata sandi baru")
	}
	pwd := string(passwordHash)
	account.PasswordHash = &pwd
	now := time.Now()
	account.LastLoginAt = &now

	if err := s.repo.Update(account); err != nil {
		return errors.New("Gagal memperbarui kata sandi")
	}
	return nil
}

func (s *authService) RequestEmailChange(accountID string, input models.RequestEmailChangeInput) error {
	account, err := s.repo.FindByID(accountID)
	if err != nil {
		return errors.New("Sesi pengguna tidak valid")
	}

	currentEmailClean := strings.ToLower(strings.TrimSpace(input.CurrentEmail))
	newEmailClean := strings.ToLower(strings.TrimSpace(input.NewEmail))

	if account.Email != nil && strings.ToLower(strings.TrimSpace(*account.Email)) != currentEmailClean {
		return errors.New("Email saat ini tidak cocok dengan data akun Anda")
	}

	if currentEmailClean == newEmailClean {
		return errors.New("Email baru tidak boleh sama dengan email saat ini")
	}

	if exists, err := s.repo.FindByIdentifier(newEmailClean); err == nil && exists != nil && exists.AccountID != accountID {
		return errors.New("Email baru sudah terdaftar pada akun lain")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("change_email_otp:%s", newEmailClean)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode verifikasi")
	}

	if err := utils.SendEmailOTP(newEmailClean, otp, "Kode Verifikasi Perubahan Email ToenToen"); err != nil {
		config.RDB.Del(config.Ctx, redisKey)
		return fmt.Errorf("Gagal mengirim kode verifikasi ke email baru: %w", err)
	}

	return nil
}

func (s *authService) VerifyEmailChange(accountID string, input models.VerifyEmailChangeInput) error {
	newEmailClean := strings.ToLower(strings.TrimSpace(input.NewEmail))
	if !isEmail(newEmailClean) {
		return errors.New("Format alamat email tidak valid")
	}

	account, err := s.repo.FindByID(accountID)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}
	if account.AccountStatus == "banned" {
		return errors.New("Akun anda telah diblokir dari platform")
	}
	if account.AccountStatus == "suspended" {
		return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	}

	redisKey := fmt.Sprintf("change_email_otp:%s", newEmailClean)
	attemptKey := fmt.Sprintf("change_email_attempts:%s", newEmailClean)

	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()
	if err == redis.Nil || storedOTP != input.OTP {
		attempts, _ := config.RDB.Incr(config.Ctx, attemptKey).Result()
		if attempts == 1 {
			config.RDB.Expire(config.Ctx, attemptKey, 5*time.Minute)
		}
		if attempts >= 5 {
			config.RDB.Del(config.Ctx, redisKey)
			config.RDB.Del(config.Ctx, attemptKey)
			return errors.New("Terlalu banyak percobaan kode OTP yang salah. Silakan minta kode OTP baru.")
		}
		return errors.New("OTP tidak valid atau sudah kadaluarsa")
	}

	if err := s.repo.UpdateEmail(accountID, newEmailClean); err != nil {
		return errors.New("Gagal memperbarui email")
	}

	config.RDB.Del(config.Ctx, redisKey)
	config.RDB.Del(config.Ctx, attemptKey)
	return nil
}

func (s *authService) RequestPhoneChange(accountID string, input models.RequestPhoneChangeInput) error {
	return errors.New("Layanan verifikasi nomor telepon/WhatsApp sedang dalam tahap pengembangan. Fitur ubah nomor telepon belum dapat digunakan saat ini.")
}

func (s *authService) VerifyPhoneChange(accountID string, input models.VerifyPhoneChangeInput) error {
	return errors.New("Layanan verifikasi nomor telepon/WhatsApp sedang dalam tahap pengembangan.")
}

func (s *authService) CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("Gagal memproses kata sandi")
	}
	pwd := string(passwordHash)

	account := models.Account{
		AccountID:     utils.GenerateUUID("ACC"),
		PasswordHash:  &pwd,
		Role:          "business_admin",
		Username:      utils.GenerateRandomUsername(input.Identifier),
		FirstName:     "Business",
		LastName:      "Admin",
		AuthProvider:  "local",
		AccountStatus: "active",
	}

	if isEmail(input.Identifier) {
		account.Email = &input.Identifier
	} else {
		account.PhoneNumber = &input.Identifier
	}

	if err := s.repo.CreateAccountWithProfile(&account, "business_admin", ownerID); err != nil {
		return nil, errors.New("Gagal membuat akun Business Admin")
	}

	account.PasswordHash = nil
	return &account, nil
}
