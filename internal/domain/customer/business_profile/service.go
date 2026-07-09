package business_profile

import (
	"fmt"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

type BusinessProfileResponse struct {
	BusinessID      string   `json:"business_id"`
	BusinessName    string   `json:"business_name"`
	LogoURL         *string  `json:"logo_url"`
	Category        string   `json:"category"`
	IsOpenNow       bool     `json:"is_open_now"`
	IsBookmarked    bool     `json:"is_bookmarked"`
	
	AddressDetail   *string  `json:"address_detail"`
	City            *string  `json:"city"`
	Province        *string  `json:"province"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`

	PhoneContact    *string  `json:"phone_contact"`
	WhatsappContact *string  `json:"whatsapp_contact"`
	SocialInstagram *string  `json:"social_instagram"`
	SocialTiktok    *string  `json:"social_tiktok"`
	SocialWebsites  []string `json:"social_websites"`

	BusinessImages  []string `json:"business_images"`

	Hours []models.BusinessHour `json:"hours"`
}

type Service interface {
	GetBusinessProfile(businessID, accountID string) (BusinessProfileResponse, error)
	ToggleBookmark(accountID, businessID string) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{
		repo: r,
	}
}

func (s *service) GetBusinessProfile(businessID, accountID string) (BusinessProfileResponse, error) {
	business, err := s.repo.GetBusinessDetail(businessID)
	if err != nil {
		return BusinessProfileResponse{}, err
	}

	var isBookmarked bool
	if accountID != "" {
		isBookmarked, _ = s.repo.CheckBookmark(accountID, businessID)
	}

	// Tentukan Kategori Utama (Ambil yang pertama jika ada)
	category := "General"
	if len(business.Categories) > 0 {
		category = business.Categories[0].Name
	}

	// Hitung IsOpenNow
	isOpenNow := false
	if !business.IsManuallyClosed {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		now := time.Now().In(loc)
		currentDayInt := int(now.Weekday())

		for _, h := range business.Hours {
			if h.DayOfWeek == currentDayInt {
				if !h.IsClosed && h.OpenTime != nil && h.CloseTime != nil {
					openT, _ := time.Parse("15:04", *h.OpenTime)
					closeT, _ := time.Parse("15:04", *h.CloseTime)
					nowT, _ := time.Parse("15:04", fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute()))

					if (nowT.After(openT) || nowT.Equal(openT)) && nowT.Before(closeT) {
						isOpenNow = true
					}
				}
				break
			}
		}
	}

	bName := ""
	if business.BusinessName != nil {
		bName = *business.BusinessName
	}

	return BusinessProfileResponse{
		BusinessID:      business.BusinessID,
		BusinessName:    bName,
		LogoURL:         business.LogoURL,
		Category:        category,
		IsOpenNow:       isOpenNow,
		IsBookmarked:    isBookmarked,
		AddressDetail:   business.AddressDetail,
		City:            business.City,
		Province:        business.Province,
		Latitude:        business.Latitude,
		Longitude:       business.Longitude,
		PhoneContact:    business.PhoneContact,
		WhatsappContact: business.WhatsappContact,
		SocialInstagram: business.SocialInstagram,
		SocialTiktok:    business.SocialTiktok,
		SocialWebsites:  business.SocialWebsites,
		BusinessImages:  business.BusinessImages,
		Hours:           business.Hours,
	}, nil
}

func (s *service) ToggleBookmark(accountID, businessID string) (bool, error) {
	// Verifikasi toko exist
	_, err := s.repo.GetBusinessDetail(businessID)
	if err != nil {
		return false, err
	}
	return s.repo.ToggleBookmark(accountID, businessID)
}
