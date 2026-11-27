package business

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/kychelper"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"
)

func encryptBusinessLegal(b *models.Business) {
	if b == nil {
		return
	}
	b.NIB, _ = crypto.EncryptPointer(b.NIB)
	for i := range b.Certificates {
		// Normalize primary fields from aliases if empty.
		if b.Certificates[i].Type == "" && b.Certificates[i].Name != "" {
			b.Certificates[i].Type = b.Certificates[i].Name
		}
		if b.Certificates[i].Number == "" && b.Certificates[i].CertificateNumber != "" {
			b.Certificates[i].Number = b.Certificates[i].CertificateNumber
		}
		if b.Certificates[i].ImageURL == "" && b.Certificates[i].URL != "" {
			b.Certificates[i].ImageURL = b.Certificates[i].URL
		} else if b.Certificates[i].ImageURL == "" && b.Certificates[i].CertificateImageURL != "" {
			b.Certificates[i].ImageURL = b.Certificates[i].CertificateImageURL
		}

		if b.Certificates[i].Number != "" {
			b.Certificates[i].Number, _ = crypto.Encrypt(b.Certificates[i].Number)
		}
		if b.Certificates[i].ImageURL != "" {
			b.Certificates[i].ImageURL, _ = crypto.Encrypt(b.Certificates[i].ImageURL)
		}

		// Synchronize aliases to prevent storing plaintext in JSONB columns.
		b.Certificates[i].Name = b.Certificates[i].Type
		b.Certificates[i].CertificateName = b.Certificates[i].Type
		b.Certificates[i].CertificateType = b.Certificates[i].Type
		b.Certificates[i].CertificateNumber = b.Certificates[i].Number
		b.Certificates[i].URL = b.Certificates[i].ImageURL
		b.Certificates[i].CertificateImageURL = b.Certificates[i].ImageURL
	}
}

func decryptBusinessLegal(b *models.Business) {
	if b == nil {
		return
	}
	b.NIB, _ = crypto.DecryptPointer(b.NIB)
	for i := range b.Certificates {
		// Fallback recovery for values stored in alias fields.
		if b.Certificates[i].Type == "" && b.Certificates[i].Name != "" {
			b.Certificates[i].Type = b.Certificates[i].Name
		}
		if b.Certificates[i].Number == "" && b.Certificates[i].CertificateNumber != "" {
			b.Certificates[i].Number = b.Certificates[i].CertificateNumber
		}
		if b.Certificates[i].ImageURL == "" && b.Certificates[i].URL != "" {
			b.Certificates[i].ImageURL = b.Certificates[i].URL
		} else if b.Certificates[i].ImageURL == "" && b.Certificates[i].CertificateImageURL != "" {
			b.Certificates[i].ImageURL = b.Certificates[i].CertificateImageURL
		}

		if b.Certificates[i].Number != "" {
			b.Certificates[i].Number, _ = crypto.Decrypt(b.Certificates[i].Number)
		}
		if b.Certificates[i].ImageURL != "" {
			b.Certificates[i].ImageURL, _ = crypto.Decrypt(b.Certificates[i].ImageURL)
		}

		// Populate aliases to ensure frontend response compatibility.
		b.Certificates[i].Name = b.Certificates[i].Type
		b.Certificates[i].CertificateName = b.Certificates[i].Type
		b.Certificates[i].CertificateType = b.Certificates[i].Type
		b.Certificates[i].CertificateNumber = b.Certificates[i].Number
		b.Certificates[i].URL = b.Certificates[i].ImageURL
		b.Certificates[i].CertificateImageURL = b.Certificates[i].ImageURL
	}
}

type BusinessService interface {
	CreateBusiness(ownerID string, input models.CreateBusinessInput) (string, error)
	GetMyBusinesses(accountID, userRole, search string, categoryID int64, tagIDs []int64, cursor string, limit int) (*models.GetBusinessListResponse, error)
	GetBusinessDetail(businessID string) (*models.Business, error)
	UpdateBusiness(ownerID, businessID string, input models.UpdateBusinessInput) (error)
	DeleteBusiness(ownerID, businessID string) error
	ToggleStoreStatus(ownerID, businessID string) (bool, error)
	ConfirmScheduleAction(requesterID, businessID string, input models.ConfirmScheduleInput) error
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
	cleanDate := strings.TrimSpace(dateStr)
	if cleanDate == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		time.DateOnly,
		"2006-01-02 15:04:05",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, cleanDate); err == nil {
			return &t
		}
	}

	return nil
}

func deduplicateInt64(slice []int64) []int64 {
	seen := make(map[int64]bool, len(slice))
	result := make([]int64, 0, len(slice))
	for _, val := range slice {
		if !seen[val] && val > 0 {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

type tagMetaRow struct {
	TagID     int64  `gorm:"column:tag_id"`
	ParentID  *int64 `gorm:"column:parent_id"`
	GroupSlug string `gorm:"column:group_slug"`
}

func resolveAndValidateCategoryTags(categoryIDs []int64, flatTags []int64, categoryTags []models.CategoryTagSelection) ([]int64, []int64, []models.BusinessHasTag, error) {
	if len(categoryTags) > 0 {
		if len(categoryTags) > 3 {
			return nil, nil, nil, errors.New("maksimal 3 kategori yang dapat dipilih")
		}

		seenCats := make(map[int64]bool, len(categoryTags))
		resolvedCatIDs := make([]int64, 0, len(categoryTags))
		allTagSet := make(map[int64]bool)
		var flatTagIDs []int64
		var pivotRows []models.BusinessHasTag

		for _, sel := range categoryTags {
			if sel.CategoryID <= 0 {
				continue
			}
			if seenCats[sel.CategoryID] {
				return nil, nil, nil, errors.New("duplikasi kategori tidak diperbolehkan")
			}
			seenCats[sel.CategoryID] = true
			resolvedCatIDs = append(resolvedCatIDs, sel.CategoryID)

			uniqueTags := deduplicateInt64(sel.TagIDs)
			if len(uniqueTags) == 0 {
				continue
			}

			finalTagSet := make(map[int64]bool, len(uniqueTags))
			if database.DB != nil {
				var activeCatCount int64
				if err := database.DB.Model(&models.Category{}).Where("category_id = ? AND is_active = ?", sel.CategoryID, true).Count(&activeCatCount).Error; err == nil && activeCatCount == 0 {
					return nil, nil, nil, fmt.Errorf("kategori %d tidak ditemukan atau sudah tidak aktif", sel.CategoryID)
				}

				var rows []tagMetaRow
				err := database.DB.Table("business_tags bt").
					Select("bt.tag_id, bt.parent_id, COALESCE(tg.slug, '') AS group_slug").
					Joins("LEFT JOIN tag_groups tg ON tg.group_id = bt.group_id").
					Joins("JOIN category_tag_rules ctr ON ctr.tag_id = bt.tag_id AND ctr.category_id = ? AND ctr.is_active = true", sel.CategoryID).
					Where("bt.tag_id IN ? AND bt.is_active = true", uniqueTags).
					Find(&rows).Error
				if err == nil && len(rows) > 0 {
					if len(rows) != len(uniqueTags) {
						return nil, nil, nil, fmt.Errorf("satu atau lebih tag tidak valid untuk kategori %d", sel.CategoryID)
					}
					quota := 0
					hasPayment := false
					for _, r := range rows {
						finalTagSet[r.TagID] = true
						// Auto-include parent tag (e.g. E-wallet accepted, Pet friendly) without consuming extra quota.
						if r.ParentID != nil && *r.ParentID > 0 {
							finalTagSet[*r.ParentID] = true
						}
						if r.GroupSlug == "payment-transaction" {
							if !hasPayment {
								hasPayment = true
								quota++
							}
						} else {
							quota++
						}
					}
					if quota > 5 {
						return nil, nil, nil, errors.New("maksimal 5 tag per kategori (metode pembayaran dihitung 1 tag)")
					}
				} else {
					// Fallback when category_tag_rules is not seeded (e.g. unit tests)
					if len(uniqueTags) > 5 {
						return nil, nil, nil, errors.New("maksimal 5 tag per kategori")
					}
					for _, tid := range uniqueTags {
						finalTagSet[tid] = true
					}
				}
			} else {
				if len(uniqueTags) > 5 {
					return nil, nil, nil, errors.New("maksimal 5 tag per kategori")
				}
				for _, tid := range uniqueTags {
					finalTagSet[tid] = true
				}
			}

			for tid := range finalTagSet {
				pivotRows = append(pivotRows, models.BusinessHasTag{
					CategoryID: sel.CategoryID,
					TagID:      tid,
				})
				if !allTagSet[tid] {
					allTagSet[tid] = true
					flatTagIDs = append(flatTagIDs, tid)
				}
			}
		}

		if len(resolvedCatIDs) > 3 {
			return nil, nil, nil, errors.New("maksimal 3 kategori yang dapat dipilih")
		}
		return resolvedCatIDs, flatTagIDs, pivotRows, nil
	}

	uniqueCats := deduplicateInt64(categoryIDs)
	if len(uniqueCats) > 3 {
		return nil, nil, nil, errors.New("maksimal 3 kategori yang dapat dipilih")
	}
	uniqueTags := deduplicateInt64(flatTags)
	if len(uniqueCats) > 0 && len(uniqueTags) > len(uniqueCats)*5 {
		// Check if payment tags collapse into 1 quota when DB is available
		if database.DB != nil {
			var rows []tagMetaRow
			if err := database.DB.Table("business_tags bt").
				Select("bt.tag_id, bt.parent_id, COALESCE(tg.slug, '') AS group_slug").
				Joins("LEFT JOIN tag_groups tg ON tg.group_id = bt.group_id").
				Where("bt.tag_id IN ?", uniqueTags).
				Find(&rows).Error; err == nil && len(rows) > 0 {
				quota := 0
				hasPayment := false
				for _, r := range rows {
					if r.GroupSlug == "payment-transaction" {
						if !hasPayment {
							hasPayment = true
							quota++
						}
					} else {
						quota++
					}
				}
				if quota > len(uniqueCats)*5 {
					return nil, nil, nil, errors.New("maksimal 5 tag per kategori (metode pembayaran dihitung 1 tag)")
				}
			} else if len(uniqueTags) > len(uniqueCats)*5 {
				return nil, nil, nil, errors.New("maksimal 5 tag per kategori")
			}
		} else {
			return nil, nil, nil, errors.New("maksimal 5 tag per kategori")
		}
	}
	return uniqueCats, uniqueTags, nil, nil
}

func (s *businessService) CreateBusiness(ownerID string, input models.CreateBusinessInput) (string, error) {
	emailVerified, kycStatus, err := s.repo.GetOwnerKYCStatus(ownerID)
	if err != nil {
		return "", errors.New("akun pemilik bisnis tidak ditemukan")
	}

	if !emailVerified {
		return "", errors.New("Email akun Anda belum terverifikasi. Silakan verifikasi email Anda terlebih dahulu sebelum membuat bisnis.")
	}

	// Merchants may create and configure stores prior to KYC verification;
	// stores are initialized in closed state (is_manually_closed = true) and remain hidden until KYC approval.
	_ = kycStatus

	resolvedCatIDs, resolvedTagIDs, categoryTagRows, errCT := resolveAndValidateCategoryTags(input.CategoryIDs, input.Tags, input.CategoryTags)
	if errCT != nil {
		return "", errCT
	}
	input.CategoryIDs = resolvedCatIDs
	input.Tags = resolvedTagIDs

	// NIB Tag validation: NIB tag requires registered NIB value.
	cleanNIB := strings.TrimSpace(input.NIB)
	hasNIB := cleanNIB != ""
	if len(input.Tags) > 0 {
		hasNIBTag, err := s.repo.HasNIBTag(input.Tags)
		if err == nil && hasNIBTag && !hasNIB {
			return "", errors.New("tag 'Verified merchant (NIB)' hanya dapat dipilih jika bisnis memiliki NIB yang terdaftar")
		}
	}

	var customCategory *string
	if input.CustomCategory != nil {
		cleanCustom := strings.TrimSpace(*input.CustomCategory)
		if len(cleanCustom) > 100 {
			return "", errors.New("kategori custom maksimal 100 karakter")
		}
		if cleanCustom != "" {
			customCategory = &cleanCustom
		}
	}

	cleanedCustomTags := utils.CleanCustomTags(input.CustomTags)
	if err := utils.ValidateCustomTags(cleanedCustomTags, 10, 50); err != nil {
		return "", err
	}

	normalizedHashtags := utils.NormalizeHashtags(input.Hashtags)
	if err := utils.ValidateHashtags(normalizedHashtags, 10, 30); err != nil {
		return "", err
	}

	// Resolve certificate aliases if primary field is empty.
	if len(input.Certificates) == 0 {
		if len(input.CertificatesAlias) > 0 {
			input.Certificates = input.CertificatesAlias
		} else if len(input.BusinessCertificates) > 0 {
			input.Certificates = input.BusinessCertificates
		} else if len(input.BusinessCertificateSing) > 0 {
			input.Certificates = input.BusinessCertificateSing
		}
	}
	if input.Certificates == nil {
		input.Certificates = []models.BusinessCertificate{}
	}

	// Resolve description from compatibility alias if provided.
	if strings.TrimSpace(input.BusinessDetails) == "" && strings.TrimSpace(input.Description) != "" {
		input.BusinessDetails = strings.TrimSpace(input.Description)
	}
	if utf8.RuneCountInString(input.BusinessDetails) > 300 {
		return "", errors.New("deskripsi bisnis maksimal 300 karakter")
	}

	business := models.Business{
		BusinessID:       utils.GenerateUUID("BUS"),
		AccountID:        ownerID,
		LogoURL:          toPointerOrNil(input.LogoURL),
		BusinessName:     &input.BusinessName,
		Email:            &input.Email, // Ditambahkan karena sekarang wajib (NOT NULL)
		Status:           "draft",
		SubscriptionTier: "free",
		BusinessDetails:  toPointerOrNil(input.BusinessDetails),
		CustomCategory:   customCategory,
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

		StoreType:        toPointerOrNil(models.NormalizeStoreType(input.StoreType)),
		PhoneContact:     toPointerOrNil(input.PhoneContact),
		WhatsappContact:  toPointerOrNil(input.WhatsappContact),
		SocialInstagram:  toPointerOrNil(input.SocialInstagram),
		SocialTiktok:     toPointerOrNil(input.SocialTiktok),
		
		BusinessImages:   input.BusinessImages,
		Certificates:     input.Certificates,
		SocialWebsites:   input.SocialWebsites,
		CustomTags:       cleanedCustomTags,
		Hashtags:         normalizedHashtags,
		
		IsManuallyClosed: true, // Default toko baru adalah Tutup sampai merchant membukanya
		IsActive:         true,
	}

	var hours []models.BusinessHour
	for _, h := range input.Hours {
		hour := models.BusinessHour{
			DayOfWeek: h.DayOfWeek,
			OpenTime:  toPointerOrNil(h.OpenTime),
			CloseTime: toPointerOrNil(h.CloseTime),
			IsClosed:  h.IsClosed,
		}
		hours = append(hours, hour)
	}

	// Looping Websites dan Images manual dihapus

	// Enkripsi data sensitif (NIB & Dokumen Sertifikat)
	encryptBusinessLegal(&business)

	if err := s.repo.CreateBusinessWithRelations(&business, input.CategoryIDs, input.Tags, hours); err != nil {
		return "", err
	}

	if len(categoryTagRows) > 0 && database.DB != nil {
		for i := range categoryTagRows {
			categoryTagRows[i].BusinessID = business.BusinessID
		}
		_ = database.DB.Exec("DELETE FROM business_has_tags WHERE business_id = ?", business.BusinessID).Error
		_ = database.DB.Create(&categoryTagRows).Error
	}

	go meilisearch.SyncBusiness(database.DB, business.BusinessID)

	return business.BusinessID, nil
}

func (s *businessService) GetMyBusinesses(accountID, userRole, search string, categoryID int64, tagIDs []int64, cursor string, limit int) (*models.GetBusinessListResponse, error) {
	// Teruskan parameter pencarian dan filter ke repository
	businessesWithPreview, err := s.repo.GetBusinessListWithPreviews(accountID, userRole, search, categoryID, tagIDs, cursor, limit+1)
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

		bizName := ""
		if b.BusinessName != nil {
			bizName = *b.BusinessName
		}

		isOpenNow := utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)

		item := models.BusinessListItem{
			BusinessID:          b.BusinessID,
			BusinessName:        bizName,
			OwnerName:           b.OwnerName,
			LogoURL:             mainImage,
			Address:             fullAddress,
			AverageRating:       b.AverageRating,
			TotalProductReviews: b.TotalProductReviews,
			TotalReviews:        b.TotalProductReviews,
			LikeCount:           b.LikeCount,
			TotalLikes:          b.LikeCount,
			IsOpenNow:           isOpenNow,
			IsManuallyClosed:    b.IsManuallyClosed,
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

	// Decrypt sensitive documents for owner/admin access.
	decryptBusinessLegal(business)

	business.BusinessNameAlias = business.BusinessName
	business.SocialWebsitesAlias = business.SocialWebsites
	business.TotalLikes = business.LikeCount
	business.TotalReviews = business.ReviewCount

	// Initialize certificates slice to prevent null JSON responses and set aliases.
	if business.Certificates == nil {
		business.Certificates = []models.BusinessCertificate{}
	}
	business.CertificatesAlias = business.Certificates
	business.BusinessCertificatesAlias = business.Certificates

	// 🪄 MAGIC: Hitung is_open_now secara live sebelum dikembalikan ke Frontend
	business.IsOpenNow = utils.CalculateStoreStatus(business.IsManuallyClosed, business.Hours, business.ConfirmedOpenDate, business.ConfirmedOvertimeDate)

	// Populate Visibility info berdasarkan status KYC owner
	_, ownerKYCStatus, _ := s.repo.GetOwnerKYCStatus(business.AccountID)
	business.Visibility = kychelper.BuildBusinessVisibility(ownerKYCStatus, business.IsActive, business.Status)
	business.IsVerified = (ownerKYCStatus == "verified")

	// Suppress NIB tag if business lacks registered NIB.
	hasNIBDetail := business.NIB != nil && strings.TrimSpace(*business.NIB) != ""
	if !hasNIBDetail && len(business.Tags) > 0 {
		var filteredTags []models.Tag
		for _, t := range business.Tags {
			lowerName := strings.ToLower(t.Name)
			if !strings.Contains(lowerName, "(nib)") && !strings.Contains(lowerName, "terverifikasi (nib)") {
				filteredTags = append(filteredTags, t)
			}
		}
		business.Tags = filteredTags
	}

	// Populate ActivePromotions
	var activePromos []models.BusinessActivePromotionItem
	if database.DB != nil {
		now := time.Now()
		var rawPromos []models.Promotion
		if errPromo := database.DB.Preload("Products.Product").
			Preload("BundleItems.Product").
			Preload("Post").
			Where("business_id = ? AND status = ? AND start_at <= ? AND end_at >= ?", businessID, models.PromotionStatusActive, now, now).
			Order("created_at DESC").
			Find(&rawPromos).Error; errPromo == nil {
			for _, p := range rawPromos {
				title := "Promo Spesial Toko"
				if p.PackageName != nil && strings.TrimSpace(*p.PackageName) != "" {
					title = strings.TrimSpace(*p.PackageName)
				} else if strings.TrimSpace(p.Caption) != "" {
					title = strings.TrimSpace(p.Caption)
				} else if p.Post != nil && strings.TrimSpace(p.Post.Content) != "" {
					title = strings.TrimSpace(p.Post.Content)
				}

				var banner *string
				if p.PackageImage != nil && *p.PackageImage != "" {
					banner = p.PackageImage
				} else if p.Post != nil && len(p.Post.Images) > 0 {
					banner = &p.Post.Images[0]
				} else if len(p.Products) > 0 && p.Products[0].Product != nil && len(p.Products[0].Product.ProductPreview) > 0 {
					banner = &p.Products[0].Product.ProductPreview[0]
				} else if len(p.BundleItems) > 0 && p.BundleItems[0].Product != nil && len(p.BundleItems[0].Product.ProductPreview) > 0 {
					banner = &p.BundleItems[0].Product.ProductPreview[0]
				}

				activePromos = append(activePromos, models.BusinessActivePromotionItem{
					PromotionID:   p.PromotionID,
					Title:         title,
					Caption:       p.Caption,
					BannerURL:     banner,
					PromotionType: p.PromotionType,
					DiscountPct:   p.SavingsPercentage,
					StartAt:       p.StartAt,
					EndAt:         p.EndAt,
				})
			}
		}

		var rawPosts []models.BusinessPost
		if errPosts := database.DB.Where("business_id = ? AND is_promotion = ? AND (promotion_end_at IS NULL OR promotion_end_at >= ?)", businessID, true, now).
			Order("created_at DESC").
			Find(&rawPosts).Error; errPosts == nil {
			for _, post := range rawPosts {
				title := "Promo Toko"
				if strings.TrimSpace(post.Content) != "" {
					title = strings.TrimSpace(post.Content)
				}

				var banner *string
				if len(post.Images) > 0 {
					banner = &post.Images[0]
				}

				endAt := time.Now().Add(24 * time.Hour)
				if post.PromotionEndAt != nil {
					endAt = *post.PromotionEndAt
				}

				activePromos = append(activePromos, models.BusinessActivePromotionItem{
					PromotionID:   "POST-" + post.PostID,
					Title:         title,
					Caption:       post.Content,
					BannerURL:     banner,
					PromotionType: "post_promo",
					StartAt:       post.CreatedAt,
					EndAt:         endAt,
				})
			}
		}
	}

	if activePromos == nil {
		activePromos = []models.BusinessActivePromotionItem{}
	}
	business.ActivePromotions = activePromos

	return business, nil
}

func (s *businessService) UpdateBusiness(requesterID, businessID string, input models.UpdateBusinessInput) error {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan")
	}

	// Decrypt existing sensitive documents to avoid double encryption upon re-saving.
	decryptBusinessLegal(business)

	// Resolve primary field from alias if omitted.
	if input.BusinessName == nil && input.Name != nil {
		input.BusinessName = input.Name
	}
	if input.SocialWebsites == nil && input.Websites != nil {
		input.SocialWebsites = input.Websites
	}
	if input.AddressDetail == nil && input.Address != nil {
		input.AddressDetail = input.Address
	}
	if input.Certificates == nil {
		if input.CertificatesAlias != nil {
			input.Certificates = input.CertificatesAlias
		} else if input.BusinessCertificates != nil {
			input.Certificates = input.BusinessCertificates
		} else if input.BusinessCertificateSing != nil {
			input.Certificates = input.BusinessCertificateSing
		}
	}

	hasHoursUpdate := (input.Hours != nil && len(*input.Hours) > 0) || (input.EventSchedules != nil && len(*input.EventSchedules) > 0)
	hasProfileUpdate := input.LogoURL != nil || input.BusinessImages != nil ||
		input.CategoryIDs != nil || input.CustomCategory != nil ||
		input.Tags != nil || input.CustomTags != nil || input.Hashtags != nil ||
		input.BusinessDetails != nil || input.StoreType != nil ||
		input.Country != nil || input.Province != nil || input.City != nil ||
		input.AddressDetail != nil || input.Address != nil || input.GoogleMapsLink != nil ||
		input.Latitude != nil || input.Longitude != nil || input.IsLocationShared != nil ||
		input.VenueName != nil || input.SocialInstagram != nil || input.SocialTiktok != nil ||
		input.SocialWebsites != nil || input.Websites != nil

	// Verify requester is owner or active admin on this store.
	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return errors.New("forbidden: Anda tidak memiliki akses untuk memperbarui bisnis ini")
		}

		// Non-owner admins cannot alter store identity, NIB, certificates, or phone number.
		hasIdentityUpdate := input.BusinessName != nil || input.Name != nil ||
			input.NIB != nil || input.Handle != nil || input.PhoneContact != nil || input.WhatsappContact != nil ||
			input.Email != nil || input.Certificates != nil
		if hasIdentityUpdate {
			return errors.New("forbidden: hanya pemilik toko (owner) yang dapat mengubah identitas, legalitas, dan kontak resmi bisnis")
		}

		if (hasHoursUpdate || hasProfileUpdate) && !permissions.HasPermission(database.DB, requesterID, businessID, models.PermEditStoreProfileAndHours) {
			return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah profil atau jam operasional bisnis")
		}
	}

	// ==========================================
	// ==========================================
	if input.BusinessName == nil && input.Name != nil {
		input.BusinessName = input.Name
	}

	if input.BusinessName != nil {
		cleanName := strings.TrimSpace(*input.BusinessName)
		if cleanName == "" {
			return errors.New("nama bisnis wajib diisi dan tidak boleh kosong")
		}
		// Record name change history if updated.
		if business.BusinessName != nil && *business.BusinessName != "" && *business.BusinessName != cleanName {
			history := models.BusinessNameHistory{
				HistoryID:  utils.GenerateUUID("BNH"),
				BusinessID: business.BusinessID,
				OldName:    *business.BusinessName,
				NewName:    cleanName,
				ChangedBy:  requesterID,
				CreatedAt:  time.Now(),
			}
			_ = s.repo.CreateNameHistory(&history)
		}
		business.BusinessName = &cleanName
	}

	if input.Email != nil {
		cleanEmail := strings.TrimSpace(*input.Email)
		if cleanEmail == "" {
			return errors.New("email bisnis wajib diisi dan tidak boleh kosong")
		}
		business.Email = &cleanEmail
	}

	if input.Handle != nil {
		cleanHandle := strings.TrimSpace(*input.Handle)
		if cleanHandle != "" {
			if s.repo.CheckHandleExists(cleanHandle, business.BusinessID) {
				return errors.New("handle sudah digunakan, silakan pilih yang lain")
			}
			business.Handle = &cleanHandle
		} else {
			business.Handle = nil
		}
	}

	if input.StoreType != nil {
		cleanStoreType := models.NormalizeStoreType(*input.StoreType)
		if cleanStoreType == "" {
			cleanStoreType = models.StoreTypeToko
		}
		business.StoreType = &cleanStoreType
	}

	// ==========================================
	// ==========================================
	if input.LogoURL != nil {
		cleanLogo := strings.TrimSpace(*input.LogoURL)
		if cleanLogo == "" {
			business.LogoURL = nil
		} else {
			business.LogoURL = &cleanLogo
		}
	}

	// Resolve description alias if primary field is empty.
	if input.BusinessDetails == nil && input.Description != nil {
		input.BusinessDetails = input.Description
	}

	if input.BusinessDetails != nil {
		cleanDetails := strings.TrimSpace(*input.BusinessDetails)
		if utf8.RuneCountInString(cleanDetails) > 300 {
			return errors.New("deskripsi bisnis maksimal 300 karakter")
		}
		if cleanDetails == "" {
			business.BusinessDetails = nil
		} else {
			business.BusinessDetails = &cleanDetails
		}
	}

	if input.CustomCategory != nil {
		cleanCustom := strings.TrimSpace(*input.CustomCategory)
		if len(cleanCustom) > 100 {
			return errors.New("kategori custom maksimal 100 karakter")
		}
		if cleanCustom == "" {
			business.CustomCategory = nil
		} else {
			business.CustomCategory = &cleanCustom
		}
	}

	if input.NIB != nil {
		cleanNIB := strings.TrimSpace(*input.NIB)
		if cleanNIB == "" {
			business.NIB = nil
		} else {
			business.NIB = &cleanNIB
		}
	}

	if input.Country != nil {
		cleanCountry := strings.TrimSpace(*input.Country)
		if cleanCountry == "" {
			defCountry := "Indonesia"
			business.Country = &defCountry
		} else {
			business.Country = &cleanCountry
		}
	}

	if input.Province != nil {
		cleanProvince := strings.TrimSpace(*input.Province)
		if cleanProvince == "" {
			business.Province = nil
		} else {
			business.Province = &cleanProvince
		}
	}

	if input.City != nil {
		cleanCity := strings.TrimSpace(*input.City)
		if cleanCity == "" {
			business.City = nil
		} else {
			business.City = &cleanCity
		}
	}

	if input.AddressDetail != nil {
		cleanAddress := strings.TrimSpace(*input.AddressDetail)
		if cleanAddress == "" {
			business.AddressDetail = nil
		} else {
			business.AddressDetail = &cleanAddress
		}
	}

	if input.GoogleMapsLink != nil {
		cleanLink := strings.TrimSpace(*input.GoogleMapsLink)
		if cleanLink == "" {
			business.GoogleMapsLink = nil
		} else {
			business.GoogleMapsLink = &cleanLink
		}
	}

	if input.Latitude != nil {
		business.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		business.Longitude = input.Longitude
	}

	if input.IsLocationShared != nil {
		business.IsLocationShared = *input.IsLocationShared
	}

	if input.VenueName != nil {
		cleanVenue := strings.TrimSpace(*input.VenueName)
		if cleanVenue == "" {
			business.VenueName = nil
		} else {
			business.VenueName = &cleanVenue
		}
	}

	if input.EventStartDate != nil {
		cleanDate := strings.TrimSpace(*input.EventStartDate)
		if cleanDate == "" {
			business.EventStartDate = nil
		} else {
			business.EventStartDate = parseDatePointer(cleanDate)
		}
	}

	if input.EventEndDate != nil {
		cleanDate := strings.TrimSpace(*input.EventEndDate)
		if cleanDate == "" {
			business.EventEndDate = nil
		} else {
			business.EventEndDate = parseDatePointer(cleanDate)
		}
	}

	if input.EventSchedules != nil {
		business.EventSchedules = *input.EventSchedules
	}

	if input.PhoneContact != nil {
		cleanPhone := strings.TrimSpace(*input.PhoneContact)
		if cleanPhone == "" {
			business.PhoneContact = nil
		} else {
			business.PhoneContact = &cleanPhone
		}
	}

	if input.WhatsappContact != nil {
		cleanWA := strings.TrimSpace(*input.WhatsappContact)
		if cleanWA == "" {
			business.WhatsappContact = nil
		} else {
			business.WhatsappContact = &cleanWA
		}
	}

	if input.SocialInstagram != nil {
		cleanIG := strings.TrimSpace(*input.SocialInstagram)
		if cleanIG == "" {
			business.SocialInstagram = nil
		} else {
			business.SocialInstagram = &cleanIG
		}
	}

	if input.SocialTiktok != nil {
		cleanTT := strings.TrimSpace(*input.SocialTiktok)
		if cleanTT == "" {
			business.SocialTiktok = nil
		} else {
			business.SocialTiktok = &cleanTT
		}
	}

	// ==========================================
	// ==========================================
	if input.BusinessImages != nil {
		if *input.BusinessImages == nil {
			business.BusinessImages = []string{}
		} else {
			business.BusinessImages = *input.BusinessImages
		}
	}
	if input.Certificates != nil {
		if *input.Certificates == nil {
			business.Certificates = []models.BusinessCertificate{}
		} else {
			business.Certificates = *input.Certificates
		}
	}
	if input.SocialWebsites != nil {
		if *input.SocialWebsites == nil {
			business.SocialWebsites = []string{}
		} else {
			business.SocialWebsites = *input.SocialWebsites
		}
	}
	if input.CustomTags != nil {
		if *input.CustomTags == nil {
			business.CustomTags = []string{}
		} else {
			cleaned := utils.CleanCustomTags(*input.CustomTags)
			if err := utils.ValidateCustomTags(cleaned, 10, 50); err != nil {
				return err
			}
			business.CustomTags = cleaned
		}
	}
	if input.Hashtags != nil {
		if *input.Hashtags == nil {
			business.Hashtags = []string{}
		} else {
			normalized := utils.NormalizeHashtags(*input.Hashtags)
			if err := utils.ValidateHashtags(normalized, 10, 30); err != nil {
				return err
			}
			business.Hashtags = normalized
		}
	}

	// ==========================================
	// ==========================================
	var hoursToUpdate *[]models.BusinessHour
	if input.Hours != nil {
		seenDays := make(map[int]bool)
		var mappedHours []models.BusinessHour
		for _, h := range *input.Hours {
			if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
				return errors.New("day_of_week pada jam operasional harus bernilai antara 0 (Minggu) sampai 6 (Sabtu)")
			}
			// Cegah duplikasi hari dalam array jam operasional
			if seenDays[h.DayOfWeek] {
				continue
			}
			seenDays[h.DayOfWeek] = true

			hour := models.BusinessHour{
				DayOfWeek: h.DayOfWeek,
				OpenTime:  toPointerOrNil(h.OpenTime),
				CloseTime: toPointerOrNil(h.CloseTime),
				IsClosed:  h.IsClosed,
			}
			mappedHours = append(mappedHours, hour)
		}
		hoursToUpdate = &mappedHours
	}

	var catIDs *[]int64
	var tagIDs *[]int64
	var categoryTagRows []models.BusinessHasTag

	if input.CategoryTags != nil {
		resolvedCats, resolvedTags, rows, errCT := resolveAndValidateCategoryTags(nil, nil, *input.CategoryTags)
		if errCT != nil {
			return errCT
		}
		catIDs = &resolvedCats
		tagIDs = &resolvedTags
		categoryTagRows = rows
	} else {
		if input.CategoryIDs != nil {
			uniqueCats := deduplicateInt64(*input.CategoryIDs)
			if len(uniqueCats) > 3 {
				return errors.New("maksimal 3 kategori yang dapat dipilih")
			}
			catIDs = &uniqueCats
		}
		if input.Tags != nil {
			var catsForCheck []int64
			if catIDs != nil {
				catsForCheck = *catIDs
			}
			_, resolvedTags, _, errCT := resolveAndValidateCategoryTags(catsForCheck, *input.Tags, nil)
			if errCT != nil {
				return errCT
			}
			tagIDs = &resolvedTags
		}
	}

	hasNIBUpdate := business.NIB != nil && strings.TrimSpace(*business.NIB) != ""

	if tagIDs != nil {
		if len(*tagIDs) > 0 {
			hasNIBTag, err := s.repo.HasNIBTag(*tagIDs)
			if err == nil && hasNIBTag && !hasNIBUpdate {
				return errors.New("tag 'Verified merchant (NIB)' hanya dapat dipilih jika bisnis memiliki NIB yang terdaftar")
			}
		}
	} else if !hasNIBUpdate {
		// If NIB is removed but tags are omitted in update request, strip NIB tag.
		_ = s.repo.RemoveNIBTagFromBusiness(business.BusinessID)
	}

	// Enkripsi dokumen sensitif sebelum disimpan ke basis data
	encryptBusinessLegal(business)

	if err := s.repo.UpdateBusiness(business, hoursToUpdate, catIDs, tagIDs); err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "business_has_categories") || strings.Contains(errStr, "business_categories") {
			return errors.New("satu atau lebih kategori yang dipilih tidak valid atau tidak ditemukan")
		}
		if strings.Contains(errStr, "business_has_tags") || strings.Contains(errStr, "business_tags") {
			return errors.New("satu atau lebih tag yang dipilih tidak valid atau tidak ditemukan")
		}
		return fmt.Errorf("gagal memperbarui data bisnis di database: %w", err)
	}

	if len(categoryTagRows) > 0 && database.DB != nil {
		for i := range categoryTagRows {
			categoryTagRows[i].BusinessID = business.BusinessID
		}
		_ = database.DB.Exec("DELETE FROM business_has_tags WHERE business_id = ?", business.BusinessID).Error
		_ = database.DB.Create(&categoryTagRows).Error
	}

	// Alert owner if staff updates store profile or operating hours.
	if !isOwner && (hasHoursUpdate || hasProfileUpdate) {
		notification.NotifyOwnerStaffActivity(
			database.DB,
			s.notifService,
			businessID,
			requesterID,
			"update_store_profile_and_hours",
			"ℹ️ Staf Memperbarui Profil/Jam Toko",
			"Staf toko Anda baru saja memperbarui profil atau jadwal jam operasional toko.",
			map[string]interface{}{
				"business_id": businessID,
			},
		)
	}

	_ = meilisearch.UpdateBusinessStatus(database.DB, businessID)
	go meilisearch.SyncBusiness(database.DB, businessID)

	return nil
}

func (s *businessService) DeleteBusiness(ownerID, businessID string) error {
	business, err := s.repo.FindBusinessByIDAndOwner(businessID, ownerID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan atau Anda tidak memiliki akses")
	}

	var productIDs []string
	database.DB.Model(&models.Product{}).Where("business_id = ?", businessID).Pluck("product_id", &productIDs)

	if err := s.repo.DeleteBusiness(business); err != nil {
		return err
	}

	if meilisearch.Client != nil {
		go func() {
			_, _ = meilisearch.Client.Index("businesses").DeleteDocument(businessID, nil)
			for _, pid := range productIDs {
				_, _ = meilisearch.Client.Index("products").DeleteDocument(pid, nil)
			}
		}()
	}

	if config.RDB != nil {
		config.RDB.Del(config.Ctx, fmt.Sprintf("business_detail:%s", businessID))
	}

	return nil
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

		if !permissions.HasPermission(database.DB, requesterID, businessID, models.PermManageStoreStatus) {
			return false, errors.New("forbidden: Anda tidak memiliki izin untuk mengubah status operasional toko")
		}
	}

	// Toggle state.
	newStatus := !business.IsManuallyClosed

	// KYC Gate: opening store (!newStatus) strictly requires approved owner KYC.
	// Closing store (newStatus == true) is permitted without KYC.
	if !newStatus {
		_, ownerKYCStatus, _ := s.repo.GetOwnerKYCStatus(business.AccountID)
		if ownerKYCStatus != "verified" {
			if ownerKYCStatus == "pending" {
				return false, errors.New("toko belum dapat dibuka karena verifikasi identitas (KYC) pemilik toko sedang dalam proses peninjauan, mohon tunggu")
			}
			if ownerKYCStatus == "rejected" {
				return false, errors.New("toko belum dapat dibuka karena pengajuan verifikasi identitas (KYC) ditolak. Silakan ajukan ulang berkas KYC Anda")
			}
			return false, errors.New("toko belum dapat dibuka karena pemilik toko belum menyelesaikan verifikasi identitas (KYC)")
		}
	}

	loc := utils.GetWIBLocation()
	todayStr := time.Now().In(loc).Format("2006-01-02")

	business.IsManuallyClosed = newStatus
	if !newStatus {
		// Merchant toggles store open.
		// Set confirmed_open_date to today to immediately open store.
		business.ConfirmedOpenDate = &todayStr
		business.ConfirmedOvertimeDate = &todayStr
	}

	if err := s.repo.ToggleStoreStatus(business); err != nil {
		return false, errors.New("gagal memperbarui status toko")
	}

	// Invalidate cache detail bisnis agar selalu fresh
	if config.RDB != nil {
		config.RDB.Del(config.Ctx, fmt.Sprintf("business_detail:%s", businessID))
	}

	_ = meilisearch.UpdateBusinessStatus(database.DB, businessID)
	go meilisearch.SyncBusiness(database.DB, businessID)

	// Alert owner if performed by staff admin.
	if !isOwner {
		statusStr := "Buka"
		if newStatus {
			statusStr = "Tutup Sementara"
		}
		notification.NotifyOwnerStaffActivity(
			database.DB,
			s.notifService,
			businessID,
			requesterID,
			"toggle_store_status",
			"⚠️ Staf Mengubah Status Operasional Toko",
			fmt.Sprintf("Staf toko Anda baru saja mengubah status operasional toko menjadi: %s", statusStr),
			map[string]interface{}{
				"is_manually_closed": newStatus,
			},
		)
	}

	return business.IsManuallyClosed, nil
}

func (s *businessService) ConfirmScheduleAction(requesterID, businessID string, input models.ConfirmScheduleInput) error {
	business, err := s.repo.FindBusinessByID(businessID)
	if err != nil {
		return errors.New("bisnis tidak ditemukan")
	}

	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return errors.New("forbidden: Anda tidak memiliki akses untuk mengonfirmasi jadwal toko ini")
		}

		if !permissions.HasPermission(database.DB, requesterID, businessID, models.PermManageStoreStatus) {
			return errors.New("forbidden: Anda tidak memiliki izin untuk mengonfirmasi jadwal toko")
		}
	}

	loc := utils.GetWIBLocation()
	todayStr := time.Now().In(loc).Format("2006-01-02")

	var actionErr error
	switch input.Action {
	case "confirm_open", "confirm_overtime":
		// KYC Gate: open and overtime confirmations strictly require approved owner KYC.
		_, ownerKYCStatus, _ := s.repo.GetOwnerKYCStatus(business.AccountID)
		if ownerKYCStatus != "verified" {
			if ownerKYCStatus == "pending" {
				return errors.New("toko belum dapat dibuka karena verifikasi identitas (KYC) pemilik toko sedang dalam proses peninjauan, mohon tunggu")
			}
			if ownerKYCStatus == "rejected" {
				return errors.New("toko belum dapat dibuka karena pengajuan verifikasi identitas (KYC) ditolak. Silakan ajukan ulang berkas KYC Anda")
			}
			return errors.New("toko belum dapat dibuka karena pemilik toko belum menyelesaikan verifikasi identitas (KYC)")
		}
		if input.Action == "confirm_open" {
			// Store confirmation: ready to open per daily schedule.
			actionErr = s.repo.ConfirmOpenToday(businessID, todayStr)
		} else {
			// Store confirmation: overtime beyond regular closing time.
			actionErr = s.repo.ConfirmOvertimeToday(businessID, todayStr)
		}
	case "decline":
		// Store decline: remain closed (permitted without KYC).
		actionErr = s.repo.DeclineSchedule(businessID)
	default:
		return errors.New("aksi tidak valid")
	}

	if actionErr != nil {
		return actionErr
	}

	// Invalidate cache detail bisnis agar selalu fresh
	if config.RDB != nil {
		config.RDB.Del(config.Ctx, fmt.Sprintf("business_detail:%s", businessID))
	}

	_ = meilisearch.UpdateBusinessStatus(database.DB, businessID)
	go meilisearch.SyncBusiness(database.DB, businessID)

	return nil
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

	isOwner := business.AccountID == requesterID
	if !isOwner {
		var assignment models.BusinessAdminAssignment
		if err := database.DB.Where("admin_id = ? AND business_id = ? AND status IN ('active', 'scheduled_for_deletion')", requesterID, businessID).First(&assignment).Error; err != nil {
			return nil, errors.New("forbidden: Anda tidak memiliki akses untuk mengelola siaran live toko ini")
		}

		if !permissions.HasPermission(database.DB, requesterID, businessID, models.PermManageLiveStreaming) {
			return nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengelola siaran live toko ini")
		}
	}

	now := time.Now()

	if input.IsLive {
		if input.Platform == nil || strings.TrimSpace(*input.Platform) == "" {
			return nil, errors.New("platform live streaming wajib dipilih ('tiktok' atau 'instagram')")
		}

		platform := strings.ToLower(strings.TrimSpace(*input.Platform))
		var socialURL string

		switch platform {
		case "tiktok":
			if business.SocialTiktok == nil || strings.TrimSpace(*business.SocialTiktok) == "" {
				return nil, errors.New("bisnis ini belum mempunyai sosial media TikTok. Silakan lengkapi profil media sosial TikTok bisnis Anda terlebih dahulu sebelum mengaktifkan Live Streaming")
			}
			socialURL = formatSocialURL("tiktok", *business.SocialTiktok)
		case "instagram":
			if business.SocialInstagram == nil || strings.TrimSpace(*business.SocialInstagram) == "" {
				return nil, errors.New("bisnis ini belum mempunyai sosial media Instagram. Silakan lengkapi profil media sosial Instagram bisnis Anda terlebih dahulu sebelum mengaktifkan Live Streaming")
			}
			socialURL = formatSocialURL("instagram", *business.SocialInstagram)
		default:
			return nil, errors.New("platform live streaming tidak valid. Platform yang didukung saat ini hanya 'tiktok' dan 'instagram'")
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
			log.Printf("❌ [ToggleLiveStatus] Error updating live status for business %s: %v", business.BusinessID, err)
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

	business.IsLive = false
	business.LivePlatform = nil
	business.LiveStartedAt = nil

	if err := s.repo.UpdateLiveStatus(business); err != nil {
		log.Printf("❌ [ToggleLiveStatus] Error turning off live status for business %s: %v", business.BusinessID, err)
		return nil, errors.New("gagal menonaktifkan status live toko")
	}

	// 🏁 Poin 5-2: Notifikasi sesi siaran langsung selesai
	if s.notifService != nil {
		bName := "Toko"
		if business.BusinessName != nil && strings.TrimSpace(*business.BusinessName) != "" {
			bName = strings.TrimSpace(*business.BusinessName)
		}
		bizID := business.BusinessID
		reqID := requesterID

		go func() {
			customerIDs, err := s.repo.GetBookmarkAccountIDs(bizID)
			if err != nil || len(customerIDs) == 0 {
				return
			}

			title := fmt.Sprintf("🏁 Sesi Live Streaming %s Telah Berakhir", bName)
			body := fmt.Sprintf("Siaran langsung toko %s telah selesai. Terima kasih atas partisipasi Anda!", bName)
			data := map[string]interface{}{
				"business_id":   bizID,
				"business_name": bName,
				"action":        "live_ended",
			}

			s.notifService.SendNotification(customerIDs, reqID, "merchant_live_ended", bizID, title, body, data, nil)
		}()
	}

	return &models.LiveStatusResponse{
		BusinessID: business.BusinessID,
		IsLive:     false,
	}, nil
}