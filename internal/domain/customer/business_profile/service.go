package business_profile

import (
	"fmt"
	"time"

	"toentoen-backend/internal/domain/merchant/analytics"
	"toentoen-backend/internal/domain/shared/models"
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
	IsOpenNow       bool     `json:"is_open_now"`
	IsBookmarked    bool     `json:"is_bookmarked"`
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
	}

	// Hitung IsOpenNow
	isOpenNow := false
	if !business.IsManuallyClosed {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		now := time.Now().In(loc)
		currentDayInt := int(now.Weekday())

		for _, h := range business.Hours {
			if h.DayOfWeek == currentDayInt {
				if h.OpenTime != nil && h.CloseTime != nil {
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
		IsOpenNow:       isOpenNow,
		IsBookmarked:    isBookmarked,
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
		Hours:           business.Hours,

		RecentProducts:    mappedRecentProducts,
		RecentPosts:       recentPosts,
		SimilarBusinesses: similarBusinesses,
		RecentReviews:     recentReviews,
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
		}

		// Hitung IsOpenNow
		isOpenNow := false
		if !business.IsManuallyClosed {
			loc, _ := time.LoadLocation("Asia/Jakarta")
			now := time.Now().In(loc)
			currentDayInt := int(now.Weekday())

			for _, h := range business.Hours {
				if h.DayOfWeek == currentDayInt {
					if h.OpenTime != nil && h.CloseTime != nil {
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
			IsOpenNow:       isOpenNow,
			IsBookmarked:    true, // always true if fetched from bookmarks
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
	return s.repo.GetAllProductGroups(businessID)
}

func mapToSimpleProduct(p models.Product) SimpleProductResponse {
	var originalPrice *float64
	currentPrice := p.Price
	var discountAmount float64
	var discountPct float64
	isDiscounted := false

	if p.Discount > 0 {
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

	var response []models.PostResponse
	for _, p := range posts {
		response = append(response, models.PostResponse{
			PostID:       p.PostID,
			BusinessID:   p.BusinessID,
			Content:      p.Content,
			Images:       p.Images,
			VideoURL:     p.VideoURL,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CreatedAt:    p.CreatedAt,
			UpdatedAt:    p.UpdatedAt,
			IsLikedByMe:  likedMap[p.PostID],
			Business:     p.Business,
		})
	}
	
	if response == nil {
		response = []models.PostResponse{}
	}

	return response, nil
}
