package services

import (
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
	"errors"
)

// Pastikan interface BusinessService di-update menjadi:
type BusinessService interface {
	CreateBusiness(ownerID int64, input models.CreateBusinessInput) error
	
	// Tambahan baru:
	GetMyBusinesses(ownerID int64) ([]models.Business, error)
	UpdateBusiness(ownerID, businessID int64, input models.UpdateBusinessInput) (*models.Business, error)
	DeleteBusiness(ownerID, businessID int64) error
}

type businessService struct {
	repo repositories.BusinessRepository
}

func NewBusinessService(repo repositories.BusinessRepository) BusinessService {
	return &businessService{repo}
}

// Helper kecil untuk mengubah string kosong menjadi nil (agar rapi di database)
func toPointerOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *businessService) CreateBusiness(ownerID int64, input models.CreateBusinessInput) error {
	// 1. Mapping data bisnis utama
	business := models.Business{
		AccountID:        ownerID,
		Name:             input.Name,
		Status:           "draft", // Ubah jadi 'draft' jika butuh verifikasi admin dulu
		SubscriptionTier: "free",
		Description:      toPointerOrNil(input.Description),
		Address:          toPointerOrNil(input.Address),
		Latitude:         &input.Latitude,
		Longitude:        &input.Longitude,
		PhoneContact:     toPointerOrNil(input.PhoneContact),
		WhatsappContact:  toPointerOrNil(input.WhatsappContact),
		EmailContact:     toPointerOrNil(input.EmailContact),
		WebsiteURL:       toPointerOrNil(input.WebsiteURL),
		SocialInstagram:  toPointerOrNil(input.SocialInstagram),
		SocialTiktok:     toPointerOrNil(input.SocialTiktok),
		LogoURL:          toPointerOrNil(input.LogoURL),
		IsActive:         true,
	}

	// 2. Mapping Jam Operasional
	var hours []models.BusinessHour
	for _, h := range input.Hours {
		hour := models.BusinessHour{
			DayOfWeek: h.DayOfWeek,
			IsClosed:  h.IsClosed,
		}
		
		// Jika toko tidak tutup, masukkan jamnya
		if !h.IsClosed {
			hour.OpenTime = toPointerOrNil(h.OpenTime)
			hour.CloseTime = toPointerOrNil(h.CloseTime)
		}
		hours = append(hours, hour)
	}

	// 3. Eksekusi ke Repository
	return s.repo.CreateBusinessWithRelations(&business, input.CategoryIDs, input.TagIDs, hours)
}

// === IMPLEMENTASI FUNGSI BARU ===

func (s *businessService) GetMyBusinesses(ownerID int64) ([]models.Business, error) {
	return s.repo.FindBusinessesByOwnerID(ownerID)
}

func (s *businessService) UpdateBusiness(ownerID, businessID int64, input models.UpdateBusinessInput) (*models.Business, error) {
	// 1. Cek kepemilikan bisnis
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return nil, errors.New("bisnis tidak ditemukan atau Anda tidak memiliki akses")
	}

	// 2. Terapkan update (Hanya update field yang tidak nil)
	if input.Name != nil { business.Name = *input.Name }
	if input.Handle != nil {
		// Cek apakah handle baru sudah dipakai orang lain
		if s.repo.CheckHandleExists(*input.Handle, business.BusinessID) {
			return nil, errors.New("handle sudah digunakan, silakan pilih yang lain")
		}
		business.Handle = input.Handle
	}

	if input.Description != nil { business.Description = input.Description }
	if input.Address != nil { business.Address = input.Address }
	if input.Latitude != nil { business.Latitude = input.Latitude }
	if input.Longitude != nil { business.Longitude = input.Longitude }
	if input.PhoneContact != nil { business.PhoneContact = input.PhoneContact }
	if input.WhatsappContact != nil { business.WhatsappContact = input.WhatsappContact }
	if input.EmailContact != nil { business.EmailContact = input.EmailContact }
	if input.WebsiteUrl != nil { business.WebsiteURL = input.WebsiteUrl }
	if input.SocialInstagram != nil { business.SocialInstagram = input.SocialInstagram }
	if input.SocialTiktok != nil { business.SocialTiktok = input.SocialTiktok }
	if input.LogoUrl != nil { business.LogoURL = input.LogoUrl }

	// 3. Mapping Jam Operasional jika ada
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

	// 4. Panggil repo dengan parameter tambahan
	if err := s.repo.UpdateBusiness(business, hoursToUpdate, input.CategoryIDs, input.TagIDs); err != nil {
		return nil, errors.New("gagal memperbarui data bisnis di database")
	}

	// 5. SOLUSI: Ambil ulang menggunakan Preload agar relasi ikut terbawa
	updatedBusiness, err := s.repo.FindBusinessByIDAndOwnerWithPreload(businessID, ownerID)
	if err != nil {
		// Jika terjadi error saat mengambil ulang, kembalikan objek lama sebagai fallback
		return business, nil
	}

	return updatedBusiness, nil
}

func (s *businessService) DeleteBusiness(ownerID, businessID int64) error {
	// Cek kepemilikan dulu sebelum menghapus
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan atau Anda tidak memiliki akses")
	}

	return s.repo.DeleteBusiness(business)
}