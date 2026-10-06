package auth

import (
	"encoding/json"
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
	VerifyRegisterOTP(input models.VerifyRegisterOTPInput) (string, error)
	SendEmailOTP(email string, role string) error
	VerifyEmailOTP(input models.VerifyEmailOTPInput) error
	SendPhoneOTP(phone string) error
	VerifyPhoneOTP(input models.VerifyPhoneOTPInput) error
	EvaluateAccountStatus(account *models.Account) string
	Login(input models.LoginUnifiedInput) error
	VerifyLoginOTP(input models.VerifyLoginOTPInput) (string, *models.Account, error)
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

// roleGroupFromInput mengubah role dari request (termasuk alias FE) menjadi kelompok role.
// Mengembalikan string kosong jika role tidak dikirim / tidak dikenal.
func roleGroupFromInput(role string) string {
	return models.RoleGroupOf(NormalizeExpectedRole(role))
}

// candidateGroups mengembalikan kelompok yang perlu diperiksa: kelompok dari role jika dikirim,
// atau kedua kelompok jika role kosong.
func candidateGroups(role string) []string {
	if g := roleGroupFromInput(role); g != "" {
		return []string{g}
	}
	return []string{models.RoleGroupCustomer, models.RoleGroupMerchant}
}

// scopedKey membuat key Redis yang dipisah per kelompok role, contoh: login_otp:customer:user@mail.com
func scopedKey(prefix, group, identifier string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, group, identifier)
}

// findAccountInGroup mencari akun secara ketat di kelompok role yang diminta (tanpa fallback
// ke kelompok lain). Dipakai untuk alur sensitif seperti reset password dan verifikasi OTP.
func (s *authService) findAccountInGroup(identifier, role string) (*models.Account, error) {
	return s.repo.FindByIdentifier(identifier, roleGroupFromInput(role))
}

// findAccountForLogin mencari akun di kelompok role yang diminta. Jika tidak ada namun identifier
// terdaftar di kelompok lain, akun tersebut dikembalikan agar pemanggil dapat memberi
// RoleMismatchError (arahkan user ke portal yang benar). Akun lain tidak pernah diberi sesi.
func (s *authService) findAccountForLogin(identifier, targetRole string) (*models.Account, error) {
	group := models.RoleGroupOf(targetRole)
	account, err := s.repo.FindByIdentifier(identifier, group)
	if errors.Is(err, gorm.ErrRecordNotFound) && group != "" {
		return s.repo.FindByIdentifier(identifier, "")
	}
	return account, err
}

func (s *authService) sendOTP(identifier string, group string, email *string, phoneNumber *string, redisPrefix string, customSubject ...string) error {
	var cleanEmail string
	if email != nil {
		cleanEmail = strings.ToLower(strings.TrimSpace(*email))
	}
	var cleanPhone string
	if phoneNumber != nil {
		cleanPhone = cleanIdentifier(*phoneNumber)
	} else if !isEmail(identifier) {
		cleanPhone = cleanIdentifier(identifier)
	}

	hasEmail := cleanEmail != "" && isEmail(cleanEmail)
	hasPhone := cleanPhone != ""

	if !hasEmail && !hasPhone {
		return errors.New("Tidak ditemukan alamat email atau nomor telepon untuk pengiriman kode verifikasi")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := scopedKey(redisPrefix, group, identifier)

	if config.RDB != nil {
		if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
			return errors.New("Gagal memproses kode verifikasi pada sistem")
		}
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

	if len(customSubject) > 0 && strings.TrimSpace(customSubject[0]) != "" {
		subject = strings.TrimSpace(customSubject[0])
	}

	var waMessage string
	subjLower := strings.ToLower(subject)
	if strings.Contains(subjLower, "aktivasi") && strings.Contains(subjLower, "admin") {
		waMessage = fmt.Sprintf("Halo Rekan Admin!\n\nAnda telah didaftarkan sebagai Admin Toko di ToenToen. Gunakan kode verifikasi (OTP) berikut untuk membuat kata sandi baru dan mengaktifkan akun Anda:\n\n%s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun demi keamanan toko.", otp)
	} else if strings.Contains(subjLower, "admin") && strings.Contains(subjLower, "registrasi") {
		waMessage = fmt.Sprintf("Halo Rekan Admin!\n\nKode verifikasi registrasi Admin Toko ToenToen Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit.", otp)
	} else if strings.Contains(subjLower, "admin") && (strings.Contains(subjLower, "reset") || strings.Contains(subjLower, "kata sandi")) {
		waMessage = fmt.Sprintf("Halo Rekan Admin!\n\nKode verifikasi reset kata sandi Admin Toko ToenToen Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit.", otp)
	} else if strings.Contains(subjLower, "admin") && (strings.Contains(subjLower, "login") || strings.Contains(subjLower, "masuk")) {
		waMessage = fmt.Sprintf("Halo Rekan Admin!\n\nKode verifikasi login Admin Toko ToenToen Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit.", otp)
	} else if strings.Contains(subjLower, "admin") && strings.Contains(subjLower, "email") {
		waMessage = fmt.Sprintf("Halo Rekan Admin!\n\nKode verifikasi email Admin Toko ToenToen Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit.", otp)
	} else if strings.Contains(subjLower, "registrasi") {
		waMessage = fmt.Sprintf("Selamat Datang di ToenToen!\n\nKode verifikasi registrasi akun Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit.", otp)
	} else if strings.Contains(subjLower, "login") || strings.Contains(subjLower, "masuk") {
		waMessage = fmt.Sprintf("Kode verifikasi login akun ToenToen Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.", otp)
	} else if strings.Contains(subjLower, "reset") || strings.Contains(subjLower, "kata sandi") {
		waMessage = fmt.Sprintf("Kode verifikasi reset kata sandi akun ToenToen Anda adalah:\n\n%s\n\nKode ini hanya berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.", otp)
	} else {
		waMessage = fmt.Sprintf("Kode verifikasi (OTP) ToenToen Anda adalah: %s\n\nKode ini berlaku selama 5 menit. Jangan berikan kode ini kepada siapapun.", otp)
	}

	var sent bool
	var lastErr error

	if hasEmail {
		if err := utils.SendEmailOTP(cleanEmail, otp, subject); err != nil {
			lastErr = err
		} else {
			sent = true
		}
	}

	if hasPhone {
		if err := utils.SendWAOTP(cleanPhone, otp, waMessage); err != nil {
			if !sent {
				lastErr = err
			}
		} else {
			sent = true
		}
	}

	if !sent {
		if config.RDB != nil {
			config.RDB.Del(config.Ctx, redisKey)
		}
		if lastErr != nil {
			return fmt.Errorf("Gagal mengirim kode verifikasi: %w", lastErr)
		}
		return errors.New("Gagal mengirim kode verifikasi")
	}

	return nil
}

func (s *authService) Register(input models.RegisterUnifiedInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)

	// Normalisasi role (alias FE) dan tentukan kelompok role.
	// customer  -> kelompok customer
	// owner / business_admin -> kelompok merchant (tidak boleh berbagi email/telepon)
	input.Role = NormalizeExpectedRole(input.Role)
	group := models.RoleGroupOf(input.Role)
	if group == "" {
		return errors.New("Role tidak valid. Gunakan customer, owner, atau business_admin")
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

	// Cek duplikasi hanya di dalam kelompok role yang sama.
	existingAccount, err := s.repo.FindByIdentifier(input.Identifier, group)
	if err == nil {
		if group == models.RoleGroupMerchant && existingAccount.Role != input.Role {
			if existingAccount.Role == "business_admin" {
				return errors.New("Email/nomor ini sudah digunakan sebagai akun Admin Toko. Akun Owner dan Admin Toko tidak boleh menggunakan email/nomor yang sama.")
			}
			return errors.New("Email/nomor ini sudah digunakan sebagai akun Pemilik Toko (Owner). Akun Owner dan Admin Toko tidak boleh menggunakan email/nomor yang sama.")
		}
		switch existingAccount.AccountStatus {
		case "active":
			if isEmail(input.Identifier) {
				return errors.New("Akun dengan email ini sudah terdaftar dan aktif. Silakan langsung login.")
			}
			return errors.New("Akun dengan nomor ini sudah terdaftar dan aktif. Silakan langsung login.")
		case "inactive":
			customSubj := "Kode Verifikasi Registrasi Akun ToenToen"
			if existingAccount.Role == "business_admin" {
				customSubj = "Kode Verifikasi Aktivasi Akun Admin Toko ToenToen"
			}
			if err := s.sendOTP(input.Identifier, group, existingAccount.Email, existingAccount.PhoneNumber, "register_otp", customSubj); err != nil {
				return err
			}
			if isEmail(input.Identifier) {
				return errors.New("Akun sudah terdaftar namun belum aktif. Kami telah mengirimkan ulang kode verifikasi yang baru, silakan cek kotak masuk/spam email Anda.")
			}
			return errors.New("Akun sudah terdaftar namun belum aktif. Kami telah mengirimkan ulang kode verifikasi yang baru ke nomor WhatsApp Anda.")
		default:
			return errors.New("Akun ini tidak dapat digunakan. Silakan hubungi layanan bantuan.")
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("Terjadi kesalahan pada server saat mengecek data akun")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Gagal memproses kata sandi")
	}
	pwd := string(passwordHash)

	otp := utils.GenerateNumericOTP(6)

	pendingData := models.PendingRegistration{
		Identifier:   input.Identifier,
		PasswordHash: pwd,
		Role:         input.Role,
		ReferralCode: cleanRefCode,
		OTP:          otp,
	}

	pendingJSON, err := json.Marshal(pendingData)
	if err != nil {
		return errors.New("Gagal memproses data pendaftaran sementara")
	}

	pendingKey := scopedKey("register_pending", group, input.Identifier)
	otpKey := scopedKey("register_otp", group, input.Identifier)

	// Simpan data pendaftaran sementara dan kode OTP ke Redis (TTL 5 menit)
	if config.RDB != nil {
		if err := config.RDB.Set(config.Ctx, pendingKey, string(pendingJSON), 5*time.Minute).Err(); err != nil {
			return errors.New("Gagal menyimpan data verifikasi ke sistem")
		}
		_ = config.RDB.Set(config.Ctx, otpKey, otp, 5*time.Minute).Err()
	}

	if isEmail(input.Identifier) {
		// Kirim kode verifikasi ke email
		subject := "Kode Verifikasi Registrasi Akun ToenToen"
		if input.Role == "business_admin" {
			subject = "Kode Verifikasi Registrasi Admin Toko ToenToen"
		}
		if err := utils.SendEmailOTP(input.Identifier, otp, subject); err != nil {
			if config.RDB != nil {
				config.RDB.Del(config.Ctx, pendingKey)
				config.RDB.Del(config.Ctx, otpKey)
			}
			return fmt.Errorf("Gagal mengirim kode verifikasi ke email: %w", err)
		}
	} else {
		// Kirim kode verifikasi ke WhatsApp
		waMsg := fmt.Sprintf("Selamat Datang di ToenToen!\n\nKode verifikasi registrasi akun Anda adalah:\n\n%s\n\nKode ini berlaku selama 5 menit.", otp)
		if input.Role == "business_admin" {
			waMsg = fmt.Sprintf("Halo Rekan Admin!\n\nKode verifikasi registrasi Admin Toko ToenToen Anda adalah:\n\n%s\n\nKode ini berlaku selama 5 menit.", otp)
		}
		if err := utils.SendWAOTP(input.Identifier, otp, waMsg); err != nil {
			if config.RDB != nil {
				config.RDB.Del(config.Ctx, pendingKey)
				config.RDB.Del(config.Ctx, otpKey)
			}
			return fmt.Errorf("Gagal mengirim kode verifikasi ke WhatsApp: %w", err)
		}
	}

	return nil
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

func (s *authService) VerifyRegisterOTP(input models.VerifyRegisterOTPInput) (string, error) {
	input.Identifier = cleanIdentifier(input.Identifier)
	groups := candidateGroups(input.Role)

	if config.RDB != nil {
		// Cari data pendaftaran sementara di setiap kelompok role yang relevan.
		// Kelompok yang dipakai adalah yang kode OTP-nya cocok.
		var pending *models.PendingRegistration
		var group string
		hasAnyPending := false
		for _, g := range groups {
			storedJSON, err := config.RDB.Get(config.Ctx, scopedKey("register_pending", g, input.Identifier)).Result()
			if err != nil || storedJSON == "" {
				continue
			}
			hasAnyPending = true
			var p models.PendingRegistration
			if err := json.Unmarshal([]byte(storedJSON), &p); err != nil {
				return "", errors.New("Gagal membaca data pendaftaran sementara")
			}
			if p.OTP != input.OTP {
				continue
			}
			if pending != nil {
				return "", ErrAmbiguousAccount
			}
			matched := p
			pending = &matched
			group = g
		}
		if pending == nil && hasAnyPending {
			return "", errors.New("OTP tidak valid atau sudah kadaluarsa")
		}

		if pending != nil {
			pendingKey := scopedKey("register_pending", group, input.Identifier)
			otpKey := scopedKey("register_otp", group, input.Identifier)

			// Idempotency check: pastikan identifier belum didaftarkan di kelompok role yang sama
			existingAccount, errFind := s.repo.FindByIdentifier(input.Identifier, group)
			if errFind == nil && existingAccount != nil {
				config.RDB.Del(config.Ctx, pendingKey)
				config.RDB.Del(config.Ctx, otpKey)
				if existingAccount.AccountStatus == "active" {
					return "", errors.New("Akun sudah terdaftar dan aktif. Silakan langsung login.")
				}
			}

			now := time.Now()
			account := models.Account{
				AccountID:    utils.GenerateUUID("ACC"),
				PasswordHash: &pending.PasswordHash,
				Role:         pending.Role,
				Username:     utils.GenerateRandomUsername(pending.Identifier),
				FirstName:    "New",
				LastName:     "Account",
				AuthProvider: "local",
				LastLoginAt:  &now, // Sesi login langsung tercatat aktif
			}

			if isEmail(pending.Identifier) {
				account.Email = &pending.Identifier
				account.EmailVerifiedAt = &now
				account.AccountStatus = "active" // Akun langsung aktif saat OTP email berhasil diverifikasi
			} else {
				account.PhoneNumber = &pending.Identifier
				account.PhoneVerifiedAt = &now
				account.AccountStatus = "inactive" // Akun nomor WA berstatus belum terverifikasi jika belum diverifikasi emailnya
			}

			if pending.Role == "owner" && pending.ReferralCode != nil && *pending.ReferralCode != "" {
				account.Owner = &models.Owner{
					AccountID:    account.AccountID,
					ReferralCode: pending.ReferralCode,
				}
			}

			// Simpan akun dan profil ke database PostgreSQL secara atomik
			if err := s.repo.CreateAccountWithProfile(&account, pending.Role); err != nil {
				return "", errors.New("Gagal membuat akun baru di database")
			}

			// Bersihkan kunci sementara dari Redis
			config.RDB.Del(config.Ctx, pendingKey)
			config.RDB.Del(config.Ctx, otpKey)

			// Terbitkan sesi login otomatis
			token, err := utils.GenerateToken(account.AccountID, account.Role)
			if err != nil {
				return "", errors.New("Akun berhasil dibuat, namun gagal membuat sesi login otomatis")
			}

			return token, nil
		}
	}

	// Alur Fallback: Jika data pending tidak ada di Redis, cari register_otp warisan atau scoped
	var matchedGroup string
	var account *models.Account

	if config.RDB != nil {
		for _, g := range groups {
			k := scopedKey("register_otp", g, input.Identifier)
			storedOTP, errOTP := config.RDB.Get(config.Ctx, k).Result()
			if errOTP == nil && storedOTP == input.OTP {
				matchedGroup = g
				config.RDB.Del(config.Ctx, k)
				break
			}
		}
		// Cek key un-scoped legacy
		if matchedGroup == "" {
			legacyKey := fmt.Sprintf("register_otp:%s", input.Identifier)
			storedOTP, errOTP := config.RDB.Get(config.Ctx, legacyKey).Result()
			if errOTP == nil && storedOTP == input.OTP {
				config.RDB.Del(config.Ctx, legacyKey)
			} else {
				return "", errors.New("OTP tidak valid atau sudah kadaluarsa. Silakan lakukan pendaftaran ulang.")
			}
		}
	}

	acc, err := s.repo.FindByIdentifier(input.Identifier, matchedGroup)
	if err != nil {
		return "", errors.New("Data pendaftaran tidak ditemukan atau sudah kadaluarsa. Silakan lakukan pendaftaran ulang.")
	}
	account = acc

	if account.AccountStatus == "banned" {
		return "", errors.New("Akun anda telah diblokir dari platform")
	}
	if account.AccountStatus == "suspended" {
		return "", errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	}

	now := time.Now()
	if isEmail(input.Identifier) {
		account.EmailVerifiedAt = &now
	} else {
		account.PhoneVerifiedAt = &now
	}
	account.LastLoginAt = &now
	newStatus := s.EvaluateAccountStatus(account)

	if isEmail(input.Identifier) {
		if err := s.repo.UpdateEmailVerified(account.AccountID, now, newStatus); err != nil {
			return "", errors.New("Gagal mengaktifkan akun")
		}
	} else {
		if err := s.repo.UpdatePhoneVerified(account.AccountID, now, newStatus); err != nil {
			return "", errors.New("Gagal mengaktifkan akun")
		}
	}
	_ = s.repo.Update(account)

	token, err := utils.GenerateToken(account.AccountID, account.Role)
	if err != nil {
		return "", errors.New("Akun berhasil diaktifkan, namun gagal membuat sesi login otomatis")
	}

	return token, nil
}

func (s *authService) SendEmailOTP(email string, role string) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if !isEmail(cleanEmail) {
		return errors.New("Format alamat email tidak valid")
	}

	groups := candidateGroups(role)
	account, err := s.repo.FindByIdentifier(cleanEmail, roleGroupFromInput(role))
	if err == nil {
		if account.AccountStatus == "banned" {
			return errors.New("Akun anda telah diblokir dari platform")
		}
		if account.AccountStatus == "suspended" {
			return errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
		}
		group := models.RoleGroupOf(account.Role)
		customSubj := "Kode Verifikasi Email ToenToen"
		if account.Role == "business_admin" {
			if account.LastLoginAt == nil || account.PasswordHash == nil || account.AccountStatus == "inactive" {
				customSubj = "Kode Verifikasi Aktivasi Akun Admin Toko ToenToen"
			} else {
				customSubj = "Kode Verifikasi Email Admin Toko ToenToen"
			}
		}
		return s.sendOTP(cleanEmail, group, account.Email, account.PhoneNumber, "email_verify_otp", customSubj)
	} else if errors.Is(err, ErrAmbiguousAccount) {
		return ErrAmbiguousAccount
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("Terjadi kesalahan sistem saat mencari data akun")
	}

	// Jika akun belum ada di database, cek apakah sedang dalam status pending registration di Redis
	if config.RDB != nil {
		for _, g := range groups {
			pendingKey := scopedKey("register_pending", g, cleanEmail)
			storedJSON, errRedis := config.RDB.Get(config.Ctx, pendingKey).Result()
			if errRedis == nil && storedJSON != "" {
				var pending models.PendingRegistration
				if err := json.Unmarshal([]byte(storedJSON), &pending); err == nil {
					newOTP := utils.GenerateNumericOTP(6)
					pending.OTP = newOTP
					updatedJSON, _ := json.Marshal(pending)
					_ = config.RDB.Set(config.Ctx, pendingKey, string(updatedJSON), 5*time.Minute).Err()
					_ = config.RDB.Set(config.Ctx, scopedKey("register_otp", g, cleanEmail), newOTP, 5*time.Minute).Err()
					subject := "Kode Verifikasi Registrasi Akun ToenToen"
					if pending.Role == "business_admin" {
						subject = "Kode Verifikasi Registrasi Admin Toko ToenToen"
					}
					return utils.SendEmailOTP(cleanEmail, newOTP, subject)
				}
			}
		}
	}

	return errors.New("Akun dengan email ini tidak ditemukan")
}

func (s *authService) VerifyEmailOTP(input models.VerifyEmailOTPInput) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(input.Email))
	if !isEmail(cleanEmail) {
		return errors.New("Format alamat email tidak valid")
	}

	groups := candidateGroups(input.Role)
	var matchedGroup string

	if config.RDB != nil {
		for _, g := range groups {
			key := scopedKey("email_verify_otp", g, cleanEmail)
			storedOTP, err := config.RDB.Get(config.Ctx, key).Result()
			if err == nil && storedOTP == input.OTP {
				matchedGroup = g
				config.RDB.Del(config.Ctx, key)
				break
			}
			regKey := scopedKey("register_otp", g, cleanEmail)
			storedRegOTP, regErr := config.RDB.Get(config.Ctx, regKey).Result()
			if regErr == nil && storedRegOTP == input.OTP {
				matchedGroup = g
				config.RDB.Del(config.Ctx, regKey)
				break
			}
		}

		// Fallback cek un-scoped legacy key
		if matchedGroup == "" {
			legacyKey := fmt.Sprintf("email_verify_otp:%s", cleanEmail)
			storedOTP, err := config.RDB.Get(config.Ctx, legacyKey).Result()
			if err == nil && storedOTP == input.OTP {
				config.RDB.Del(config.Ctx, legacyKey)
			} else {
				legacyReg := fmt.Sprintf("register_otp:%s", cleanEmail)
				storedRegOTP, regErr := config.RDB.Get(config.Ctx, legacyReg).Result()
				if regErr == nil && storedRegOTP == input.OTP {
					config.RDB.Del(config.Ctx, legacyReg)
				} else {
					return errors.New("OTP tidak valid atau sudah kadaluarsa")
				}
			}
		}
	}

	account, err := s.repo.FindByIdentifier(cleanEmail, matchedGroup)
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

	if err := s.repo.UpdateEmailVerified(account.AccountID, now, newStatus); err != nil {
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

	if input.ExpectedRole == "" && input.Role != "" {
		input.ExpectedRole = input.Role
	}
	targetRole := NormalizeExpectedRole(input.ExpectedRole)

	// Cari akun yang sesuai target role atau deteksi keberadaan di role lain
	account, err := s.findAccountForLogin(input.Identifier, targetRole)
	if err != nil {
		if errors.Is(err, ErrAmbiguousAccount) {
			return ErrAmbiguousAccount
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("Email atau password salah")
		}
		return errors.New("Terjadi kesalahan sistem")
	}

	// Jika akun ditemukan tetapi rolenya bukan target role yang dituju
	if targetRole != "" && account.Role != targetRole {
		return NewRoleMismatchError(account.Role, targetRole)
	}

	switch account.AccountStatus {
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

	group := models.RoleGroupOf(account.Role)
	loginSubj := "Kode Verifikasi Login ToenToen"
	if account.Role == "business_admin" {
		loginSubj = "Kode Verifikasi Login Admin Toko ToenToen"
	}
	return s.sendOTP(input.Identifier, group, account.Email, account.PhoneNumber, "login_otp", loginSubj)
}

func (s *authService) VerifyLoginOTP(input models.VerifyLoginOTPInput) (string, *models.Account, error) {
	input.Identifier = cleanIdentifier(input.Identifier)

	if input.ExpectedRole == "" && input.Role != "" {
		input.ExpectedRole = input.Role
	}
	targetRole := NormalizeExpectedRole(input.ExpectedRole)
	groups := candidateGroups(targetRole)

	var matchedGroup string
	if config.RDB != nil {
		for _, g := range groups {
			key := scopedKey("login_otp", g, input.Identifier)
			storedOTP, err := config.RDB.Get(config.Ctx, key).Result()
			if err == nil && storedOTP == input.OTP {
				matchedGroup = g
				config.RDB.Del(config.Ctx, key)
				break
			}
		}

		// Fallback check legacy key
		if matchedGroup == "" {
			legacyKey := fmt.Sprintf("login_otp:%s", input.Identifier)
			storedOTP, err := config.RDB.Get(config.Ctx, legacyKey).Result()
			if err == nil && storedOTP == input.OTP {
				config.RDB.Del(config.Ctx, legacyKey)
			} else {
				return "", nil, errors.New("OTP tidak valid atau sudah kadaluarsa")
			}
		}
	}

	// Cari akun pada matched group jika ada, atau target role
	lookupRole := targetRole
	if matchedGroup != "" && targetRole == "" {
		if matchedGroup == models.RoleGroupCustomer {
			lookupRole = "customer"
		}
	}

	account, err := s.findAccountInGroup(input.Identifier, lookupRole)
	if err != nil {
		return "", nil, errors.New("Gagal memuat data akun")
	}

	if targetRole != "" && account.Role != targetRole {
		return "", nil, NewRoleMismatchError(account.Role, targetRole)
	}

	now := time.Now()
	account.LastLoginAt = &now
	s.repo.Update(account)

	token, err := utils.GenerateToken(account.AccountID, account.Role)
	if err != nil {
		return "", nil, errors.New("Gagal membuat sesi pengguna")
	}

	return token, account, nil
}

func (s *authService) ForgotPassword(input models.ForgotPasswordInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if !isEmail(input.Identifier) {
		return errors.New("Layanan reset kata sandi menggunakan nomor telepon sedang dalam pengembangan. Silakan gunakan email terdaftar Anda.")
	}

	account, err := s.findAccountInGroup(input.Identifier, input.Role)
	if err != nil {
		if errors.Is(err, ErrAmbiguousAccount) {
			return ErrAmbiguousAccount
		}
		return errors.New("Akun tidak ditemukan")
	}

	group := models.RoleGroupOf(account.Role)
	// Kirim OTP verifikasi (berlaku untuk semua akun lokal maupun akun OAuth yang ingin membuat kata sandi baru)
	subject := "Kode Verifikasi Reset Password ToenToen"
	if account.Role == "business_admin" {
		if account.LastLoginAt == nil || account.PasswordHash == nil || account.AccountStatus == "inactive" {
			subject = "Kode Verifikasi Aktivasi Akun Admin Toko ToenToen"
		} else {
			subject = "Kode Verifikasi Reset Password Admin Toko ToenToen"
		}
	}
	return s.sendOTP(input.Identifier, group, account.Email, account.PhoneNumber, "forgot_otp", subject)
}

func (s *authService) VerifyForgotOTP(input models.VerifyForgotOTPInput) (string, error) {
	input.Identifier = cleanIdentifier(input.Identifier)
	groups := candidateGroups(input.Role)

	var matchedGroup string
	for _, g := range groups {
		redisKey := scopedKey("forgot_otp", g, input.Identifier)
		storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()
		if err == nil && storedOTP == input.OTP {
			matchedGroup = g
			config.RDB.Del(config.Ctx, redisKey)
			config.RDB.Del(config.Ctx, scopedKey("forgot_otp_attempts", g, input.Identifier))
			break
		}
	}

	// Fallback check legacy key
	if matchedGroup == "" {
		legacyKey := fmt.Sprintf("forgot_otp:%s", input.Identifier)
		storedOTP, err := config.RDB.Get(config.Ctx, legacyKey).Result()
		if err == nil && storedOTP == input.OTP {
			config.RDB.Del(config.Ctx, legacyKey)
			config.RDB.Del(config.Ctx, fmt.Sprintf("forgot_otp_attempts:%s", input.Identifier))
		} else {
			return "", errors.New("Kode OTP tidak valid atau sudah kadaluarsa")
		}
	}

	// Generate secure UUID reset token
	resetToken := utils.GenerateUUID("RST")
	if matchedGroup != "" {
		config.RDB.Set(config.Ctx, scopedKey("reset_token", matchedGroup, input.Identifier), resetToken, 10*time.Minute)
	}
	config.RDB.Set(config.Ctx, fmt.Sprintf("reset_token:%s", input.Identifier), resetToken, 10*time.Minute)

	return resetToken, nil
}

func (s *authService) ResetPassword(input models.ResetPasswordInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("Kata sandi baru dan konfirmasi kata sandi tidak cocok")
	}

	groups := candidateGroups(input.Role)
	var matchedGroup string
	var verifiedToken bool

	for _, g := range groups {
		tokenRedisKey := scopedKey("reset_token", g, input.Identifier)
		storedToken, err := config.RDB.Get(config.Ctx, tokenRedisKey).Result()
		if err == nil && storedToken == input.ResetToken {
			matchedGroup = g
			verifiedToken = true
			config.RDB.Del(config.Ctx, tokenRedisKey)
			break
		}
	}

	if !verifiedToken {
		legacyKey := fmt.Sprintf("reset_token:%s", input.Identifier)
		storedToken, err := config.RDB.Get(config.Ctx, legacyKey).Result()
		if err == nil && storedToken == input.ResetToken {
			verifiedToken = true
			config.RDB.Del(config.Ctx, legacyKey)
		}
	}

	if !verifiedToken {
		return errors.New("Sesi reset password tidak valid atau sudah kadaluarsa")
	}

	account, err := s.repo.FindByIdentifier(input.Identifier, matchedGroup)
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

	if !isEmail(newEmailClean) {
		return errors.New("Format alamat email baru tidak valid")
	}

	// 1. Cek apakah user saat ini sudah memiliki email terdaftar
	hasExistingEmail := account.Email != nil && strings.TrimSpace(*account.Email) != ""
	if hasExistingEmail {
		// Kasus: Mengganti Email (Change Email)
		if currentEmailClean == "" {
			return errors.New("Email saat ini wajib diisi untuk verifikasi keamanan")
		}
		if strings.ToLower(strings.TrimSpace(*account.Email)) != currentEmailClean {
			return errors.New("Email saat ini tidak cocok dengan data akun Anda")
		}
		if currentEmailClean == newEmailClean {
			return errors.New("Email baru tidak boleh sama dengan email saat ini")
		}
	}
	// Catatan: Jika hasExistingEmail == false (kasus penautan pertama kali / Bind),
	// current_email tidak perlu diperiksa dan tidak wajib diisi.

	// 2. Cek apakah email baru sudah terdaftar pada akun lain di kelompok role yang sama
	group := models.RoleGroupOf(account.Role)
	if exists, err := s.repo.FindByIdentifier(newEmailClean, group); err == nil && exists != nil && exists.AccountID != accountID {
		return errors.New("Email baru sudah terdaftar pada akun lain")
	}

	otp := utils.GenerateNumericOTP(6)
	redisKey := scopedKey("change_email_otp", group, newEmailClean)

	if err := config.RDB.Set(config.Ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return errors.New("Gagal memproses kode verifikasi")
	}

	subject := "Kode Verifikasi Perubahan Email ToenToen"
	if !hasExistingEmail {
		subject = "Kode Verifikasi Penautan Email Akun ToenToen"
	}

	if err := utils.SendEmailOTP(newEmailClean, otp, subject); err != nil {
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

	group := models.RoleGroupOf(account.Role)
	redisKey := scopedKey("change_email_otp", group, newEmailClean)
	attemptKey := scopedKey("change_email_attempts", group, newEmailClean)

	storedOTP, err := config.RDB.Get(config.Ctx, redisKey).Result()
	if err == redis.Nil || storedOTP != input.OTP {
		// Fallback check legacy key
		legacyKey := fmt.Sprintf("change_email_otp:%s", newEmailClean)
		legacyOTP, legErr := config.RDB.Get(config.Ctx, legacyKey).Result()
		if legErr != nil || legacyOTP != input.OTP {
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
		config.RDB.Del(config.Ctx, legacyKey)
	}

	if err := s.repo.UpdateEmail(accountID, newEmailClean); err != nil {
		return errors.New("Gagal memperbarui email")
	}

	config.RDB.Del(config.Ctx, redisKey)
	config.RDB.Del(config.Ctx, attemptKey)
	return nil
}

func (s *authService) RequestPhoneChange(accountID string, input models.RequestPhoneChangeInput) error {
	account, err := s.repo.FindByID(accountID)
	if err != nil {
		return errors.New("Sesi pengguna tidak valid")
	}

	currentPhoneClean := strings.TrimSpace(input.CurrentPhone)
	newPhoneClean := strings.TrimSpace(input.NewPhone)

	if newPhoneClean == "" {
		return errors.New("Nomor telepon baru wajib diisi")
	}

	// 1. Cek apakah user saat ini sudah memiliki nomor telepon terdaftar
	hasExistingPhone := account.PhoneNumber != nil && strings.TrimSpace(*account.PhoneNumber) != ""
	if hasExistingPhone {
		// Kasus: Mengganti Nomor Telepon (Change Phone)
		if currentPhoneClean == "" {
			return errors.New("Nomor telepon saat ini wajib diisi untuk verifikasi keamanan")
		}
		if strings.TrimSpace(*account.PhoneNumber) != currentPhoneClean {
			return errors.New("Nomor telepon saat ini tidak cocok dengan data akun Anda")
		}
		if currentPhoneClean == newPhoneClean {
			return errors.New("Nomor telepon baru tidak boleh sama dengan nomor telepon saat ini")
		}
	}
	// Catatan: Jika hasExistingPhone == false (kasus penautan pertama kali / Bind),
	// current_phone tidak perlu diperiksa dan tidak wajib diisi.

	// 2. Cek apakah nomor telepon baru sudah terdaftar pada akun lain di kelompok role yang sama
	group := models.RoleGroupOf(account.Role)
	if exists, err := s.repo.FindByIdentifier(newPhoneClean, group); err == nil && exists != nil && exists.AccountID != accountID {
		return errors.New("Nomor telepon baru sudah terdaftar pada akun lain")
	}

	return errors.New("Layanan verifikasi nomor telepon/WhatsApp sedang dalam tahap pengembangan. Fitur nomor telepon belum dapat digunakan saat ini.")
}

func (s *authService) VerifyPhoneChange(accountID string, input models.VerifyPhoneChangeInput) error {
	return errors.New("Layanan verifikasi nomor telepon/WhatsApp sedang dalam tahap pengembangan.")
}

func (s *authService) CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error) {
	// Pastikan identitas tidak dipakai oleh akun lain di kelompok merchant (owner atau admin)
	cleanIdent := cleanIdentifier(input.Identifier)
	if exists, err := s.repo.FindByIdentifier(cleanIdent, models.RoleGroupMerchant); err == nil && exists != nil {
		if exists.Role == "owner" {
			return nil, errors.New("Email/nomor ini sudah digunakan sebagai akun Pemilik Toko (Owner). Akun Owner dan Admin Toko tidak boleh menggunakan identitas yang sama.")
		}
		return nil, errors.New("Email/nomor ini sudah terdaftar sebagai Admin Toko lain.")
	}

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
