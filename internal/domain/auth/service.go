package auth

import (
	"errors"
	"fmt"
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
	VerifyRegisterOTP(input models.VerifyRegisterOTPInput) error
	Login(input models.LoginUnifiedInput) error
	VerifyLoginOTP(input models.VerifyLoginOTPInput) (string, error)
	ForgotPassword(input models.ForgotPasswordInput) error
	VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error)
	ResetPassword(input models.ResetPasswordInput) error
	ChangePassword(accountID string, input models.ChangePasswordInput) error
	RequestEmailChange(accountID string, input models.RequestEmailChangeInput) error
	VerifyEmailChange(accountID string, input models.VerifyEmailChangeInput) error
	CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error)
}

type authService struct {
	repo AuthRepository
}

func NewAuthService(repo AuthRepository) AuthService {
	return &authService{repo}
}

func isEmail(identifier string) bool {
	return strings.Contains(identifier, "@")
}

func (s *authService) sendOTP(identifier string, email *string, phoneNumber *string, redisPrefix string) error {
	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("%s:%s", redisPrefix, identifier)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode verifikasi")
	}

	if isEmail(identifier) && email != nil {
		go utils.SendEmailOTP(*email, otp)
	} else if phoneNumber != nil {
		go utils.SendWAOTP(*phoneNumber, otp)
	}

	return nil
}

func (s *authService) Register(input models.RegisterUnifiedInput) error {
	existingAccount, err := s.repo.FindByIdentifier(input.Identifier)

	if err == nil {
		switch existingAccount.AccountStatus {
		case "active":
			return errors.New("Akun dengan email/nomor ini sudah terdaftar dan aktif. Silakan langsung login.")
		case "inactive":
			s.sendOTP(input.Identifier, existingAccount.Email, existingAccount.PhoneNumber, "register_otp")
			return errors.New("Akun sudah terdaftar namun belum aktif. Kami telah mengirimkan ulang kode verifikasi yang baru, silakan cek email/WA Anda.")
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
		Fullname:      "New Account",
		AuthProvider:  "local",
		AccountStatus: "inactive",
	}

	if isEmail(input.Identifier) {
		account.Email = &input.Identifier
	} else {
		account.PhoneNumber = &input.Identifier
	}

	if err := s.repo.CreateAccountWithProfile(&account, input.Role); err != nil {
		return errors.New("Gagal menyimpan data akun baru")
	}

	return s.sendOTP(input.Identifier, account.Email, account.PhoneNumber, "register_otp")
}

func (s *authService) VerifyRegisterOTP(input models.VerifyRegisterOTPInput) error {
	redisKey := fmt.Sprintf("register_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return errors.New("OTP tidak valid atau sudah kadaluarsa")
	}

	config.RDB.Del(config.Ctx, redisKey)

	if err := s.repo.UpdateAccountStatus(input.Identifier, "active"); err != nil {
		return errors.New("Gagal mengaktifkan akun")
	}

	return nil
}

func (s *authService) Login(input models.LoginUnifiedInput) error {
	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("Email atau password salah")
		}
		return errors.New("Terjadi kesalahan sistem")
	}

	switch account.AccountStatus {
	case "inactive":
		return errors.New("Akun anda belum aktif. Silakan cek email/WA Anda untuk OTP verifikasi.")
	case "suspended":
		return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	case "banned":
		return errors.New("Akun anda telah diblokir dari platform")
	}

	if account.PasswordHash == nil {
		return errors.New("Silakan login menggunakan Google")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.Password)); err != nil {
		return errors.New("Email atau password salah")
	}

	return s.sendOTP(input.Identifier, account.Email, account.PhoneNumber, "login_otp")
}

func (s *authService) VerifyLoginOTP(input models.VerifyLoginOTPInput) (string, error) {
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
	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return errors.New("Akun tidak ditemukan")
	}

	if account.PasswordHash == nil {
		return errors.New("Silakan reset password melalui layanan Google")
	}

	return s.sendOTP(input.Identifier, account.Email, account.PhoneNumber, "forgot_otp")
}

func (s *authService) VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error) {
	redisKey := fmt.Sprintf("forgot_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return "", errors.New("OTP tidak valid atau sudah kadaluarsa")
	}

	config.RDB.Del(config.Ctx, redisKey)

	resetToken := utils.GenerateNumericOTP(6)
	tokenRedisKey := fmt.Sprintf("reset_token:%s", input.Identifier)
	config.RDB.Set(config.Ctx, tokenRedisKey, resetToken, 15*time.Minute)

	return resetToken, nil
}

func (s *authService) ResetPassword(input models.ResetPasswordInput) error {
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("Kata sandi dan konfirmasi kata sandi tidak cocok")
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

	if account.PasswordHash != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.NewPassword)); err == nil {
			return errors.New("Kata sandi baru tidak boleh sama dengan kata sandi lama")
		}
	}

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	pwd := string(passwordHash)
	account.PasswordHash = &pwd

	if err := s.repo.Update(account); err != nil {
		return errors.New("Gagal memperbarui kata sandi")
	}

	config.RDB.Del(config.Ctx, tokenRedisKey)
	return nil
}

func (s *authService) ChangePassword(accountID string, input models.ChangePasswordInput) error {
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("Kata sandi baru dan konfirmasi kata sandi tidak cocok")
	}

	account, err := s.repo.FindByID(accountID)
	if err != nil {
		return errors.New("Sesi pengguna tidak valid")
	}

	if account.PasswordHash == nil {
		return errors.New("Silakan atur kata sandi melalui layanan Google")
	}

	if input.CurrentPassword == input.NewPassword {
		return errors.New("Kata sandi baru tidak boleh sama dengan kata sandi saat ini")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.CurrentPassword)); err != nil {
		return errors.New("Kata sandi saat ini salah")
	}

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	pwd := string(passwordHash)
	account.PasswordHash = &pwd

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

	if account.Email != nil && *account.Email != input.CurrentEmail {
		return errors.New("Email saat ini tidak cocok dengan data kami")
	}

	if input.CurrentEmail == input.NewEmail {
		return errors.New("Email baru tidak boleh sama dengan email saat ini")
	}

	_, err = s.repo.FindByIdentifier(input.NewEmail)
	if err == nil {
		return errors.New("Email baru sudah terdaftar pada akun lain")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("change_email_otp:%s", input.NewEmail)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode verifikasi")
	}

	go utils.SendEmailOTP(input.NewEmail, otp)

	return nil
}

func (s *authService) VerifyEmailChange(accountID string, input models.VerifyEmailChangeInput) error {
	redisKey := fmt.Sprintf("change_email_otp:%s", input.NewEmail)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return errors.New("OTP tidak valid atau sudah kadaluarsa")
	}

	if err := s.repo.UpdateEmail(accountID, input.NewEmail); err != nil {
		return errors.New("Gagal memperbarui email")
	}

	config.RDB.Del(config.Ctx, redisKey)
	return nil
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
		Fullname:      "Business Admin",
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