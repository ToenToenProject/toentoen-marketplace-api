package business

import (
	"errors"
	"fmt"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"
)

type BusinessService interface {
	CreateBusiness(ownerID string, input models.CreateBusinessInput) (string, error)
	GetMyBusinesses(ownerID, search string, categoryID int64, tagIDs []int64, cursor string, limit int) (*models.GetBusinessListResponse, error)
	GetBusinessDetail(businessID string) (*models.Business, error)
	UpdateBusiness(ownerID, businessID string, input models.UpdateBusinessInput) (error)
	DeleteBusiness(ownerID, businessID string) error
	ToggleStoreStatus(ownerID, businessID string) (bool, error)
}

type businessService struct {
	repo BusinessRepository
}

func NewBusinessService(repo BusinessRepository) BusinessService {
	return &businessService{repo}
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
		return nil, errors.New("failed to retrieve business list")
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
			BusinessID:    b.BusinessID,
			BusinessName:  *b.BusinessName,
			OwnerName:     b.OwnerName,
			LogoURL:       mainImage,
			Address:       fullAddress,
			PreviewImages: b.PreviewImages,
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
		return nil, errors.New("business not found")
	}

	// 🪄 MAGIC: Hitung is_open_now secara live sebelum dikembalikan ke Frontend
	business.IsOpenNow = utils.CalculateStoreStatus(business.IsManuallyClosed, business.Hours)

	return business, nil
}

func (s *businessService) UpdateBusiness(requesterID, businessID string, input models.UpdateBusinessInput) (error) {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return errors.New("business is not found")
	}

	// Verifikasi apakah requester adalah owner atau admin aktif di toko ini
	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return errors.New("you don't have access to update this business")
		}

		if assignment.Role == "staff" && (len(input.Hours) > 0 || len(input.EventSchedules) > 0) {
			if !permissions.HasPermission(database.DB, requesterID, businessID, models.PermChangeBizHours) {
				return errors.New("you don't have permission to change business hours")
			}
		}
	}

	// Update field sederhana
	if input.BusinessName != "" {
		business.BusinessName = &input.BusinessName
	}
	if input.Handle != "" {
		if s.repo.CheckHandleExists(input.Handle, business.BusinessID) {
			return errors.New("handle already exists, please choose another one")
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

	if err := s.repo.UpdateBusiness(business, hoursToUpdate, catIDs, tagIDs); err != nil {
		return fmt.Errorf("failed to update business data in the database: %w", err)
	}

	return nil
}

func (s *businessService) DeleteBusiness(ownerID, businessID string) error {
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("business is not found or you don't have access")
	}

	return s.repo.DeleteBusiness(business)
}

func (s *businessService) ToggleStoreStatus(requesterID, businessID string) (bool, error) {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return false, errors.New("business is not found")
	}

	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return false, errors.New("you don't have access to toggle store status")
		}

		if assignment.Role == "staff" && !permissions.HasPermission(database.DB, requesterID, businessID, models.PermChangeBizHours) {
			return false, errors.New("you don't have permission to change store operational status")
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
		return false, errors.New("failed to update store status")
	}

	return business.IsManuallyClosed, nil
}