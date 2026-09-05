package business

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"
)

func encryptBusinessLegal(b *models.Business) {
	if b == nil {
		return
	}
	b.NIB, _ = crypto.EncryptPointer(b.NIB)
	for i := range b.Certificates {
		if b.Certificates[i].Number != "" {
			b.Certificates[i].Number, _ = crypto.Encrypt(b.Certificates[i].Number)
		}
		if b.Certificates[i].ImageURL != "" {
			b.Certificates[i].ImageURL, _ = crypto.Encrypt(b.Certificates[i].ImageURL)
		}
	}
}

func decryptBusinessLegal(b *models.Business) {
	if b == nil {
		return
	}
	b.NIB, _ = crypto.DecryptPointer(b.NIB)
	for i := range b.Certificates {
		if b.Certificates[i].Number != "" {
			b.Certificates[i].Number, _ = crypto.Decrypt(b.Certificates[i].Number)
		}
		if b.Certificates[i].ImageURL != "" {
			b.Certificates[i].ImageURL, _ = crypto.Decrypt(b.Certificates[i].ImageURL)
		}
	}
}

type BusinessService interface {
	CreateBusiness(ownerID string, input models.CreateBusinessInput) (string, error)
	GetMyBusinesses(ownerID, search string, categoryID int64, tagIDs []int64, cursor string, limit int) (*models.GetBusinessListResponse, error)
	GetBusinessDetail(businessID string) (*models.Business, error)
	UpdateBusiness(ownerID, businessID string, input models.UpdateBusinessInput) (error)
	DeleteBusiness(ownerID, businessID string) error
	ToggleStoreStatus(ownerID, businessID string) (bool, error)
	ToggleLiveStatus(requesterID, businessID string, input models.ToggleLiveInput) (*models.LiveStatusResponse, error)
}

type businessService struct {
	repo         BusinessRepository
	notifService notification.NotificationService
}

func NewBusinessService(repo BusinessRepository, notifService notification.NotificationService) BusinessService {
	return &businessService{repo, notifService}
}

func toPointerOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseDatePointer(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse(time.DateOnly, dateStr)
	if err != nil {
		return nil
	}
	return &t
}

func (s *businessService) CreateBusiness(ownerID string, input models.CreateBusinessInput) (string, error) {
	business := models.Business{
		BusinessID:       utils.GenerateUUID("BUS"),
		AccountID:        ownerID,
		LogoURL:          toPointerOrNil(input.LogoURL),
		BusinessName:     &input.BusinessName,
		Email:            &input.Email, // Ditambahkan karena sekarang wajib (NOT NULL)
		Status:           "draft",
		SubscriptionTier: "free",
		BusinessDetails:  toPointerOrNil(input.BusinessDetails),
		NIB:              toPointerOrNil(input.NIB),
		Country:          toPointerOrNil(input.Country),
		Province:         toPointerOrNil(input.Province),
		City:             toPointerOrNil(input.City),
		AddressDetail:    toPointerOrNil(input.AddressDetail),
		GoogleMapsLink:   toPointerOrNil(input.GoogleMapsLink),
		Latitude:         &input.Latitude,
		Longitude:        &input.Longitude,
		
		IsLocationShared: input.IsLocationShared,
		VenueName:        toPointerOrNil(input.VenueName),
		EventStartDate:   parseDatePointer(input.EventStartDate),
		EventEndDate:     parseDatePointer(input.EventEndDate),
		EventSchedules:   input.EventSchedules,

		StoreType:        toPointerOrNil(input.StoreType),
		PhoneContact:     toPointerOrNil(input.PhoneContact),
		WhatsappContact:  toPointerOrNil(input.WhatsappContact),
		SocialInstagram:  toPointerOrNil(input.SocialInstagram),
		SocialTiktok:     toPointerOrNil(input.SocialTiktok),
		
		// 🪄 MAGIC JSONB: Langsung masukkan array-nya, GORM akan mengurus sisanya
		BusinessImages:   input.BusinessImages,
		Certificates:     input.Certificates,
		SocialWebsites:   input.SocialWebsites,
		
		IsActive:         true,
	}

	var hours []models.BusinessHour
	for _, h := range input.Hours {
		hour := models.BusinessHour{
			DayOfWeek: h.DayOfWeek,
			OpenTime:  toPointerOrNil(h.OpenTime),
			CloseTime: toPointerOrNil(h.CloseTime),
			// IsClosed diabaikan, diasumsikan open jika masuk list
		}
		hours = append(hours, hour)
	}

	// Looping Websites dan Images manual dihapus

	// Enkripsi data sensitif (NIB & Dokumen Sertifikat)
	encryptBusinessLegal(&business)

	// Panggil Repository dengan 4 Argumen saja
	if err := s.repo.CreateBusinessWithRelations(&business, input.CategoryIDs, input.Tags, hours); err != nil {
		return "", err
	}

	return business.BusinessID, nil
}

func (s *businessService) GetMyBusinesses(ownerID, search string, categoryID int64, tagIDs []int64, cursor string, limit int) (*models.GetBusinessListResponse, error) {
	// Teruskan parameter pencarian dan filter ke repository
	businessesWithPreview, err := s.repo.GetBusinessListWithPreviews(ownerID, search, categoryID, tagIDs, cursor, limit+1)
	if err != nil {
		return nil, errors.New("gagal mengambil daftar bisnis")
	}

	var responseList []models.BusinessListItem
	hasMore := false
	nextCursor := ""

	if len(businessesWithPreview) > limit {
		hasMore = true
		businessesWithPreview = businessesWithPreview[:limit]
	}

	for _, b := range businessesWithPreview {
		var mainImage string
		if b.LogoURL != nil && *b.LogoURL != "" {
			mainImage = *b.LogoURL
		} else if len(b.BusinessImages) > 0 {
			mainImage = b.BusinessImages[0]
		}

		var fullAddress string
		if b.AddressDetail != nil {
			fullAddress += *b.AddressDetail
		}
		if b.City != nil {
			if fullAddress != "" {
				fullAddress += ", "
			}
			fullAddress += *b.City
		}
		if b.Province != nil {
			if fullAddress != "" {
				fullAddress += ", "
			}
			fullAddress += *b.Province
		}

		item := models.BusinessListItem{
			BusinessID:          b.BusinessID,
			BusinessName:        *b.BusinessName,
			OwnerName:           b.OwnerName,
			LogoURL:             mainImage,
			Address:             fullAddress,
			AverageRating:       b.AverageRating,
			TotalProductReviews: b.TotalProductReviews,
		}
		
		responseList = append(responseList, item)
	}

	if len(responseList) > 0 && hasMore {
		nextCursor = responseList[len(responseList)-1].BusinessID
	}

	response := &models.GetBusinessListResponse{
		Businesses: responseList,
		Pagination: models.PaginationMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}

	return response, nil
}

func (s *businessService) GetBusinessDetail(businessID string) (*models.Business, error) {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return nil, errors.New("bisnis tidak ditemukan")
	}

	// Dekripsi dokumen sensitif untuk owner/admin
	decryptBusinessLegal(business)

	// 🪄 MAGIC: Hitung is_open_now secara live sebelum dikembalikan ke Frontend
	business.IsOpenNow = utils.CalculateStoreStatus(business.IsManuallyClosed, business.Hours)

	return business, nil
}

func (s *businessService) UpdateBusiness(requesterID, businessID string, input models.UpdateBusinessInput) (error) {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan")
	}

	// Verifikasi apakah requester adalah owner atau admin aktif di toko ini
	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return errors.New("forbidden: Anda tidak memiliki akses untuk memperbarui bisnis ini")
		}

		if assignment.Role == "staff" && (len(input.Hours) > 0 || len(input.EventSchedules) > 0) {
			if !permissions.HasPermission(database.DB, requesterID, businessID, models.PermChangeBizHours) {
				return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah jam operasional bisnis")
			}
		}
	}

	// Update field sederhana
	if input.BusinessName != "" {
		business.BusinessName = &input.BusinessName
	}
	if input.Handle != "" {
		if s.repo.CheckHandleExists(input.Handle, business.BusinessID) {
			return errors.New("handle sudah digunakan, silakan pilih yang lain")
		}
		business.Handle = &input.Handle
	}
	if input.Email != "" {
		business.Email = &input.Email
	}

	if input.BusinessDetails != "" {
		business.BusinessDetails = &input.BusinessDetails
	}
	if input.NIB != "" {
		business.NIB = &input.NIB
	}
	if input.Country != "" {
		business.Country = &input.Country
	}
	if input.Province != "" {
		business.Province = &input.Province
	}
	if input.City != "" {
		business.City = &input.City
	}
	if input.AddressDetail != "" {
		business.AddressDetail = &input.AddressDetail
	}
	if input.GoogleMapsLink != "" {
		business.GoogleMapsLink = &input.GoogleMapsLink
	}
	
	// Address, StartDate, EndDate dihapus karena tidak ada di model
	
	if input.Latitude != 0 {
		business.Latitude = &input.Latitude
	}
	if input.Longitude != 0 {
		business.Longitude = &input.Longitude
	}

	business.IsLocationShared = input.IsLocationShared
	
	if input.VenueName != "" {
		business.VenueName = &input.VenueName
	}
	if input.EventStartDate != "" {
		business.EventStartDate = parseDatePointer(input.EventStartDate)
	}
	if input.EventEndDate != "" {
		business.EventEndDate = parseDatePointer(input.EventEndDate)
	}
	if len(input.EventSchedules) > 0 {
		business.EventSchedules = input.EventSchedules
	}

	if input.StoreType != "" {
		business.StoreType = &input.StoreType
	}

	if input.PhoneContact != "" {
		business.PhoneContact = &input.PhoneContact
	}
	if input.WhatsappContact != "" {
		business.WhatsappContact = &input.WhatsappContact
	}
	if input.SocialInstagram != "" {
		business.SocialInstagram = &input.SocialInstagram
	}
	if input.SocialTiktok != "" {
		business.SocialTiktok = &input.SocialTiktok
	}
	if input.LogoURL != "" {
		business.LogoURL = &input.LogoURL
	}

	// 🪄 Update Field JSONB
	if len(input.BusinessImages) > 0 {
		business.BusinessImages = input.BusinessImages
	}
	if len(input.Certificates) > 0 {
		business.Certificates = input.Certificates
	}
	if len(input.SocialWebsites) > 0 {
		business.SocialWebsites = input.SocialWebsites
	}

	// Update Hours (tetap butuh pointer ke array untuk di-pass ke repo)
	var hoursToUpdate *[]models.BusinessHour
	if len(input.Hours) > 0 {
		var mappedHours []models.BusinessHour
		for _, h := range input.Hours {
			hour := models.BusinessHour{
				DayOfWeek: h.DayOfWeek,
				OpenTime:  toPointerOrNil(h.OpenTime),
				CloseTime: toPointerOrNil(h.CloseTime),
			}
			mappedHours = append(mappedHours, hour)
		}
		hoursToUpdate = &mappedHours
	}

	// Looping Update Websites dan Images manual dihapus

	// Lempar ke repository dengan jumlah parameter yang baru (array JSONB tidak perlu dilempar terpisah)
	// Kita cukup melempar CategoryIDs dan TagIDs karena masih butuh update ke tabel pivot (many-to-many)
	var catIDs *[]int64
	if len(input.CategoryIDs) > 0 {
		catIDs = &input.CategoryIDs
	}
	var tagIDs *[]int64
	if len(input.Tags) > 0 {
		tagIDs = &input.Tags
	}

	// Enkripsi dokumen sensitif sebelum disimpan ke basis data
	encryptBusinessLegal(business)

	if err := s.repo.UpdateBusiness(business, hoursToUpdate, catIDs, tagIDs); err != nil {
		return fmt.Errorf("gagal memperbarui data bisnis di database: %w", err)
	}

	return nil
}

func (s *businessService) DeleteBusiness(ownerID, businessID string) error {
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan atau Anda tidak memiliki akses")
	}

	return s.repo.DeleteBusiness(business)
}

func (s *businessService) ToggleStoreStatus(requesterID, businessID string) (bool, error) {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return false, errors.New("bisnis tidak ditemukan")
	}

	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return false, errors.New("forbidden: Anda tidak memiliki akses untuk mengubah status toko")
		}

		if assignment.Role == "staff" && !permissions.HasPermission(database.DB, requesterID, businessID, models.PermChangeBizHours) {
			return false, errors.New("forbidden: Anda tidak memiliki izin untuk mengubah status operasional toko")
		}
	}

	if len(business.Hours) == 0 {
		database.DB.Where("business_id = ?", businessID).Find(&business.Hours)
	}

	// Nilai baru yang akan di-apply (Flip)
	newStatus := !business.IsManuallyClosed

	// 🚨 LOGIKA VALIDASI: Jika owner mau MEMBUKA toko (newStatus == false)
	if newStatus == false {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		currentDay := int(time.Now().In(loc).Weekday())

		hasValidSchedule := false

		// Cek apakah di array "Hours" ada jadwal untuk hari ini (berarti buka)
		fmt.Printf("DEBUG: currentDay=%d, HoursCount=%d\n", currentDay, len(business.Hours))
		for _, h := range business.Hours {
			fmt.Printf("DEBUG: Checking h.DayOfWeek=%d\n", h.DayOfWeek)
			if h.DayOfWeek == currentDay {
				hasValidSchedule = true
				break
			}
		}

		// Jika hari ini tidak ada jadwalnya (berarti libur)
		if !hasValidSchedule {
			return true, errors.New("Toko tidak bisa dibuka. Jadwal hari ini kosong (toko libur). Silakan perbarui jadwal buka toko Anda terlebih dahulu")
		}
	}

	// Jika validasi aman, simpan status barunya
	business.IsManuallyClosed = newStatus
	if err := s.repo.ToggleStoreStatus(business); err != nil {
		return false, errors.New("gagal memperbarui status toko")
	}

	return business.IsManuallyClosed, nil
}

func formatSocialURL(platform string, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	cleanUser := strings.TrimPrefix(raw, "@")
	switch strings.ToLower(platform) {
	case "tiktok":
		return fmt.Sprintf("https://www.tiktok.com/@%s", cleanUser)
	case "instagram":
		return fmt.Sprintf("https://www.instagram.com/%s", cleanUser)
	case "facebook":
		return fmt.Sprintf("https://www.facebook.com/%s", cleanUser)
	case "youtube":
		return fmt.Sprintf("https://www.youtube.com/@%s", cleanUser)
	default:
		return fmt.Sprintf("https://%s", raw)
	}
}

func (s *businessService) ToggleLiveStatus(requesterID, businessID string, input models.ToggleLiveInput) (*models.LiveStatusResponse, error) {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return nil, errors.New("bisnis tidak ditemukan")
	}

	// 1. Verifikasi hak akses: Owner atau Admin toko yang aktif
	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return nil, errors.New("forbidden: Anda tidak memiliki akses untuk mengelola siaran live toko ini")
		}
	}

	now := time.Now()

	// 2. Jika mengaktifkan Live Streaming (Toggle ON)
	if input.IsLive {
		if input.Platform == nil || strings.TrimSpace(*input.Platform) == "" {
			return nil, errors.New("platform live streaming wajib dipilih (contoh: tiktok, instagram, facebook, youtube)")
		}

		platform := strings.ToLower(strings.TrimSpace(*input.Platform))
		var socialURL string

		// Validasi kelengkapan akun media sosial di profil toko
		switch platform {
		case "tiktok":
			if business.SocialTiktok == nil || strings.TrimSpace(*business.SocialTiktok) == "" {
				return nil, errors.New("silakan lengkapi akun/link TikTok di profil toko Anda terlebih dahulu sebelum mengaktifkan status Live")
			}
			socialURL = formatSocialURL("tiktok", *business.SocialTiktok)
		case "instagram":
			if business.SocialInstagram == nil || strings.TrimSpace(*business.SocialInstagram) == "" {
				return nil, errors.New("silakan lengkapi akun/link Instagram di profil toko Anda terlebih dahulu sebelum mengaktifkan status Live")
			}
			socialURL = formatSocialURL("instagram", *business.SocialInstagram)
		case "facebook":
			found := ""
			for _, w := range business.SocialWebsites {
				if strings.Contains(strings.ToLower(w), "facebook.com") || strings.Contains(strings.ToLower(w), "fb.com") {
					found = w
					break
				}
			}
			if found == "" {
				return nil, errors.New("silakan lengkapi link Facebook di daftar tautan profil toko Anda terlebih dahulu sebelum mengaktifkan status Live")
			}
			socialURL = formatSocialURL("facebook", found)
		case "youtube":
			found := ""
			for _, w := range business.SocialWebsites {
				if strings.Contains(strings.ToLower(w), "youtube.com") || strings.Contains(strings.ToLower(w), "youtu.be") {
					found = w
					break
				}
			}
			if found == "" {
				return nil, errors.New("silakan lengkapi link YouTube di daftar tautan profil toko Anda terlebih dahulu sebelum mengaktifkan status Live")
			}
			socialURL = formatSocialURL("youtube", found)
		default:
			// Fallback jika platform lain
			if len(business.SocialWebsites) > 0 {
				socialURL = business.SocialWebsites[0]
			} else if business.SocialTiktok != nil && *business.SocialTiktok != "" {
				socialURL = formatSocialURL("tiktok", *business.SocialTiktok)
			} else if business.SocialInstagram != nil && *business.SocialInstagram != "" {
				socialURL = formatSocialURL("instagram", *business.SocialInstagram)
			} else {
				return nil, errors.New("silakan lengkapi akun/tautan media sosial di profil toko Anda terlebih dahulu")
			}
		}

		business.IsLive = true
		business.LivePlatform = &platform
		business.LiveStartedAt = &now

		// Poin 1: Cooldown spam notifikasi (15 menit)
		shouldSendNotif := false
		if business.LastLiveNotifiedAt == nil || now.Sub(*business.LastLiveNotifiedAt) > 15*time.Minute {
			shouldSendNotif = true
			business.LastLiveNotifiedAt = &now
		}

		if err := s.repo.UpdateLiveStatus(business); err != nil {
			return nil, errors.New("gagal memperbarui status live toko")
		}

		// Poin 4 & 7: Broadcast notifikasi secara asinkron di background goroutine
		if shouldSendNotif && s.notifService != nil {
			bName := "Toko"
			if business.BusinessName != nil && strings.TrimSpace(*business.BusinessName) != "" {
				bName = strings.TrimSpace(*business.BusinessName)
			}
			platformDisplay := strings.ToUpper(platform[:1]) + platform[1:]
			bizID := business.BusinessID
			reqID := requesterID
			sURL := socialURL

			go func() {
				customerIDs, err := s.repo.GetBookmarkAccountIDs(bizID)
				if err != nil || len(customerIDs) == 0 {
					return
				}

				title := fmt.Sprintf("🔴 %s sedang Live Streaming!", bName)
				body := fmt.Sprintf("Toko %s sedang siaran langsung di %s. Kunjungi profilnya sekarang!", bName, platformDisplay)
				data := map[string]interface{}{
					"business_id":   bizID,
					"business_name": bName,
					"live_platform": platform,
					"social_url":    sURL,
					"screen":        "business_profile",
					"click_action":  "FLUTTER_NOTIFICATION_CLICK",
					"action":        "open_business_profile",
				}

				s.notifService.SendNotification(customerIDs, reqID, "merchant_live", bizID, title, body, data, nil)
			}()
		}

		return &models.LiveStatusResponse{
			BusinessID:    business.BusinessID,
			IsLive:        true,
			LivePlatform:  &platform,
			LiveStartedAt: &now,
			SocialURL:     &socialURL,
		}, nil
	}

	// 3. Jika mematikan Live Streaming (Toggle OFF)
	business.IsLive = false
	business.LivePlatform = nil
	business.LiveStartedAt = nil

	if err := s.repo.UpdateLiveStatus(business); err != nil {
		return nil, errors.New("gagal menonaktifkan status live toko")
	}

	return &models.LiveStatusResponse{
		BusinessID: business.BusinessID,
		IsLive:     false,
	}, nil
}