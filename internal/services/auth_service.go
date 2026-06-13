package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
	"toentoen-backend/pkg/utils"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(input models.RegisterUnifiedInput) error
	VerifyRegisterOTP(input models.VerifyRegisterOTPInput) error
	Login(input models.LoginUnifiedInput) error
	VerifyLoginOTP(input models.VerifyLoginOTPInput) (*models.Account, string, error)
	ForgotPassword(input models.ForgotPasswordInput) error
	VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error)
	ResetPassword(input models.ResetPasswordInput) error
	ChangePassword(input models.ChangePasswordInput) error
	CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error)
	SaveFCMToken(accountID string, input models.RegisterFCMTokenInput) error
}

type authService struct {
	repo repositories.AuthRepository
}

func NewAuthService(repo repositories.AuthRepository) AuthService {
	return &authService{repo}
}

func isEmail(identifier string) bool {
	return strings.Contains(identifier, "@")
}

func (s *authService) Register(input models.RegisterUnifiedInput) error {
	existingAccount, err := s.repo.FindByIdentifier(input.Identifier)
	
	if err == nil {
		switch existingAccount.AccountStatus {
		case "active":
			return errors.New("Akun dengan email/nomor ini sudah terdaftar dan aktif. Silakan langsung login.")
			
		case "inactive":
			otp := utils.GenerateNumericOTP(6)
			redisKey := fmt.Sprintf("register_otp:%s", input.Identifier)

			if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
				return errors.New("failed to generate new otp")
			}

			if isEmail(input.Identifier) && existingAccount.Email != nil {
				go utils.SendEmailOTP(*existingAccount.Email, otp)
			} else if existingAccount.PhoneNumber != nil {
				go utils.SendWAOTP(*existingAccount.PhoneNumber, otp)
			}
			
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

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("register_otp:%s", input.Identifier)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal membuat kode verifikasi")
	}

	if isEmail(input.Identifier) && account.Email != nil {
		go utils.SendEmailOTP(*account.Email, otp)
	} else if account.PhoneNumber != nil {
		go utils.SendWAOTP(*account.PhoneNumber, otp)
	}

	return nil
}

func (s *authService) VerifyRegisterOTP(input models.VerifyRegisterOTPInput) error {
	redisKey := fmt.Sprintf("register_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return errors.New("invalid or expired otp")
	}

	config.RDB.Del(config.Ctx, redisKey)

	err = s.repo.UpdateAccountStatus(input.Identifier, "active")
	if err != nil {
		return errors.New("failed to activate account")
	}

	return nil
}

func (s *authService) Login(input models.LoginUnifiedInput) error {
	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("account not found")
		}
		return err
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
		return errors.New("google account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.Password)); err != nil {
		return errors.New("invalid credentials")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("login_otp:%s", input.Identifier)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("failed to generate otp")
	}

	if isEmail(input.Identifier) && account.Email != nil {
		go utils.SendEmailOTP(*account.Email, otp)
	} else if account.PhoneNumber != nil {
		go utils.SendWAOTP(*account.PhoneNumber, otp)
	} else if account.Email != nil {
		go utils.SendEmailOTP(*account.Email, otp)
	}
	return nil
}

func (s *authService) VerifyLoginOTP(input models.VerifyLoginOTPInput) (*models.Account, string, error) {
	redisKey := fmt.Sprintf("login_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return nil, "", errors.New("invalid or expired otp")
	}

	config.RDB.Del(config.Ctx, redisKey)

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	account.LastLoginAt = &now
	s.repo.Update(account)
	s.repo.LoadAssociations(account)

	token, err := utils.GenerateToken(account.AccountID, account.Role)
	if err != nil {
		return nil, "", err
	}

	return account, token, nil
}

func (s *authService) ForgotPassword(input models.ForgotPasswordInput) error {
	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return errors.New("account not found")
	}

	if account.PasswordHash == nil {
		return errors.New("google account")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := fmt.Sprintf("forgot_otp:%s", input.Identifier)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return err
	}

	if isEmail(input.Identifier) && account.Email != nil {
		go utils.SendEmailOTP(*account.Email, otp)
	} else if account.PhoneNumber != nil {
		go utils.SendWAOTP(*account.PhoneNumber, otp)
	} else if account.Email != nil {
		go utils.SendEmailOTP(*account.Email, otp)
	}
	return nil
}

func (s *authService) VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error) {
	redisKey := fmt.Sprintf("forgot_otp:%s", input.Identifier)
	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()

	if err == redis.Nil || storedOTP != input.OTP {
		return "", errors.New("invalid or expired otp")
	}

	config.RDB.Del(config.Ctx, redisKey)

	resetToken := utils.GenerateNumericOTP(6)
	tokenRedisKey := fmt.Sprintf("reset_token:%s", input.Identifier)
	config.RDB.Set(config.Ctx, tokenRedisKey, resetToken, 15*time.Minute)

	return resetToken, nil
}

func (s *authService) ResetPassword(input models.ResetPasswordInput) error {
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("password mismatch")
	}

	tokenRedisKey := fmt.Sprintf("reset_token:%s", input.Identifier)
	storedToken, err := config.RDB.Get(config.Ctx, tokenRedisKey).Result()

	if err == redis.Nil || storedToken != input.ResetToken {
		return errors.New("invalid or expired reset token")
	}

	account, err := s.repo.FindByIdentifier(input.Identifier)
	if err != nil {
		return errors.New("account not found")
	}

	if account.PasswordHash != nil {
		err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.NewPassword))
		if err == nil {
			return errors.New("new password must be different from previous password")
		}
	}

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	pwd := string(passwordHash)
	account.PasswordHash = &pwd

	if err := s.repo.Update(account); err != nil {
		return err
	}

	config.RDB.Del(config.Ctx, tokenRedisKey)
	return nil
}

func (s *authService) ChangePassword(input models.ChangePasswordInput) error {
	account, err := s.repo.FindByIdentifier(input.Email)
	if err != nil {
		return errors.New("account not found")
	}

	if input.CurrentPassword == input.NewPassword {
		return errors.New("New password must be different from current password")
	}

	if account.PasswordHash == nil {
		return errors.New("google account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.CurrentPassword)); err != nil {
		return errors.New("wrong current password")
	}

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	pwd := string(passwordHash)
	account.PasswordHash = &pwd

	return s.repo.Update(account)
}

func (s *authService) CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	account.PasswordHash = nil

	return &account, nil
}

func (s *authService) SaveFCMToken(accountID string, input models.RegisterFCMTokenInput) error {
	return s.repo.SaveDeviceToken(accountID, input.Token, input.DeviceType)
}