package business

import (
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type BusinessRepository interface {
	// Parameter websites dan images dihapus karena sudah otomatis masuk di *models.Business (JSONB)
	CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour) error

	FindBusinessesByOwnerID(ownerID string) ([]models.Business, error)
	FindBusinessByIDAndOwner(businessID, ownerID string) (*models.Business, error)
	CheckHandleExists(handle, excludeBusinessID string) bool
	
	// Parameter newWebsites dan newImages dihapus
	UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64) error
	
	DeleteBusiness(business *models.Business) error
	FindBusinessByIDAndOwnerWithPreload(businessID, ownerID string) (*models.Business, error)
	CheckBusinessNameExists(ownerID, name string) bool
	GetSpecificBusiness(accountID string, businessID string) (*models.Business, error)
	FindBusinessByID(businessID string) (*models.Business, error)
	ToggleStoreStatus(business *models.Business) error
	ResetManualCloseStatus() error
	ConfirmOpenToday(businessID string, todayDate string) error
	ConfirmOvertimeToday(businessID string, todayDate string) error
	DeclineSchedule(businessID string) error
	GetBusinessesWithHours() ([]models.Business, error)
	UpdateBusinessRating(businessID string, avgRating float64, totalReviews int) error
	GetBusinessListWithPreviews(accountID string, role string, search string, categoryID int64, tagIDs []int64, cursor string, limit int) ([]models.BusinessWithPreview, error)
	GetBusinessStaffAccountIDs(businessID string) ([]string, error)
	UpdateLiveStatus(business *models.Business) error
	GetBookmarkAccountIDs(businessID string) ([]string, error)
	ResetLiveStatus() error
	GetOwnerKYCStatus(ownerID string) (emailVerified bool, kycStatus string, err error)
	HasNIBTag(tagIDs []int64) (bool, error)
	RemoveNIBTagFromBusiness(businessID string) error
	CreateNameHistory(history *models.BusinessNameHistory) error
}

type businessRepository struct {
	db *gorm.DB
}

func NewBusinessRepository(db *gorm.DB) BusinessRepository {
	return &businessRepository{db}
}

func (r *businessRepository) CreateBusinessWithRelations(business *models.Business, categoryIDs []int64, tagIDs []int64, hours []models.BusinessHour) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create Business (Otomatis menyimpan kolom Photos dan SocialWebsites karena JSONB)
		if err := tx.Create(business).Error; err != nil {
			return err
		}

		// 2. Simpan Relasi Business Hours
		if len(hours) > 0 {
			for i := range hours {
				hours[i].BusinessID = business.BusinessID
			}
			if err := tx.Create(&hours).Error; err != nil {
				return err
			}
		}

		// 3. Simpan Relasi Pivot Categories (Manual)
		if len(categoryIDs) > 0 {
			// Anda bisa juga menggunakan tx.Model(business).Association("Categories").Append(...)
			// Tapi cara manual ini juga sangat aman dan valid.
			var businessCategories []map[string]interface{}
			for _, catID := range categoryIDs {
				businessCategories = append(businessCategories, map[string]interface{}{
					"business_id": business.BusinessID,
					"category_id": catID,
				})
			}
			if err := tx.Table("business_has_categories").Create(&businessCategories).Error; err != nil {
				return err
			}
		}

		// 4. Simpan Relasi Pivot Tags (Manual)
		if len(tagIDs) > 0 {
			var businessTags []map[string]interface{}
			for _, tagID := range tagIDs {
				businessTags = append(businessTags, map[string]interface{}{
					"business_id": business.BusinessID,
					"tag_id":      tagID,
				})
			}
			if err := tx.Table("business_has_tags").Create(&businessTags).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *businessRepository) FindBusinessesByOwnerID(ownerID string) ([]models.Business, error) {
	var businesses []models.Business
	err := r.db.
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		// Preload("Websites") & Preload("Images") dihapus karena tidak perlu
		Where("account_id = ?", ownerID).
		Find(&businesses).Error
	return businesses, err
}

func (r *businessRepository) FindBusinessByIDAndOwner(businessID, ownerID string) (*models.Business, error) {
	var business models.Business
	err := r.db.Where("business_id = ? AND account_id = ?", businessID, ownerID).First(&business).Error
	return &business, err
}

func (r *businessRepository) FindBusinessByIDAndOwnerWithPreload(businessID, ownerID string) (*models.Business, error) {
	var business models.Business
	err := r.db.
		Preload("Hours").
		Preload("Categories").
		Preload("Tags").
		// Preload("Websites") & Preload("Images") dihapus karena tidak perlu
		Where("business_id = ? AND account_id = ?", businessID, ownerID).
		First(&business).Error

	return &business, err
}

func (r *businessRepository) CheckHandleExists(handle, excludeBusinessID string) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("handle = ? AND business_id != ?", handle, excludeBusinessID).Count(&count)
	return count > 0
}

func (r *businessRepository) UpdateBusiness(business *models.Business, newHours *[]models.BusinessHour, newCategoryIDs *[]int64, newTagIDs *[]int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// tx.Omit mencegah GORM meng-upsert relasi yang ter-preload yang bisa memicu conflict key
		if err := tx.Omit("Hours", "Categories", "Tags").Save(business).Error; err != nil {
			return err
		}

		if newHours != nil {
			if err := tx.Where("business_id = ?", business.BusinessID).Delete(&models.BusinessHour{}).Error; err != nil {
				return err
			}
			if len(*newHours) > 0 {
				hours := *newHours
				for i := range hours {
					hours[i].BusinessID = business.BusinessID
				}
				if err := tx.Create(&hours).Error; err != nil {
					return err
				}
			}
		}

		// Blok delete & recreate newWebsites dan newImages telah dihapus

		if newCategoryIDs != nil {
			if err := tx.Exec("DELETE FROM business_has_categories WHERE business_id = ?", business.BusinessID).Error; err != nil {
				return err
			}
			if len(*newCategoryIDs) > 0 {
				var businessCategories []map[string]interface{}
				for _, catID := range *newCategoryIDs {
					businessCategories = append(businessCategories, map[string]interface{}{
						"business_id": business.BusinessID,
						"category_id": catID,
					})
				}
				if err := tx.Table("business_has_categories").Create(&businessCategories).Error; err != nil {
					return err
				}
			}
		}

		if newTagIDs != nil {
			if err := tx.Exec("DELETE FROM business_has_tags WHERE business_id = ?", business.BusinessID).Error; err != nil {
				return err
			}
			if len(*newTagIDs) > 0 {
				var businessTags []map[string]interface{}
				for _, tagID := range *newTagIDs {
					businessTags = append(businessTags, map[string]interface{}{
						"business_id": business.BusinessID,
						"tag_id":      tagID,
					})
				}
				if err := tx.Table("business_has_tags").Create(&businessTags).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (r *businessRepository) DeleteBusiness(business *models.Business) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Delete(business).Error
	})
}

func (r *businessRepository) CheckBusinessNameExists(ownerID, name string) bool {
	var count int64
	// PERBAIKAN: name diubah menjadi business_name agar sesuai dengan kolom SQL
	r.db.Model(&models.Business{}).Where("account_id = ? AND business_name = ?", ownerID, name).Count(&count)
	return count > 0
}

func (r *businessRepository) GetSpecificBusiness(accountID string, businessID string) (*models.Business, error) {
	var business models.Business

	err := r.db.
		Preload("Categories").
		Where("account_id = ? AND business_id = ?", accountID, businessID).
		First(&business).Error

	if err != nil {
		return nil, err
	}

	return &business, nil
}

func (r *businessRepository) FindBusinessByID(businessID string) (*models.Business, error) {
	var business models.Business
	
	// Gunakan Preload agar array data pendukung ikut terambil (bukan cuma null)
	err := r.db.
		Preload("Categories").
		Preload("Tags").
		Preload("Hours").
		Where("business_id = ?", businessID).
		First(&business).Error

	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *businessRepository) ToggleStoreStatus(business *models.Business) error {
	// Update kolom status operasional di database menggunakan Raw SQL agar GORM tidak mencoba save relasi (ON CONFLICT)
	if !business.IsManuallyClosed {
		return r.db.Exec(`
			UPDATE businesses 
			SET is_manually_closed = false, 
			    confirmed_open_date = ?, 
			    confirmed_overtime_date = ? 
			WHERE business_id = ?`,
			business.ConfirmedOpenDate, business.ConfirmedOvertimeDate, business.BusinessID,
		).Error
	}
	return r.db.Exec("UPDATE businesses SET is_manually_closed = true WHERE business_id = ?", business.BusinessID).Error
}

func (r *businessRepository) ResetManualCloseStatus() error {
	// Otomatis tutup toko di tengah malam (00:00 WIB) jika merchant lupa mematikan toggle (is_manually_closed = false -> true)
	return r.db.Exec("UPDATE businesses SET is_manually_closed = true WHERE is_manually_closed = false AND status != 'draft'").Error
}

func (r *businessRepository) ConfirmOpenToday(businessID string, todayDate string) error {
	return r.db.Exec("UPDATE businesses SET confirmed_open_date = ?, is_manually_closed = false WHERE business_id = ?", todayDate, businessID).Error
}

func (r *businessRepository) ConfirmOvertimeToday(businessID string, todayDate string) error {
	return r.db.Exec("UPDATE businesses SET confirmed_open_date = ?, confirmed_overtime_date = ?, is_manually_closed = false WHERE business_id = ?", todayDate, todayDate, businessID).Error
}

func (r *businessRepository) DeclineSchedule(businessID string) error {
	return r.db.Exec("UPDATE businesses SET is_manually_closed = true WHERE business_id = ?", businessID).Error
}

func (r *businessRepository) GetBusinessesWithHours() ([]models.Business, error) {
	var businesses []models.Business
	err := r.db.Model(&models.Business{}).
		Preload("Hours").
		Where("status != 'draft' AND is_active = true").
		Find(&businesses).Error
	return businesses, err
}

// Di repositories/business_repository.go
func (r *businessRepository) UpdateBusinessRating(businessID string, avgRating float64, totalReviews int) error {
	return r.db.Model(&models.Business{}).Where("business_id = ?", businessID).Updates(map[string]interface{}{
		"average_rating": avgRating,
		"review_count":   totalReviews,
	}).Error
}

// FUNGSI BARU UNTUK MENDUKUNG UI LIST
func (r *businessRepository) GetBusinessListWithPreviews(accountID string, role string, search string, categoryID int64, tagIDs []int64, cursor string, limit int) ([]models.BusinessWithPreview, error) {
	var businesses []models.Business
	
	// Memulai query dasar
	query := r.db.Model(&models.Business{})
	if role == "business_admin" {
		query = query.Where("business_id IN (SELECT business_id FROM business_admin_assignments WHERE admin_id = ? AND status IN ('active', 'scheduled_for_deletion'))", accountID)
	} else {
		query = query.Where("account_id = ?", accountID)
	}
	
	// --- 1. LOGIKA PENCARIAN (Search) ---
	if search != "" {
		// Menggunakan ILIKE agar pencarian tidak case-sensitive (huruf besar/kecil tidak masalah)
		query = query.Where("business_name ILIKE ?", "%"+search+"%")
	}

	// --- 2. LOGIKA FILTER KATEGORI ---
	if categoryID > 0 {
		// Subquery: Cari ID bisnis yang punya kategori ini di tabel pivot
		query = query.Where("business_id IN (SELECT business_id FROM business_has_categories WHERE category_id = ?)", categoryID)
	}

	// --- 3. LOGIKA FILTER TAGS ---
	if len(tagIDs) > 0 {
		// Subquery: Cari ID bisnis yang punya MINIMAL SALAH SATU tag yang dikirim (OR logic)
		query = query.Where("business_id IN (SELECT business_id FROM business_has_tags WHERE tag_id IN ?)", tagIDs)
	}
	
	// --- 4. LOGIKA KURSOR & SORTING ---
	if cursor != "" {
		var cursorBusiness models.Business
		if err := r.db.Select("created_at").Where("business_id = ?", cursor).First(&cursorBusiness).Error; err == nil {
			query = query.Where("created_at < ? OR (created_at = ? AND business_id < ?)", cursorBusiness.CreatedAt, cursorBusiness.CreatedAt, cursor)
		}
	}
	
	err := query.Preload("Hours").Order("created_at DESC, business_id DESC").Limit(limit).Find(&businesses).Error
	if err != nil {
		return nil, err
	}

	if len(businesses) == 0 {
		return []models.BusinessWithPreview{}, nil
	}

	var businessIDs []string
	for _, b := range businesses {
		businessIDs = append(businessIDs, b.BusinessID)
	}

	var accountName string
	r.db.Table("accounts").Select("full_name").Where("account_id = ?", accountID).Scan(&accountName) 

	// Ambil Rating Rata-rata dari Seluruh Review Produk Toko
	type ProductRatingAgg struct {
		BusinessID   string
		AvgRating    float64
		TotalReviews int64
	}
	var ratings []ProductRatingAgg

	err = r.db.Raw(`
		SELECT 
			pr.business_id,
			COALESCE(AVG(pr.rating), 0) as avg_rating,
			COUNT(pr.review_id) as total_reviews
		FROM product_reviews pr
		WHERE pr.business_id IN ? AND pr.rating > 0
		GROUP BY pr.business_id
	`, businessIDs).Scan(&ratings).Error

	ratingMap := make(map[string]ProductRatingAgg)
	for _, r := range ratings {
		ratingMap[r.BusinessID] = r
	}

	var result []models.BusinessWithPreview
	for _, b := range businesses {
		agg := ratingMap[b.BusinessID]
		roundedAvg := float64(int(agg.AvgRating*10)) / 10
		result = append(result, models.BusinessWithPreview{
			Business:            b,
			OwnerName:           accountName,
			AverageRating:       roundedAvg,
			TotalProductReviews: agg.TotalReviews,
		})
	}

	return result, nil
}

func (r *businessRepository) GetBusinessStaffAccountIDs(businessID string) ([]string, error) {
	var accountIDs []string

	// 1. Ambil owner (AccountID di tabel businesses)
	var business models.Business
	if err := r.db.Select("account_id").Where("business_id = ?", businessID).First(&business).Error; err != nil {
		return nil, err
	}
	accountIDs = append(accountIDs, business.AccountID)

	// 2. Ambil admins (admin_id di tabel business_admin_assignments)
	var assignments []models.BusinessAdminAssignment
	if err := r.db.Select("admin_id").Where("business_id = ? AND status IN ('accepted', 'active', 'scheduled_for_deletion')", businessID).Find(&assignments).Error; err == nil {
		for _, a := range assignments {
			accountIDs = append(accountIDs, a.AdminID)
		}
	}

	return accountIDs, nil
}

func (r *businessRepository) UpdateLiveStatus(business *models.Business) error {
	return r.db.Exec(`
		UPDATE businesses 
		SET is_live = ?, 
		    live_platform = ?, 
		    live_started_at = ?, 
		    last_live_notified_at = ?, 
		    updated_at = ? 
		WHERE business_id = ?`,
		business.IsLive,
		business.LivePlatform,
		business.LiveStartedAt,
		business.LastLiveNotifiedAt,
		time.Now(),
		business.BusinessID,
	).Error
}

func (r *businessRepository) GetBookmarkAccountIDs(businessID string) ([]string, error) {
	var accountIDs []string
	err := r.db.Table("business_bookmarks").
		Where("business_id = ?", businessID).
		Pluck("account_id", &accountIDs).Error
	return accountIDs, err
}

func (r *businessRepository) ResetLiveStatus() error {
	return r.db.Model(&models.Business{}).
		Where("is_live = ?", true).
		Updates(map[string]interface{}{
			"is_live":         false,
			"live_platform":   nil,
			"live_started_at": nil,
			"updated_at":      time.Now(),
		}).Error
}

func (r *businessRepository) GetOwnerKYCStatus(ownerID string) (bool, string, error) {
	var account models.Account
	if err := r.db.Select("email_verified_at").Where("account_id = ?", ownerID).First(&account).Error; err != nil {
		return false, "", err
	}
	emailVerified := account.EmailVerifiedAt != nil

	var owner models.Owner
	if err := r.db.Select("kyc_status").Where("account_id = ?", ownerID).First(&owner).Error; err != nil {
		return emailVerified, "unverified", nil
	}

	kycStatus := owner.KycStatus
	if kycStatus == "" {
		kycStatus = "unverified"
	}

	return emailVerified, kycStatus, nil
}

func (r *businessRepository) HasNIBTag(tagIDs []int64) (bool, error) {
	if len(tagIDs) == 0 {
		return false, nil
	}
	var count int64
	err := r.db.Model(&models.Tag{}).
		Where("tag_id IN ? AND (LOWER(name) LIKE ? OR LOWER(name) LIKE ?)", tagIDs, "%(nib)%", "%terverifikasi%nib%").
		Count(&count).Error
	return count > 0, err
}

func (r *businessRepository) RemoveNIBTagFromBusiness(businessID string) error {
	var nibTagIDs []int64
	if err := r.db.Model(&models.Tag{}).
		Where("LOWER(name) LIKE ? OR LOWER(name) LIKE ?", "%(nib)%", "%terverifikasi%nib%").
		Pluck("tag_id", &nibTagIDs).Error; err != nil || len(nibTagIDs) == 0 {
		return err
	}
	return r.db.Table("business_has_tags").
		Where("business_id = ? AND tag_id IN ?", businessID, nibTagIDs).
		Delete(nil).Error
}

func (r *businessRepository) CreateNameHistory(history *models.BusinessNameHistory) error {
	return r.db.Create(history).Error
}