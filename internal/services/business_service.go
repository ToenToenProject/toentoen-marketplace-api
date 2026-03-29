package services

import (
	"errors"
	"time"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
	"toentoen-backend/pkg/utils"
)

type BusinessService interface {
	CreateBusiness(ownerID string, input models.CreateBusinessInput) (*models.Business, error)
	GetMyBusinesses(ownerID string) ([]models.Business, error)
	UpdateBusiness(ownerID, businessID string, input models.UpdateBusinessInput) (*models.Business, error)
	DeleteBusiness(ownerID, businessID string) error
}

type businessService struct {
	repo repositories.BusinessRepository
}

func NewBusinessService(repo repositories.BusinessRepository) BusinessService {
	return &businessService{repo}
}

func toPointerOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseDatePtr(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse(time.DateOnly, dateStr)
	if err != nil {
		return nil
	}
	return &t
}

func (s *businessService) CreateBusiness(ownerID string, input models.CreateBusinessInput) (*models.Business, error) {
	business := models.Business{
		BusinessID:       utils.GenerateUUID("BUS"),
		AccountID:        ownerID,
		BusinessName:     input.Name,
		Status:           "draft",
		SubscriptionTier: "free",
		Description:      toPointerOrNil(input.Description),
		Province:         toPointerOrNil(input.Province),
		City:             toPointerOrNil(input.City),
		AddressDetail:    toPointerOrNil(input.AddressDetail),
		Address:          toPointerOrNil(input.Address),
		Latitude:         &input.Latitude,
		Longitude:        &input.Longitude,
		StoreType:        toPointerOrNil(input.StoreType),
		StartDate:        parseDatePtr(input.StartDate),
		EndDate:          parseDatePtr(input.EndDate),
		PhoneContact:     toPointerOrNil(input.PhoneContact),
		WhatsappContact:  toPointerOrNil(input.WhatsappContact),
		SocialInstagram:  toPointerOrNil(input.SocialInstagram),
		SocialTiktok:     toPointerOrNil(input.SocialTiktok),
		LogoURL:          toPointerOrNil(input.LogoURL),
		IsActive:         true,
	}

	var hours []models.BusinessHour
	for _, h := range input.Hours {
		hour := models.BusinessHour{DayOfWeek: h.DayOfWeek, IsClosed: h.IsClosed}
		if !h.IsClosed {
			hour.OpenTime = toPointerOrNil(h.OpenTime)
			hour.CloseTime = toPointerOrNil(h.CloseTime)
		}
		hours = append(hours, hour)
	}

	var websites []models.BusinessWebsite
	for _, w := range input.Websites {
		if w != "" {
			websites = append(websites, models.BusinessWebsite{URL: w})
		}
	}

	var images []models.BusinessImage
	for _, img := range input.Images {
		if img != "" {
			images = append(images, models.BusinessImage{ImageURL: img})
		}
	}

	if err := s.repo.CreateBusinessWithRelations(&business, input.CategoryIDs, input.TagIDs, hours, websites, images); err != nil {
		return nil, err
	}

	createdBusiness, err := s.repo.FindBusinessByIDAndOwnerWithPreload(business.BusinessID, ownerID)
	if err != nil {
		return &business, nil
	}

	return createdBusiness, nil
}

func (s *businessService) GetMyBusinesses(ownerID string) ([]models.Business, error) {
	return s.repo.FindBusinessesByOwnerID(ownerID)
}

func (s *businessService) UpdateBusiness(ownerID, businessID string, input models.UpdateBusinessInput) (*models.Business, error) {
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return nil, errors.New("bisnis tidak ditemukan atau Anda tidak memiliki akses")
	}

	if input.Name != nil {
		business.BusinessName = *input.Name
	}
	if input.Handle != nil {
		if s.repo.CheckHandleExists(*input.Handle, business.BusinessID) {
			return nil, errors.New("handle sudah digunakan, silakan pilih yang lain")
		}
		business.Handle = input.Handle
	}

	if input.Description != nil {
		business.Description = input.Description
	}
	if input.Province != nil {
		business.Province = input.Province
	}
	if input.City != nil {
		business.City = input.City
	}
	if input.AddressDetail != nil {
		business.AddressDetail = input.AddressDetail
	}
	if input.Address != nil {
		business.Address = input.Address
	}
	if input.Latitude != nil {
		business.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		business.Longitude = input.Longitude
	}

	if input.StoreType != nil {
		business.StoreType = input.StoreType
	}
	if input.StartDate != nil {
		business.StartDate = parseDatePtr(*input.StartDate)
	}
	if input.EndDate != nil {
		business.EndDate = parseDatePtr(*input.EndDate)
	}

	if input.PhoneContact != nil {
		business.PhoneContact = input.PhoneContact
	}
	if input.WhatsappContact != nil {
		business.WhatsappContact = input.WhatsappContact
	}
	if input.EmailContact != nil {
		business.EmailContact = input.EmailContact
	}
	if input.SocialInstagram != nil {
		business.SocialInstagram = input.SocialInstagram
	}
	if input.SocialTiktok != nil {
		business.SocialTiktok = input.SocialTiktok
	}
	if input.LogoUrl != nil {
		business.LogoURL = input.LogoUrl
	}

	var hoursToUpdate *[]models.BusinessHour
	if input.Hours != nil {
		var mappedHours []models.BusinessHour
		for _, h := range *input.Hours {
			hour := models.BusinessHour{
				DayOfWeek: h.DayOfWeek,
				IsClosed:  h.IsClosed,
			}
			if !h.IsClosed {
				hour.OpenTime = toPointerOrNil(h.OpenTime)
				hour.CloseTime = toPointerOrNil(h.CloseTime)
			}
			mappedHours = append(mappedHours, hour)
		}
		hoursToUpdate = &mappedHours
	}

	var websitesToUpdate *[]models.BusinessWebsite
	if input.Websites != nil {
		var mappedWebsites []models.BusinessWebsite
		for _, w := range *input.Websites {
			if w != "" {
				mappedWebsites = append(mappedWebsites, models.BusinessWebsite{URL: w})
			}
		}
		websitesToUpdate = &mappedWebsites
	}

	var imagesToUpdate *[]models.BusinessImage
	if input.Images != nil {
		var mappedImages []models.BusinessImage
		for _, img := range *input.Images {
			if img != "" {
				mappedImages = append(mappedImages, models.BusinessImage{ImageURL: img})
			}
		}
		imagesToUpdate = &mappedImages
	}

	if err := s.repo.UpdateBusiness(business, hoursToUpdate, input.CategoryIDs, input.TagIDs, websitesToUpdate, imagesToUpdate); err != nil {
		return nil, errors.New("gagal memperbarui data bisnis di database")
	}

	updatedBusiness, err := s.repo.FindBusinessByIDAndOwnerWithPreload(businessID, ownerID)
	if err != nil {
		return business, nil
	}

	return updatedBusiness, nil
}

func (s *businessService) DeleteBusiness(ownerID, businessID string) error {
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan atau Anda tidak memiliki akses")
	}

	return s.repo.DeleteBusiness(business)
}
