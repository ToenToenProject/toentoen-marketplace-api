package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
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
	// Normalize phone number by removing spaces and separators.
	identifier = strings.ReplaceAll(identifier, " ", "")
	identifier = strings.ReplaceAll(identifier, "-", "")
	identifier = strings.ReplaceAll(identifier, "+", "")
	return identifier
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var typoEmailDomains = map[string]bool{
	"gmaill.com":  true,
	"gamil.com":   true,
	"gmai.com":    true,
	"gmaild.com":  true,
	"gmail.co":    true,
	"yaho.com":    true,
	"yaho.co":     true,
	"yaho.co.id":  true,
	"hotmial.com": true,
	"outlok.com":  true,
	"outlock.com": true,
	"iclud.com":   true,
}

var disposableEmailDomains = map[string]bool{
	"mailinator.com":    true,
	"tempmail.com":      true,
	"guerrillamail.com": true,
	"10minutemail.com":  true,
	"throwawaymail.com": true,
	"dispostable.com":   true,
	"trashmail.com":     true,
	"yopmail.com":       true,
}

var knownValidDomains = map[string]bool{
	"gmail.com":      true,
	"googlemail.com": true,
	"yahoo.com":      true,
	"yahoo.co.id":    true,
	"ymail.com":      true,
	"outlook.com":    true,
	"hotmail.com":    true,
	"live.com":       true,
	"icloud.com":     true,
	"me.com":         true,
	"mac.com":        true,
	"proton.me":      true,
	"protonmail.com": true,
	"zoho.com":       true,
	"mail.com":       true,
	"aol.com":        true,
	"toentoen.com":   true,
}

var domainMXCache sync.Map

func checkDomainMX(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return false
	}
	if knownValidDomains[domain] {
		return true
	}
	if typoEmailDomains[domain] || disposableEmailDomains[domain] {
		return false
	}

	if val, ok := domainMXCache.Load(domain); ok {
		if hasMX, ok := val.(bool); ok {
			return hasMX
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	mxs, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			domainMXCache.Store(domain, false)
			return false
		}
		if strings.Contains(strings.ToLower(err.Error()), "no such host") {
			domainMXCache.Store(domain, false)
			return false
		}
		log.Printf("⚠️ DNS MX lookup gagal/timeout untuk domain %s: %v", domain, err)
		return false
	}

	hasMX := len(mxs) > 0
	domainMXCache.Store(domain, hasMX)
	return hasMX
}

func hasExcessiveRepetitiveChars(s string, maxConsecutive int) bool {
	if len(s) == 0 {
		return false
	}
	count := 1
	var last rune
	for i, r := range s {
		if i == 0 {
			last = r
			continue
		}
		if r == last {
			count++
			if count >= maxConsecutive {
				return true
			}
		} else {
			last = r
			count = 1
		}
	}
	return false
}

func isEmail(identifier string) bool {
	identifier = strings.TrimSpace(strings.ToLower(identifier))
	if len(identifier) < 6 || len(identifier) > 254 {
		return false
	}
	if strings.Contains(identifier, "..") {
		return false
	}
	addr, err := mail.ParseAddress(identifier)
	if err != nil {
		return false
	}
	if !emailRegex.MatchString(addr.Address) {
		return false
	}

	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return false
	}
	localPart, domain := parts[0], parts[1]

	// Reject known typo email domains.
	if typoEmailDomains[domain] {
		return false
	}

	// Reject disposable email domains.
	if disposableEmailDomains[domain] {
		return false
	}

	// Reject excessive repeated characters in local email part.
	if hasExcessiveRepetitiveChars(localPart, 5) {
		return false
	}

	// Verify domain has active mail server (DNS MX record).
	if !checkDomainMX(domain) {
		return false
	}

	return true
}

// roleGroupFromInput normalizes request role aliases to canonical role groups.
func roleGroupFromInput(role string) string {
	return models.RoleGroupOf(NormalizeExpectedRole(role))
}

// candidateGroups returns target role groups to inspect based on request parameters.
func candidateGroups(role string) []string {
	if g := roleGroupFromInput(role); g != "" {
		return []string{g}
	}
	return []string{models.RoleGroupCustomer, models.RoleGroupOwner, models.RoleGroupAdmin}
}

// scopedKey generates role-group-scoped Redis keys (e.g. login_otp:customer:user@mail.com).
func scopedKey(prefix, group, identifier string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, group, identifier)
}

// findAccountInGroup strictly searches within the requested role group without cross-group fallback.
func (s *authService) findAccountInGroup(identifier, role string) (*models.Account, error) {
	return s.repo.FindByIdentifier(identifier, roleGroupFromInput(role))
}

// findAccountForLogin finds accounts within the specified role group, supporting cross-merchant lookups
// (owner <-> business_admin) to guide users with RoleMismatchError without crossing into customer scope.
func (s *authService) findAccountForLogin(identifier, targetRole string) (*models.Account, error) {
	group := models.RoleGroupOf(targetRole)
	if group != "" {
		account, err := s.repo.FindByIdentifier(identifier, group)
		if err == nil {
			return account, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		// Sibling merchant fallback (owner <-> admin) to produce RoleMismatchError guidance.
		if group == models.RoleGroupOwner {
			if adminAcc, errAdmin := s.repo.FindByIdentifier(identifier, models.RoleGroupAdmin); errAdmin == nil && adminAcc != nil {
				return adminAcc, nil
			}
		} else if group == models.RoleGroupAdmin {
			if ownerAcc, errOwner := s.repo.FindByIdentifier(identifier, models.RoleGroupOwner); errOwner == nil && ownerAcc != nil {
				return ownerAcc, nil
			}
		}

		return nil, gorm.ErrRecordNotFound
	}

	return s.repo.FindByIdentifier(identifier, "")
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
	if isReviewerIdentifier(identifier) {
		otp = DefaultReviewerOTP
	}
	redisKey := scopedKey(redisPrefix, group, identifier)

	if config.RDB != nil {
		ttl := 5 * time.Minute
		if isReviewerIdentifier(identifier) {
			ttl = 24 * time.Hour
		}
		if err := config.RDB.Set(config.Ctx, redisKey, otp, ttl).Err(); err != nil {
			return errors.New("Gagal memproses kode verifikasi pada sistem")
		}
	}

	if isReviewerIdentifier(identifier) {
		log.Printf("ℹ️ [REVIEWER-OTP] Akun tester terdeteksi (%s), OTP statis %s aktif tanpa kirim email", identifier, DefaultReviewerOTP)
		return nil
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

	var sent bool
	var lastErr error

	if hasEmail {
		if err := utils.SendEmailOTP(cleanEmail, otp, subject); err != nil {
			lastErr = err
		} else {
			sent = true
		}
	}

	// Phone number verification is handled via Firebase Phone Authentication.
	// Phone identifier without registered email.
	if !sent && hasPhone {
		// Google Play reviewer test accounts use static OTP bypass.
		if isReviewerIdentifier(identifier) || isReviewerIdentifier(cleanPhone) {
			sent = true
		} else {
			// OTP is stored in Redis (5m TTL) for verification matching.
			sent = true
		}
	}

	if !sent {
		if config.RDB != nil {
			config.RDB.Del(config.Ctx, redisKey)
		}
		if lastErr != nil {
			log.Printf("[SEND-OTP-ERROR] Gagal mengirim OTP (%s): %v", identifier, lastErr)
		}
		if hasEmail {
			return errors.New("Gagal mengirimkan kode verifikasi ke email Anda. Pastikan alamat email aktif dan valid, atau coba beberapa saat lagi.")
		}
		return errors.New("Gagal memproses kode verifikasi. Pastikan data akun valid dan coba lagi.")
	}

	return nil
}

func (s *authService) Register(input models.RegisterUnifiedInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if strings.Contains(input.Identifier, "@") {
		if !isEmail(input.Identifier) {
			return errors.New("Format alamat email tidak valid")
		}
	}

	// Normalisasi role (alias FE) dan tentukan kelompok role.
	// customer  -> kelompok customer
	// Merchant roles (owner, business_admin) cannot share credentials.
	input.Role = NormalizeExpectedRole(input.Role)
	switch input.Role {
	case "customer", "owner", "business_admin":
		// allowed for self-registration
	default:
		return errors.New("Role tidak valid. Gunakan customer, owner, atau business_admin")
	}
	group := models.RoleGroupOf(input.Role)
	if group == "" {
		return errors.New("Role tidak valid. Gunakan customer, owner, atau business_admin")
	}

	// Enforce per-role-group blocklist from operations.blocked_identities
	if database.DB != nil {
		var blockedCount int64
		normID := strings.ToLower(strings.TrimSpace(input.Identifier))
		_ = database.DB.Raw(
			`SELECT COUNT(*) FROM operations.blocked_identities WHERE role_group = ? AND identifier_value = ? AND is_active = true`,
			input.Role, normID,
		).Scan(&blockedCount).Error
		if blockedCount > 0 {
			return errors.New("Identitas ini telah diblokir untuk role " + input.Role + " dan tidak dapat digunakan untuk mendaftar")
		}
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

	// Check uniqueness within the target role group.
	existingAccount, err := s.repo.FindByIdentifier(input.Identifier, group)
	if err == nil {
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

	// Cache pending registration payload and OTP in Redis (5m TTL).
	if config.RDB != nil {
		if err := config.RDB.Set(config.Ctx, pendingKey, string(pendingJSON), 5*time.Minute).Err(); err != nil {
			return errors.New("Gagal menyimpan data verifikasi ke sistem")
		}
		_ = config.RDB.Set(config.Ctx, otpKey, otp, 5*time.Minute).Err()
	}

	if isEmail(input.Identifier) {
		// Send verification OTP to email.
		subject := "Kode Verifikasi Registrasi Akun ToenToen"
		if input.Role == "business_admin" {
			subject = "Kode Verifikasi Registrasi Admin Toko ToenToen"
		}
		if err := utils.SendEmailOTP(input.Identifier, otp, subject); err != nil {
			if config.RDB != nil {
				config.RDB.Del(config.Ctx, pendingKey)
				config.RDB.Del(config.Ctx, otpKey)
			}
			return errors.New("Gagal mengirimkan kode verifikasi ke email Anda. Pastikan alamat email aktif dan valid, atau coba beberapa saat lagi.")
		}
	} else {
		// Phone registrations verify via Firebase Phone Auth.
		log.Printf("ℹ️ [REGISTER-PHONE] Registrasi nomor %s disimpan di pending registrasi untuk verifikasi via SMS Firebase", input.Identifier)
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

// verifyScopedOTPWithLockout checks scoped and legacy OTP keys using constant-time comparison
// and enforces a max-5-attempts lockout per identifier.
func verifyScopedOTPWithLockout(prefix string, groups []string, identifier, inputOTP string) (string, bool, error) {
	if config.RDB == nil {
		return "", false, errors.New("Layanan verifikasi sedang tidak tersedia")
	}
	cleanInput := strings.TrimSpace(inputOTP)
	if cleanInput == "" {
		return "", false, errors.New("OTP wajib diisi")
	}

	var matchedKey string
	var matchedAttemptsKey string
	var matchedGroup string

	for _, g := range groups {
		k := scopedKey(prefix, g, identifier)
		storedOTP, err := config.RDB.Get(config.Ctx, k).Result()
		if err != nil {
			continue
		}
		ak := scopedKey(prefix+"_attempts", g, identifier)
		if subtle.ConstantTimeCompare([]byte(storedOTP), []byte(cleanInput)) == 1 {
			config.RDB.Del(config.Ctx, k, ak)
			return g, true, nil
		}
		if matchedKey == "" {
			matchedKey = k
			matchedAttemptsKey = ak
			matchedGroup = g
		}
	}

	legacyKey := fmt.Sprintf("%s:%s", prefix, identifier)
	if storedOTP, err := config.RDB.Get(config.Ctx, legacyKey).Result(); err == nil {
		ak := fmt.Sprintf("%s_attempts:%s", prefix, identifier)
		if subtle.ConstantTimeCompare([]byte(storedOTP), []byte(cleanInput)) == 1 {
			config.RDB.Del(config.Ctx, legacyKey, ak)
			return "", true, nil
		}
		if matchedKey == "" {
			matchedKey = legacyKey
			matchedAttemptsKey = ak
		}
	}

	if matchedKey != "" && matchedAttemptsKey != "" {
		attempts, _ := config.RDB.Incr(config.Ctx, matchedAttemptsKey).Result()
		if attempts == 1 {
			config.RDB.Expire(config.Ctx, matchedAttemptsKey, 5*time.Minute)
		}
		if attempts >= 5 {
			config.RDB.Del(config.Ctx, matchedKey, matchedAttemptsKey)
			return matchedGroup, false, errors.New("Terlalu banyak percobaan OTP salah, silakan minta kode baru")
		}
	}
	return "", false, errors.New("OTP tidak valid atau sudah kadaluarsa")
}

func (s *authService) VerifyRegisterOTP(input models.VerifyRegisterOTPInput) (string, error) {
	input.Identifier = cleanIdentifier(input.Identifier)
	groups := candidateGroups(input.Role)

	tokenStr := strings.TrimSpace(input.FirebaseToken)
	if tokenStr == "" {
		tokenStr = strings.TrimSpace(input.IDToken)
	}
	isFirebaseVerified := false
	if tokenStr != "" {
		verifiedPhone, err := utils.VerifyFirebasePhoneToken(tokenStr)
		if err != nil {
			return "", err
		}
		if !utils.MatchPhoneNumbers(verifiedPhone, input.Identifier) {
			return "", errors.New("Nomor telepon pada klaim token Firebase tidak cocok dengan identitas pendaftaran")
		}
		isFirebaseVerified = true
	}

	if config.RDB != nil {
		// Locate pending registration across candidate role groups.
		// Use the role group matching the submitted OTP.
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
			if !isFirebaseVerified && p.OTP != input.OTP {
				continue
			}
			if pending != nil {
				return "", ErrAmbiguousAccount
			}
			matched := p
			pending = &matched
			group = g
		}
		if pending == nil && hasAnyPending && !isFirebaseVerified {
			return "", errors.New("OTP tidak valid atau sudah kadaluarsa")
		}

		if pending != nil {
			pendingKey := scopedKey("register_pending", group, input.Identifier)
			otpKey := scopedKey("register_otp", group, input.Identifier)

			// Prevent duplicate registrations within the same role group.
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
				Username:     utils.GenerateRandomUsername(pending.Identifier, pending.Role),
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

			// Persist account and role profile atomically.
			if err := s.repo.CreateAccountWithProfile(&account, pending.Role); err != nil {
				log.Printf("[REGISTER-DB-ERROR] Gagal create account (%s, %s): %v", pending.Identifier, pending.Role, err)
				return "", errors.New("Gagal membuat akun baru di database")
			}

			// Clean up temporary Redis cache keys.
			config.RDB.Del(config.Ctx, pendingKey)
			config.RDB.Del(config.Ctx, otpKey)

			// Issue authentication tokens.
			var emailStr string
			if account.Email != nil {
				emailStr = *account.Email
			}
			token, err := utils.GenerateToken(account.AccountID, account.Role, emailStr)
			if err != nil {
				return "", errors.New("Akun berhasil dibuat, namun gagal membuat sesi login otomatis")
			}

			return token, nil
		}
	}

	// Fallback lookup for legacy or scoped registration OTP.
	matchedGroup, ok, errOTP := verifyScopedOTPWithLockout("register_otp", groups, input.Identifier, input.OTP)
	if !ok {
		if errOTP != nil && strings.Contains(errOTP.Error(), "Terlalu banyak") {
			return "", errOTP
		}
		if errOTP != nil && strings.Contains(errOTP.Error(), "tidak tersedia") {
			return "", errOTP
		}
		return "", errors.New("OTP tidak valid atau sudah kadaluarsa. Silakan lakukan pendaftaran ulang.")
	}
	var account *models.Account

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

	var emailStr string
	if account.Email != nil {
		emailStr = *account.Email
	}
	token, err := utils.GenerateToken(account.AccountID, account.Role, emailStr)
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

	account, err := s.repo.FindByIdentifier(cleanEmail, roleGroupFromInput(role))
	if err != nil {
		if errors.Is(err, ErrAmbiguousAccount) {
			return ErrAmbiguousAccount
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
}

func (s *authService) VerifyEmailOTP(input models.VerifyEmailOTPInput) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(input.Email))
	if !isEmail(cleanEmail) {
		return errors.New("Format alamat email tidak valid")
	}

	if config.RDB == nil {
		return errors.New("Layanan verifikasi sedang tidak tersedia")
	}

	groups := candidateGroups(input.Role)
	matchedGroup, ok, errOTP := verifyScopedOTPWithLockout("email_verify_otp", groups, cleanEmail, input.OTP)
	if !ok {
		return errOTP
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
	cleanPhone := utils.NormalizePhoneNumber(phone)
	if len(cleanPhone) < 7 || len(cleanPhone) > 15 {
		return errors.New("Nomor telepon tidak valid (harus 7-15 digit angka)")
	}

	// 1. Google Play reviewer bypass does not dispatch SMS/WA.
	if isReviewerIdentifier(phone) || isReviewerIdentifier(cleanPhone) {
		return nil
	}

	// 2. Rate limit OTP dispatch to 60-second cooldown per phone number.
	rateKey := fmt.Sprintf("ratelimit:phone_otp:%s", cleanPhone)
	if config.RDB != nil {
		if exists, _ := config.RDB.Exists(config.Ctx, rateKey).Result(); exists > 0 {
			return errors.New("Harap tunggu 60 detik sebelum meminta kode OTP kembali")
		}
	}

	// 3. Store OTP in Redis (5m TTL) for verification matching.
	otp := utils.GenerateNumericOTP(6)
	otpKey := fmt.Sprintf("phone_otp:%s", cleanPhone)
	if config.RDB != nil {
		if err := config.RDB.Set(config.Ctx, otpKey, otp, 5*time.Minute).Err(); err != nil {
			return errors.New("Gagal memproses kode OTP pada sistem")
		}
		_ = config.RDB.Set(config.Ctx, rateKey, "1", 60*time.Second).Err()
	}

	log.Printf("ℹ️ [PHONE-OTP] OTP untuk nomor %s disiapkan di Redis (aplikasi memverifikasi via SMS Firebase)", cleanPhone)
	return nil
}

func (s *authService) VerifyPhoneOTP(input models.VerifyPhoneOTPInput) error {
	cleanPhone := utils.NormalizePhoneNumber(input.Phone)
	if len(cleanPhone) < 7 || len(cleanPhone) > 15 {
		return errors.New("Nomor telepon tidak valid")
	}

	tokenStr := strings.TrimSpace(input.FirebaseToken)
	if tokenStr == "" {
		tokenStr = strings.TrimSpace(input.IDToken)
	}

	// Mode 1: Verification via Firebase ID Token.
	if tokenStr != "" {
		verifiedPhone, err := utils.VerifyFirebasePhoneToken(tokenStr)
		if err != nil {
			return err
		}
		if !utils.MatchPhoneNumbers(verifiedPhone, input.Phone) {
			return errors.New("Nomor telepon pada klaim token Firebase tidak cocok dengan nomor yang diajukan")
		}
	} else {
		// Mode 2: Verification via OTP / WhatsApp / Reviewer mock.
		inputOTP := strings.TrimSpace(input.OTP)
		if inputOTP == "" {
			return errors.New("Kode OTP atau Firebase Token wajib disertakan")
		}

		isReviewer := isReviewerIdentifier(input.Phone) || isReviewerIdentifier(cleanPhone)
		if !(isReviewer && inputOTP == DefaultReviewerOTP) {
			if config.RDB == nil {
				return errors.New("Layanan verifikasi sedang tidak tersedia")
			}
			otpKey := fmt.Sprintf("phone_otp:%s", cleanPhone)
			attemptsKey := fmt.Sprintf("phone_otp_attempts:%s", cleanPhone)
			storedOTP, err := config.RDB.Get(config.Ctx, otpKey).Result()
			if err != nil || storedOTP == "" {
				return errors.New("Kode OTP tidak valid atau sudah kedaluwarsa")
			}
			if subtle.ConstantTimeCompare([]byte(storedOTP), []byte(inputOTP)) != 1 {
				attempts, _ := config.RDB.Incr(config.Ctx, attemptsKey).Result()
				if attempts == 1 {
					config.RDB.Expire(config.Ctx, attemptsKey, 5*time.Minute)
				}
				if attempts >= 5 {
					config.RDB.Del(config.Ctx, otpKey, attemptsKey)
					return errors.New("Terlalu banyak percobaan OTP salah, silakan minta kode baru")
				}
				return errors.New("Kode OTP salah")
			}
			config.RDB.Del(config.Ctx, otpKey, attemptsKey)
		}
	}

	// Mark account phone as verified.
	roleGroup := ""
	if input.Role != "" {
		roleGroup = models.RoleGroupOf(NormalizeExpectedRole(input.Role))
	}
	account, err := s.repo.FindByIdentifier(input.Phone, roleGroup)
	if err == nil && account != nil {
		now := time.Now()
		account.PhoneVerifiedAt = &now
		if account.AccountStatus == "inactive" {
			account.AccountStatus = "active"
		}
		_ = s.repo.Update(account)
	}

	return nil
}

func (s *authService) Login(input models.LoginUnifiedInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)

	if isReviewerIdentifier(input.Identifier) {
		_ = s.ensureReviewerAccounts(input.Identifier)
	}

	if input.ExpectedRole == "" && input.Role != "" {
		input.ExpectedRole = input.Role
	}
	targetRole := NormalizeExpectedRole(input.ExpectedRole)

	// Check all matching accounts across role groups when role is unspecified.
	if targetRole == "" {
		allAccounts, err := s.repo.FindAllByIdentifier(input.Identifier)
		if err != nil {
			return errors.New("Terjadi kesalahan sistem")
		}
		if len(allAccounts) == 0 {
			return errors.New("Email atau password salah")
		}

		// Validate password across matched role accounts.
		var matchingAccounts []*models.Account
		var suspendedOrBannedErr error
		for _, acc := range allAccounts {
			if acc.PasswordHash == nil {
				continue
			}
			passMatch := bcrypt.CompareHashAndPassword([]byte(*acc.PasswordHash), []byte(input.Password)) == nil
			if !passMatch && isReviewerIdentifier(input.Identifier) && isReviewerPassword(input.Password) {
				passMatch = true
			}
			if passMatch {
				if acc.AccountStatus == "suspended" {
					suspendedOrBannedErr = errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
					continue
				}
				if acc.AccountStatus == "banned" {
					suspendedOrBannedErr = errors.New("Akun anda telah diblokir dari platform")
					continue
				}
				matchingAccounts = append(matchingAccounts, acc)
			}
		}

		if len(matchingAccounts) == 0 {
			if suspendedOrBannedErr != nil {
				return suspendedOrBannedErr
			}
			return errors.New("Email atau password salah")
		}

		// Return MultiRoleRequiredError when credentials match multiple role accounts.
		if len(matchingAccounts) > 1 {
			var roles []string
			var emailStr string
			for _, acc := range matchingAccounts {
				roles = append(roles, acc.Role)
				if emailStr == "" && acc.Email != nil && *acc.Email != "" {
					emailStr = *acc.Email
				}
			}
			if emailStr == "" && isEmail(input.Identifier) {
				emailStr = input.Identifier
			}
			return &MultiRoleRequiredError{
				Roles: roles,
				Email: emailStr,
			}
		}

		// Single matching role: proceed with authentication.
		account := matchingAccounts[0]
		group := models.RoleGroupOf(account.Role)
		loginSubj := "Kode Verifikasi Login ToenToen"
		if account.Role == "business_admin" {
			loginSubj = "Kode Verifikasi Login Admin Toko ToenToen"
		}
		return s.sendOTP(input.Identifier, group, account.Email, account.PhoneNumber, "login_otp", loginSubj)
	}

	// Target role explicitly specified.
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

	// Account exists but role differs from target portal.
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

	passMatch := bcrypt.CompareHashAndPassword([]byte(*account.PasswordHash), []byte(input.Password)) == nil
	if !passMatch && isReviewerIdentifier(input.Identifier) && isReviewerPassword(input.Password) {
		passMatch = true
	}
	if !passMatch {
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
	tokenStr := strings.TrimSpace(input.FirebaseToken)
	if tokenStr == "" {
		tokenStr = strings.TrimSpace(input.IDToken)
	}

	if tokenStr != "" {
		if config.RDB == nil {
			return "", nil, errors.New("Layanan verifikasi sedang tidak tersedia")
		}
		verifiedPhone, err := utils.VerifyFirebasePhoneToken(tokenStr)
		if err != nil {
			return "", nil, err
		}
		if !utils.MatchPhoneNumbers(verifiedPhone, input.Identifier) {
			return "", nil, errors.New("Nomor telepon pada klaim token Firebase tidak cocok dengan identitas login")
		}
		// Require a pending login challenge created by POST /auth/login (password check).
		challengeFound := false
		for _, g := range groups {
			k := scopedKey("login_otp", g, input.Identifier)
			if exists, errEx := config.RDB.Exists(config.Ctx, k).Result(); errEx == nil && exists > 0 {
				matchedGroup = g
				challengeFound = true
				config.RDB.Del(config.Ctx, k, scopedKey("login_otp_attempts", g, input.Identifier))
				break
			}
		}
		if !challengeFound {
			return "", nil, errors.New("Sesi login tidak ditemukan atau sudah kadaluarsa, silakan login ulang")
		}
	} else if isReviewerIdentifier(input.Identifier) && input.OTP == DefaultReviewerOTP {
		if targetRole != "" {
			matchedGroup = models.RoleGroupOf(targetRole)
		} else {
			matchedGroup = models.RoleGroupCustomer
		}
	} else {
		var ok bool
		var errOTP error
		matchedGroup, ok, errOTP = verifyScopedOTPWithLockout("login_otp", groups, input.Identifier, input.OTP)
		if !ok {
			return "", nil, errOTP
		}
	}

	// Look up account in matched group or target role.
	lookupRole := targetRole
	if matchedGroup != "" && targetRole == "" {
		lookupRole = matchedGroup
	}

	account, err := s.findAccountInGroup(input.Identifier, lookupRole)
	if err != nil {
		return "", nil, errors.New("Gagal memuat data akun")
	}

	if targetRole != "" && account.Role != targetRole {
		return "", nil, NewRoleMismatchError(account.Role, targetRole)
	}

	switch account.AccountStatus {
	case "active":
		// allowed
	case "banned":
		return "", nil, errors.New("Akun anda telah diblokir dari platform")
	case "suspended":
		return "", nil, errors.New("Akun anda sedang ditangguhkan, silakan hubungi support untuk informasi lebih lanjut")
	default:
		return "", nil, errors.New("Akun tidak aktif atau sedang dalam proses penghapusan")
	}

	now := time.Now()
	account.LastLoginAt = &now
	s.repo.Update(account)

	var emailStr string
	if account.Email != nil {
		emailStr = *account.Email
	}
	token, err := utils.GenerateToken(account.AccountID, account.Role, emailStr)
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
	// Send verification OTP for password reset.
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
	if isReviewerIdentifier(input.Identifier) && input.OTP == DefaultReviewerOTP {
		if input.Role != "" {
			matchedGroup = models.RoleGroupOf(input.Role)
		} else {
			matchedGroup = models.RoleGroupCustomer
		}
	} else {
		var ok bool
		var errOTP error
		matchedGroup, ok, errOTP = verifyScopedOTPWithLockout("forgot_otp", groups, input.Identifier, input.OTP)
		if !ok {
			if errOTP != nil && (strings.Contains(errOTP.Error(), "Terlalu banyak") || strings.Contains(errOTP.Error(), "tidak tersedia")) {
				return "", errOTP
			}
			return "", errors.New("Kode OTP tidak valid atau sudah kadaluarsa")
		}
	}

	if matchedGroup == "" {
		matchedGroup = models.RoleGroupCustomer
	}
	resetToken, errTok := utils.GenerateSecureToken(32)
	if errTok != nil {
		return "", errors.New("Gagal membuat token reset kata sandi")
	}
	if config.RDB == nil {
		return "", errors.New("Layanan verifikasi sedang tidak tersedia")
	}
	config.RDB.Set(config.Ctx, scopedKey("reset_token", matchedGroup, input.Identifier), resetToken, 10*time.Minute)

	return resetToken, nil
}

func (s *authService) ResetPassword(input models.ResetPasswordInput) error {
	input.Identifier = cleanIdentifier(input.Identifier)
	if input.NewPassword != input.ConfirmNewPassword {
		return errors.New("Kata sandi baru dan konfirmasi kata sandi tidak cocok")
	}
	if config.RDB == nil {
		return errors.New("Layanan verifikasi sedang tidak tersedia")
	}

	groups := candidateGroups(input.Role)
	var matchedGroup string
	var verifiedToken bool

	for _, g := range groups {
		tokenRedisKey := scopedKey("reset_token", g, input.Identifier)
		storedToken, err := config.RDB.Get(config.Ctx, tokenRedisKey).Result()
		if err == nil && subtle.ConstantTimeCompare([]byte(storedToken), []byte(input.ResetToken)) == 1 {
			matchedGroup = g
			verifiedToken = true
			config.RDB.Del(config.Ctx, tokenRedisKey)
			break
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

	// Activate inactive account upon successful password reset verification.
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

	// Require current password verification when setting a new password.
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

	// Check whether user currently has a registered email.
	hasExistingEmail := account.Email != nil && strings.TrimSpace(*account.Email) != ""
	if hasExistingEmail {
		// Case: updating existing email.
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
	// Initial email binding does not require current email verification.

	// Ensure new email is not taken within the same role group.
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
		return errors.New("Gagal mengirimkan kode verifikasi ke email baru. Pastikan alamat email aktif dan valid, atau coba beberapa saat lagi.")
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

	// Check whether user currently has a registered phone number.
	hasExistingPhone := account.PhoneNumber != nil && strings.TrimSpace(*account.PhoneNumber) != ""
	if hasExistingPhone {
		// Case: updating existing phone number.
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
	// Initial phone binding does not require current phone verification.

	// Ensure new phone number is not taken within the same role group.
	group := models.RoleGroupOf(account.Role)
	if exists, err := s.repo.FindByIdentifier(newPhoneClean, group); err == nil && exists != nil && exists.AccountID != accountID {
		return errors.New("Nomor telepon baru sudah terdaftar pada akun lain")
	}

	// Cache change request in Redis for client verification.
	otp := utils.GenerateNumericOTP(6)
	changeData := map[string]string{
		"new_phone": newPhoneClean,
		"otp":       otp,
	}
	changeJSON, _ := json.Marshal(changeData)
	redisKey := fmt.Sprintf("phone_change:%s", accountID)
	if config.RDB != nil {
		_ = config.RDB.Set(config.Ctx, redisKey, string(changeJSON), 5*time.Minute).Err()
	}

	return nil
}

func (s *authService) VerifyPhoneChange(accountID string, input models.VerifyPhoneChangeInput) error {
	account, err := s.repo.FindByID(accountID)
	if err != nil {
		return errors.New("Sesi pengguna tidak valid")
	}

	newPhoneClean := strings.TrimSpace(input.NewPhone)
	if newPhoneClean == "" {
		return errors.New("Nomor telepon baru wajib diisi")
	}

	tokenStr := strings.TrimSpace(input.FirebaseToken)
	if tokenStr == "" {
		tokenStr = strings.TrimSpace(input.IDToken)
	}

	// Verify new phone number via Firebase token or OTP.
	if tokenStr != "" {
		verifiedPhone, err := utils.VerifyFirebasePhoneToken(tokenStr)
		if err != nil {
			return err
		}
		if !utils.MatchPhoneNumbers(verifiedPhone, newPhoneClean) {
			return errors.New("Nomor telepon pada token Firebase tidak cocok dengan nomor baru yang diajukan")
		}
	} else {
		inputOTP := strings.TrimSpace(input.OTP)
		if inputOTP == "" {
			return errors.New("Kode OTP atau Firebase Token wajib disertakan")
		}

		isReviewer := isReviewerIdentifier(newPhoneClean)
		if !(isReviewer && inputOTP == DefaultReviewerOTP) {
			if config.RDB == nil {
				return errors.New("Layanan verifikasi sedang tidak tersedia")
			}
			redisKey := fmt.Sprintf("phone_change:%s", accountID)
			attemptsKey := fmt.Sprintf("phone_change_attempts:%s", accountID)
			storedJSON, err := config.RDB.Get(config.Ctx, redisKey).Result()
			if err != nil || storedJSON == "" {
				return errors.New("Permintaan ubah nomor telepon tidak ditemukan atau sudah kedaluwarsa")
			}
			var changeData map[string]string
			if err := json.Unmarshal([]byte(storedJSON), &changeData); err != nil {
				return errors.New("Gagal memproses data verifikasi")
			}
			if subtle.ConstantTimeCompare([]byte(changeData["otp"]), []byte(inputOTP)) != 1 || !utils.MatchPhoneNumbers(changeData["new_phone"], newPhoneClean) {
				attempts, _ := config.RDB.Incr(config.Ctx, attemptsKey).Result()
				if attempts == 1 {
					config.RDB.Expire(config.Ctx, attemptsKey, 5*time.Minute)
				}
				if attempts >= 5 {
					config.RDB.Del(config.Ctx, redisKey, attemptsKey)
					return errors.New("Terlalu banyak percobaan OTP salah, silakan minta kode baru")
				}
				return errors.New("Kode OTP tidak valid")
			}
			config.RDB.Del(config.Ctx, redisKey, attemptsKey)
		}
	}

	// Ensure verified phone number is not in use within the role group.
	group := models.RoleGroupOf(account.Role)
	if exists, err := s.repo.FindByIdentifier(newPhoneClean, group); err == nil && exists != nil && exists.AccountID != accountID {
		return errors.New("Nomor telepon baru sudah terdaftar pada akun lain")
	}

	// Save new phone number to account.
	now := time.Now()
	account.PhoneNumber = &newPhoneClean
	account.PhoneVerifiedAt = &now
	if err := s.repo.Update(account); err != nil {
		return errors.New("Gagal menyimpan nomor telepon baru")
	}

	return nil
}

func (s *authService) CreateBusinessAdmin(ownerID string, input models.CreateBusinessAdminInput) (*models.Account, error) {
	// Ensure identity is not claimed by another business admin.
	cleanIdent := cleanIdentifier(input.Identifier)
	if exists, err := s.repo.FindByIdentifier(cleanIdent, models.RoleGroupAdmin); err == nil && exists != nil {
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
		Username:      utils.GenerateRandomUsername(input.Identifier, "business_admin"),
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
