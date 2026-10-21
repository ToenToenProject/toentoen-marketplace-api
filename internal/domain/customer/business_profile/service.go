package business_profile

import (
	"math"
	"strings"
	"time"

	"toentoen-backend/internal/domain/merchant/analytics"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/utils"
)

type SimpleProductResponse struct {
	ProductID      string   `json:"product_id"`
	ProductName    string   `json:"product_name"`
	ProductPreview []string `json:"product_preview"`
	Price          float64  `json:"price"`
	OriginalPrice  *float64 `json:"original_price,omitempty"`
	IsDiscounted   bool     `json:"is_discounted"`
	DiscountType   string   `json:"discount_type"`
	DiscountAmount float64  `json:"discount_amount"` // Potongan harga (Rupiah)
	DiscountPct    float64  `json:"discount_pct"`    // Besaran diskon (%)
	AverageRating  float64  `json:"average_rating"`
	ReviewCount    int      `json:"review_count"`
	StockStatus    string   `json:"stock_status"`
}

type BusinessProfileResponse struct {
	BusinessID      string   `json:"business_id"`
	BusinessName    string   `json:"business_name"`
	LogoURL         *string  `json:"logo_url"`
	Category        string   `json:"category"`
	CustomCategory  *string  `json:"custom_category,omitempty"`
	CustomTags      []string `json:"custom_tags,omitempty"`
	Hashtags         []string `json:"hashtags,omitempty"`
	IsOpenNow        bool     `json:"is_open_now"`
	IsManuallyClosed bool     `json:"is_manually_closed"`
	IsBookmarked     bool     `json:"is_bookmarked"`
	IsLive          bool     `json:"is_live"`
	LivePlatform    *string  `json:"live_platform,omitempty"`
	LiveStartedAt   *time.Time `json:"live_started_at,omitempty"`
	
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
	Certificates    []models.BusinessCertificate `json:"certificates"`

	Hours []models.BusinessHour `json:"hours"`

	RecentProducts    []SimpleProductResponse    `json:"recent_products"`
	RecentPosts       []models.BusinessPost      `json:"recent_posts"`
	SimilarBusinesses []models.Business          `json:"similar_businesses"`
	RecentReviews     []models.Review            `json:"recent_reviews"`
}

type Service interface {
	GetBusinessProfile(businessID string, accountID string, sourceType string, searchedTag string, productID string) (BusinessProfileResponse, error)
	ToggleBookmark(accountID, businessID string) (bool, error)
	GetBookmarks(accountID string) ([]BusinessProfileResponse, error)

	GetAllProductGroups(businessID string) ([]models.ProductGroup, error)
	GetAllProducts(businessID string, groupID string) ([]SimpleProductResponse, error)
	GetAllPosts(businessID string, accountID string) ([]models.PostResponse, error)
}

type service struct {
	repo         Repository
	analyticsSvc analytics.AnalyticsService
}

func NewService(r Repository, analyticsSvc analytics.AnalyticsService) Service {
	return &service{
		repo:         r,
		analyticsSvc: analyticsSvc,
	}
}

func (s *service) GetBusinessProfile(businessID, accountID string, sourceType string, searchedTag string, productID string) (BusinessProfileResponse, error) {
	business, err := s.repo.GetBusinessDetail(businessID)
	if err != nil {
		return BusinessProfileResponse{}, err
	}

	// Catat event visitor secara async (fire-and-forget, tidak memblok response)
	// Abaikan kunjungan dari pemilik toko sendiri agar data analitik akurat
	isOwner := accountID != "" && business.AccountID == accountID
	if s.analyticsSvc != nil && !isOwner {
		if sourceType == "" {
			sourceType = "direct"
		}
		go s.analyticsSvc.TrackVisitor(businessID, accountID, sourceType, searchedTag, productID)
	}

	var isBookmarked bool
	if accountID != "" {
		isBookmarked, _ = s.repo.CheckBookmark(accountID, businessID)
	}

	// Tentukan Kategori Utama (Ambil yang pertama jika ada)
	category := "General"
	if len(business.Categories) > 0 {
		category = business.Categories[0].Name
	} else if business.CustomCategory != nil && strings.TrimSpace(*business.CustomCategory) != "" {
		category = *business.CustomCategory
	}

	// Hitung IsOpenNow secara akurat menggunakan utils.CalculateStoreStatus (terkunci ke WIB / UTC+7)
	isOpenNow := utils.CalculateStoreStatus(business.IsManuallyClosed, business.Hours, business.ConfirmedOpenDate, business.ConfirmedOvertimeDate)

	bName := ""
	if business.BusinessName != nil {
		bName = *business.BusinessName
	}
	
	city := ""
	if business.City != nil {
		city = *business.City
	}

	recentProducts, _ := s.repo.GetRecentProducts(businessID, 3)
	var mappedRecentProducts []SimpleProductResponse
	if recentProducts == nil {
		mappedRecentProducts = []SimpleProductResponse{}
	} else {
		for _, rp := range recentProducts {
			mappedRecentProducts = append(mappedRecentProducts, mapToSimpleProduct(rp))
		}
	}

	recentPosts, _ := s.repo.GetRecentPosts(businessID, 2)
	if recentPosts == nil {
		recentPosts = []models.BusinessPost{}
	}

	similarBusinesses, _ := s.repo.GetSimilarBusinesses(category, city, businessID, 10)
	if similarBusinesses == nil {
		similarBusinesses = []models.Business{}
	} else {
		for i := range similarBusinesses {
			similarBusinesses[i].IsOpenNow = utils.CalculateStoreStatus(similarBusinesses[i].IsManuallyClosed, similarBusinesses[i].Hours, similarBusinesses[i].ConfirmedOpenDate, similarBusinesses[i].ConfirmedOvertimeDate)
		}
	}

	recentReviews, _ := s.repo.GetRecentTopReviews(businessID, 4)
	if recentReviews == nil {
		recentReviews = []models.Review{}
	}

	// Hitung status Live Streaming (otomatis kedaluwarsa jika sudah lewat 4 jam)
	isLive := business.IsLive
	var livePlatform *string
	var liveStartedAt *time.Time
	if isLive {
		if business.LiveStartedAt != nil && time.Since(*business.LiveStartedAt) > 4*time.Hour {
			isLive = false
		} else {
			livePlatform = business.LivePlatform
			liveStartedAt = business.LiveStartedAt
		}
	}

	return BusinessProfileResponse{
		BusinessID:      business.BusinessID,
		BusinessName:    bName,
		LogoURL:         business.LogoURL,
		Category:        category,
		CustomCategory:  business.CustomCategory,
		CustomTags:      business.CustomTags,
		Hashtags:         business.Hashtags,
		IsOpenNow:        isOpenNow,
		IsManuallyClosed: business.IsManuallyClosed,
		IsBookmarked:     isBookmarked,
		IsLive:          isLive,
		LivePlatform:    livePlatform,
		LiveStartedAt:   liveStartedAt,
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
		Certificates:    decryptBusinessCertificates(business.Certificates),
		Hours:           business.Hours,

		RecentProducts:    mappedRecentProducts,
		RecentPosts:       recentPosts,
		SimilarBusinesses: similarBusinesses,
		RecentReviews:     recentReviews,
	}, nil
}

func decryptBusinessCertificates(certs []models.BusinessCertificate) []models.BusinessCertificate {
	if certs == nil {
		return []models.BusinessCertificate{}
	}
	decrypted := make([]models.BusinessCertificate, len(certs))
	copy(decrypted, certs)
	for i := range decrypted {
		if decrypted[i].Number != "" {
			decrypted[i].Number, _ = crypto.Decrypt(decrypted[i].Number)
		}
		if decrypted[i].ImageURL != "" {
			decrypted[i].ImageURL, _ = crypto.Decrypt(decrypted[i].ImageURL)
		}
		decrypted[i].Name = decrypted[i].Type
		decrypted[i].CertificateName = decrypted[i].Type
		decrypted[i].CertificateType = decrypted[i].Type
		decrypted[i].CertificateNumber = decrypted[i].Number
		decrypted[i].URL = decrypted[i].ImageURL
		decrypted[i].CertificateImageURL = decrypted[i].ImageURL
	}
	return decrypted
}

func (s *service) ToggleBookmark(accountID, businessID string) (bool, error) {
	// Verifikasi toko exist
	_, err := s.repo.GetBusinessDetail(businessID)
	if err != nil {
		return false, err
	}
	return s.repo.ToggleBookmark(accountID, businessID)
}

func (s *service) GetBookmarks(accountID string) ([]BusinessProfileResponse, error) {
	businesses, err := s.repo.GetBookmarkedBusinesses(accountID)
	if err != nil {
		return nil, err
	}

	var responses []BusinessProfileResponse
	for _, business := range businesses {
		// Tentukan Kategori Utama (Ambil yang pertama jika ada)
		category := "General"
		if len(business.Categories) > 0 {
			category = business.Categories[0].Name
		} else if business.CustomCategory != nil && strings.TrimSpace(*business.CustomCategory) != "" {
			category = *business.CustomCategory
		}

		// Hitung IsOpenNow secara akurat menggunakan utils.CalculateStoreStatus (terkunci ke WIB / UTC+7)
		isOpenNow := utils.CalculateStoreStatus(business.IsManuallyClosed, business.Hours, business.ConfirmedOpenDate, business.ConfirmedOvertimeDate)

		bName := ""
		if business.BusinessName != nil {
			bName = *business.BusinessName
		}

		isLive := business.IsLive
		var livePlatform *string
		var liveStartedAt *time.Time
		if isLive {
			if business.LiveStartedAt != nil && time.Since(*business.LiveStartedAt) > 4*time.Hour {
				isLive = false
			} else {
				livePlatform = business.LivePlatform
				liveStartedAt = business.LiveStartedAt
			}
		}

		responses = append(responses, BusinessProfileResponse{
			BusinessID:      business.BusinessID,
			BusinessName:    bName,
			LogoURL:         business.LogoURL,
			Category:        category,
			CustomCategory:  business.CustomCategory,
			CustomTags:      business.CustomTags,
			Hashtags:         business.Hashtags,
			IsOpenNow:        isOpenNow,
			IsManuallyClosed: business.IsManuallyClosed,
			IsBookmarked:     true, // always true if fetched from bookmarks
			IsLive:          isLive,
			LivePlatform:    livePlatform,
			LiveStartedAt:   liveStartedAt,
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
			Certificates:    decryptBusinessCertificates(business.Certificates),
			Hours:           business.Hours,
		})
	}

	// Inisialisasi menjadi slice kosong agar bukan null (di JSON)
	if responses == nil {
		responses = []BusinessProfileResponse{}
	}

	return responses, nil
}

func (s *service) GetAllProductGroups(businessID string) ([]models.ProductGroup, error) {
	groups, err := s.repo.GetAllProductGroups(businessID)
	if err != nil {
		return nil, err
	}
	if s.repo.HasActivePromotion(businessID) {
		virtualPromo := models.ProductGroup{
			ProductGroupID: "promotion",
			BusinessID:     businessID,
			Name:           "Promosi",
			DisplayOrder:   0,
			IsActive:       true,
		}
		groups = append([]models.ProductGroup{virtualPromo}, groups...)
	}
	return groups, nil
}

func mapToSimpleProduct(p models.Product) SimpleProductResponse {
	var originalPrice *float64
	currentPrice := p.Price
	var discountAmount float64
	var discountPct float64
	isDiscounted := false

	if utils.IsDiscountActive(p.Discount, p.DiscountStartAt, p.DiscountEndAt, time.Now()) {
		cut, pct := utils.CalculateProductDiscount(p.Price, p.DiscountType, p.Discount, p.MaxDiscount)
		if cut > 0 {
			isDiscounted = true
			currentPrice = p.Price - cut
			orig := p.Price
			originalPrice = &orig
			discountAmount = cut
			discountPct = pct
		}
	}

	stockStatus := "Out of Stock"
	if p.StockType == "always_available" || p.StockType == "in_stock" {
		stockStatus = "In Stock"
	}

	return SimpleProductResponse{
		ProductID:      p.ProductID,
		ProductName:    p.ProductName,
		ProductPreview: p.ProductPreview,
		Price:          currentPrice,
		OriginalPrice:  originalPrice,
		IsDiscounted:   isDiscounted,
		DiscountType:   p.DiscountType,
		DiscountAmount: discountAmount,
		DiscountPct:    discountPct,
		AverageRating:  p.AverageRating,
		ReviewCount:    p.ReviewCount,
		StockStatus:    stockStatus,
	}
}

func (s *service) GetAllProducts(businessID string, groupID string) ([]SimpleProductResponse, error) {
	products, err := s.repo.GetAllProducts(businessID, groupID)
	if err != nil {
		return nil, err
	}

	var simpleProducts []SimpleProductResponse
	for _, p := range products {
		simpleProducts = append(simpleProducts, mapToSimpleProduct(p))
	}
	if simpleProducts == nil {
		simpleProducts = []SimpleProductResponse{}
	}

	return simpleProducts, nil
}

func (s *service) GetAllPosts(businessID string, accountID string) ([]models.PostResponse, error) {
	posts, err := s.repo.GetAllPosts(businessID)
	if err != nil {
		return nil, err
	}

	likedMap := make(map[string]bool)
	if accountID != "" {
		var postIDs []string
		for _, p := range posts {
			postIDs = append(postIDs, p.PostID)
		}
		likedMap, _ = s.repo.GetPostLikesByAccount(accountID, postIDs)
	}

	now := time.Now()
	var response []models.PostResponse
	for _, p := range posts {
		isPromoActive := p.IsPromotion
		if isPromoActive && p.PromotionEndAt != nil && p.PromotionEndAt.Before(now) {
			isPromoActive = false
		}

		var badgeText *string
		var targetAction *string
		var remainingDays *int
		if isPromoActive {
			badge := "Promosi"
			badgeText = &badge
			if p.PromotionID != nil && *p.PromotionID != "" {
				action := "open_promotion_group"
				targetAction = &action
			}
			if p.PromotionEndAt != nil && p.PromotionEndAt.After(now) {
				rem := int(math.Ceil(p.PromotionEndAt.Sub(now).Hours() / 24.0))
				remainingDays = &rem
			}
		}

		response = append(response, models.PostResponse{
			PostID:         p.PostID,
			BusinessID:     p.BusinessID,
			Content:        p.Content,
			Images:         p.Images,
			VideoURL:       p.VideoURL,
			LikeCount:      p.LikeCount,
			CommentCount:   p.CommentCount,
			CreatedAt:      p.CreatedAt,
			UpdatedAt:      p.UpdatedAt,
			IsLikedByMe:    likedMap[p.PostID],
			Business:       p.Business,
			IsPromotion:    isPromoActive,
			PromotionID:    p.PromotionID,
			PromotionEndAt: p.PromotionEndAt,
			RemainingDays:  remainingDays,
			BadgeText:      badgeText,
			TargetAction:   targetAction,
		})
	}
	
	if response == nil {
		response = []models.PostResponse{}
	}

	return response, nil

}
