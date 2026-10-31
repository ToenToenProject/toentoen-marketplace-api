package business_profile

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"toentoen-backend/internal/domain/merchant/analytics"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/utils"
)

type SavedShopItem struct {
	BusinessID              string     `json:"business_id"`
	ShopID                  string     `json:"shop_id"` // Alias untuk kemudahan FE
	ID                      string     `json:"id"`      // Alias untuk kemudahan FE
	BusinessName            string     `json:"business_name"`
	Name                    string     `json:"name"` // Alias untuk kemudahan FE
	LogoURL                 *string    `json:"logo_url"`
	Category                string     `json:"category"`
	CustomCategory          *string    `json:"custom_category,omitempty"`
	City                    *string    `json:"city"`
	Province                *string    `json:"province"`
	AddressDetail           *string    `json:"address_detail,omitempty"`
	Latitude                *float64   `json:"latitude,omitempty"`
	Longitude               *float64   `json:"longitude,omitempty"`
	AverageRating           float64    `json:"average_rating"`
	Rating                  float64    `json:"rating"` // Alias
	TotalReviews            int64      `json:"total_reviews"`
	ReviewCount             int64      `json:"review_count"` // Alias
	TotalLikes              int        `json:"total_likes"`
	LikeCount               int        `json:"like_count"` // Alias
	IsOpenNow               bool       `json:"is_open_now"`
	IsManuallyClosed        bool       `json:"is_manually_closed"`
	IsVerified              bool       `json:"is_verified"`
	HasNIB                  bool       `json:"has_nib"`
	IsLive                  bool       `json:"is_live"`
	LivePlatform            *string    `json:"live_platform,omitempty"`
	IsSaved                 bool       `json:"is_saved"`
	IsBookmarked            bool       `json:"is_bookmarked"` // Alias
	IsWishlisted            bool       `json:"is_wishlisted"`
	WishlistedProductsCount int        `json:"wishlisted_products_count"`
	SavedAt                 *time.Time `json:"saved_at,omitempty"`
	Source                  string     `json:"source"` // "bookmark", "wishlist", or "both"
}

type SavedAndWishlistedShopsResponse struct {
	SavedShopIDs          []string        `json:"saved_shop_ids"`
	WishlistedShopIDs     []string        `json:"wishlisted_shop_ids"`
	AllShopIDs            []string        `json:"all_shop_ids"`
	SavedMerchantIDs      []string        `json:"saved_merchant_ids"`
	WishlistedMerchantIDs []string        `json:"wishlisted_merchant_ids"`
	AllMerchantIDs        []string        `json:"all_merchant_ids"`
	TotalSaved            int             `json:"total_saved"`
	TotalWishlisted       int             `json:"total_wishlisted"`
	TotalShops            int             `json:"total_shops"`
	TotalMerchants        int             `json:"total_merchants"`
	Shops                 []SavedShopItem `json:"shops"`
	Merchants             []SavedShopItem `json:"merchants"`
}

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

	// Promosi & Bundling
	IsPromoted       bool    `json:"is_promoted"`
	PromotionID      *string `json:"promotion_id,omitempty"`
	PromotionName    *string `json:"promotion_name,omitempty"`
	DiscountSource   string  `json:"discount_source,omitempty"`
	IsBundle         bool    `json:"is_bundle"`
	IsPackage        bool    `json:"is_package"`
	BundleItemsCount int     `json:"bundle_items_count,omitempty"`
}

type BusinessProfileResponse struct {
	BusinessID       string     `json:"business_id"`
	BusinessName     string     `json:"business_name"`
	LogoURL          *string    `json:"logo_url"`
	Category         string     `json:"category"`
	CustomCategory   *string    `json:"custom_category,omitempty"`
	CustomTags       []string   `json:"custom_tags,omitempty"`
	Hashtags         []string   `json:"hashtags,omitempty"`
	IsOpenNow        bool       `json:"is_open_now"`
	IsManuallyClosed bool       `json:"is_manually_closed"`
	IsBookmarked     bool       `json:"is_bookmarked"`
	IsLikedByMe      bool       `json:"is_liked_by_me"`
	LikeCount        int        `json:"like_count"`
	TotalLikes       int        `json:"total_likes"`
	AverageRating    float64    `json:"average_rating"`
	TotalReviews     int64      `json:"total_reviews"`
	ReviewCount      int64      `json:"review_count"`
	IsVerified       bool       `json:"is_verified"`
	HasNIB           bool       `json:"has_nib"`
	IsLive           bool       `json:"is_live"`
	LivePlatform     *string    `json:"live_platform,omitempty"`
	LiveStartedAt    *time.Time `json:"live_started_at,omitempty"`

	AddressDetail *string  `json:"address_detail"`
	City          *string  `json:"city"`
	Province      *string  `json:"province"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`

	PhoneContact    *string  `json:"phone_contact"`
	WhatsappContact *string  `json:"whatsapp_contact"`
	SocialInstagram *string  `json:"social_instagram"`
	SocialTiktok    *string  `json:"social_tiktok"`
	SocialWebsites  []string `json:"social_websites"`

	BusinessImages []string                     `json:"business_images"`
	Certificates   []models.BusinessCertificate `json:"certificates"`

	Hours []models.BusinessHour `json:"hours"`

	ActivePromotions  []models.BusinessActivePromotionItem `json:"active_promotions"`
	RecentProducts    []SimpleProductResponse              `json:"recent_products"`
	RecentPosts       []models.BusinessPost                `json:"recent_posts"`
	SimilarBusinesses []models.Business                    `json:"similar_businesses"`
	RecentReviews     []models.Review                      `json:"recent_reviews"`
}

type Service interface {
	GetBusinessProfile(businessID string, accountID string, sourceType string, searchedTag string, productID string) (BusinessProfileResponse, error)
	ToggleBookmark(accountID, businessID string) (bool, error)
	GetBookmarks(accountID string) ([]BusinessProfileResponse, error)
	ToggleLike(accountID, businessID string) (bool, int64, error)

	GetAllProductGroups(businessID string) ([]models.ProductGroup, error)
	GetAllProducts(businessID string, groupID string) ([]SimpleProductResponse, error)
	GetAllPosts(businessID string, accountID string) ([]models.PostResponse, error)
	GetSavedAndWishlistedShops(accountID string, filter string) (SavedAndWishlistedShopsResponse, error)
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
	var isLikedByMe bool
	if accountID != "" {
		isBookmarked, _ = s.repo.CheckBookmark(accountID, businessID)
		isLikedByMe, _ = s.repo.CheckBusinessLike(accountID, businessID)
	}

	likeCount, errLikeCount := s.repo.GetBusinessLikeCount(businessID)
	if errLikeCount != nil {
		likeCount = int64(business.LikeCount)
	}

	isVerified, hasNIB, _ := s.repo.GetVerificationStatus(businessID)

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

	activePromos, _ := s.repo.GetActivePromotions(businessID)
	activePromoPosts, _ := s.repo.GetActivePromotionPosts(businessID)
	activePromotionsList := buildActivePromotionsList(activePromos, activePromoPosts)
	promoMap := buildProductPromoMap(activePromos)

	recentProducts, _ := s.repo.GetRecentProducts(businessID, 3)
	var mappedRecentProducts []SimpleProductResponse
	if recentProducts == nil {
		mappedRecentProducts = []SimpleProductResponse{}
	} else {
		for _, rp := range recentProducts {
			var promoItem *productPromoItem
			if p, ok := promoMap[rp.ProductID]; ok {
				promoItem = &p
			}
			mappedRecentProducts = append(mappedRecentProducts, mapToSimpleProduct(rp, promoItem))
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
		BusinessID:        business.BusinessID,
		BusinessName:      bName,
		LogoURL:           business.LogoURL,
		Category:          category,
		CustomCategory:    business.CustomCategory,
		CustomTags:        business.CustomTags,
		Hashtags:          business.Hashtags,
		IsOpenNow:         isOpenNow,
		IsManuallyClosed:  business.IsManuallyClosed,
		IsBookmarked:      isBookmarked,
		IsLikedByMe:       isLikedByMe,
		LikeCount:         int(likeCount),
		TotalLikes:        int(likeCount),
		AverageRating:     business.AverageRating,
		TotalReviews:      business.ReviewCount,
		ReviewCount:       business.ReviewCount,
		IsVerified:        isVerified,
		HasNIB:            hasNIB,
		IsLive:            isLive,
		LivePlatform:      livePlatform,
		LiveStartedAt:     liveStartedAt,
		AddressDetail:     business.AddressDetail,
		City:              business.City,
		Province:          business.Province,
		Latitude:          business.Latitude,
		Longitude:         business.Longitude,
		PhoneContact:      business.PhoneContact,
		WhatsappContact:   business.WhatsappContact,
		SocialInstagram:   business.SocialInstagram,
		SocialTiktok:      business.SocialTiktok,
		SocialWebsites:    business.SocialWebsites,
		BusinessImages:    business.BusinessImages,
		Certificates:      decryptBusinessCertificates(business.Certificates),
		Hours:             business.Hours,
		ActivePromotions:  activePromotionsList,
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

func (s *service) ToggleLike(accountID, businessID string) (bool, int64, error) {
	businessID = strings.TrimSpace(businessID)
	accountID = strings.TrimSpace(accountID)
	if businessID == "" {
		return false, 0, errors.New("business not found")
	}
	return s.repo.ToggleBusinessLike(accountID, businessID)
}

func (s *service) GetBookmarks(accountID string) ([]BusinessProfileResponse, error) {
	businesses, err := s.repo.GetBookmarkedBusinesses(accountID)
	if err != nil {
		return nil, err
	}

	var bizIDs []string
	for _, b := range businesses {
		bizIDs = append(bizIDs, b.BusinessID)
	}
	verifMap, _ := s.repo.BatchGetVerificationStatus(bizIDs)

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

		var isVerif, hasNIB bool
		if v, ok := verifMap[business.BusinessID]; ok {
			isVerif = v.IsVerified
			hasNIB = v.HasNIB
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
			BusinessID:       business.BusinessID,
			BusinessName:     bName,
			LogoURL:          business.LogoURL,
			Category:         category,
			CustomCategory:   business.CustomCategory,
			CustomTags:       business.CustomTags,
			Hashtags:         business.Hashtags,
			IsOpenNow:        isOpenNow,
			IsManuallyClosed: business.IsManuallyClosed,
			IsBookmarked:     true, // always true if fetched from bookmarks
			LikeCount:        business.LikeCount,
			TotalLikes:       business.LikeCount,
			AverageRating:    business.AverageRating,
			TotalReviews:     business.ReviewCount,
			ReviewCount:      business.ReviewCount,
			IsVerified:       isVerif,
			HasNIB:           hasNIB,
			IsLive:           isLive,
			LivePlatform:     livePlatform,
			LiveStartedAt:    liveStartedAt,
			AddressDetail:    business.AddressDetail,
			City:             business.City,
			Province:         business.Province,
			Latitude:         business.Latitude,
			Longitude:        business.Longitude,
			PhoneContact:     business.PhoneContact,
			WhatsappContact:  business.WhatsappContact,
			SocialInstagram:  business.SocialInstagram,
			SocialTiktok:     business.SocialTiktok,
			SocialWebsites:   business.SocialWebsites,
			BusinessImages:   business.BusinessImages,
			Certificates:     decryptBusinessCertificates(business.Certificates),
			Hours:            business.Hours,
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

type productPromoItem struct {
	PromotionID    string
	PromotionName  string
	DiscountSource string
	DiscountType   string
	Discount       float64
	MaxDiscount    *float64
}

func buildProductPromoMap(promos []models.Promotion) map[string]productPromoItem {
	promoMap := make(map[string]productPromoItem)
	for _, promo := range promos {
		promoTitle := "Promo Spesial Toko"
		if promo.PackageName != nil && strings.TrimSpace(*promo.PackageName) != "" {
			promoTitle = strings.TrimSpace(*promo.PackageName)
		} else if strings.TrimSpace(promo.Caption) != "" {
			promoTitle = strings.TrimSpace(promo.Caption)
		} else if promo.Post != nil && strings.TrimSpace(promo.Post.Content) != "" {
			promoTitle = strings.TrimSpace(promo.Post.Content)
		}

		for _, pp := range promo.Products {
			if _, exists := promoMap[pp.ProductID]; !exists {
				discSource := pp.DiscountSource
				if discSource == "" {
					discSource = "promotion"
				}
				promoMap[pp.ProductID] = productPromoItem{
					PromotionID:    promo.PromotionID,
					PromotionName:  promoTitle,
					DiscountSource: discSource,
					DiscountType:   pp.DiscountType,
					Discount:       pp.Discount,
					MaxDiscount:    pp.MaxDiscount,
				}
			}
		}

		if promo.MainProductID != "" {
			if _, exists := promoMap[promo.MainProductID]; !exists {
				promoMap[promo.MainProductID] = productPromoItem{
					PromotionID:    promo.PromotionID,
					PromotionName:  promoTitle,
					DiscountSource: "promotion",
				}
			}
		}
	}
	return promoMap
}

func buildActivePromotionsList(promos []models.Promotion, posts []models.BusinessPost) []models.BusinessActivePromotionItem {
	var list []models.BusinessActivePromotionItem
	seenPromoIDs := make(map[string]bool)

	for _, p := range promos {
		if seenPromoIDs[p.PromotionID] {
			continue
		}
		seenPromoIDs[p.PromotionID] = true

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

		list = append(list, models.BusinessActivePromotionItem{
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

	for _, post := range posts {
		postPromoID := "POST-" + post.PostID
		if seenPromoIDs[postPromoID] {
			continue
		}
		seenPromoIDs[postPromoID] = true

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

		list = append(list, models.BusinessActivePromotionItem{
			PromotionID:   postPromoID,
			Title:         title,
			Caption:       post.Content,
			BannerURL:     banner,
			PromotionType: "post_promo",
			StartAt:       post.CreatedAt,
			EndAt:         endAt,
		})
	}

	if list == nil {
		list = []models.BusinessActivePromotionItem{}
	}
	return list
}

func mapToSimpleProduct(p models.Product, promo *productPromoItem) SimpleProductResponse {
	var originalPrice *float64
	currentPrice := p.Price
	var discountAmount float64
	var discountPct float64
	isDiscounted := false
	discountType := p.DiscountType

	isPromoted := false
	var promoID *string
	var promoName *string
	discountSource := ""

	if promo != nil {
		isPromoted = true
		promoID = &promo.PromotionID
		promoName = &promo.PromotionName
		discountSource = promo.DiscountSource
		if discountSource == "" {
			discountSource = "promotion"
		}

		if promo.DiscountType != "" && promo.Discount > 0 {
			cut, pct := utils.CalculateProductDiscount(p.Price, promo.DiscountType, promo.Discount, promo.MaxDiscount)
			if cut > 0 {
				isDiscounted = true
				currentPrice = p.Price - cut
				orig := p.Price
				originalPrice = &orig
				discountAmount = cut
				discountPct = pct
				discountType = promo.DiscountType
			}
		}
	}

	if !isDiscounted && utils.IsDiscountActive(p.Discount, p.DiscountStartAt, p.DiscountEndAt, time.Now()) {
		cut, pct := utils.CalculateProductDiscount(p.Price, p.DiscountType, p.Discount, p.MaxDiscount)
		if cut > 0 {
			isDiscounted = true
			currentPrice = p.Price - cut
			orig := p.Price
			originalPrice = &orig
			discountAmount = cut
			discountPct = pct
			discountSource = "product"
		}
	}

	stockStatus := "Out of Stock"
	if p.StockType == "always_available" || p.StockType == "in_stock" {
		stockStatus = "In Stock"
	}

	isBundle := false
	isPackage := p.IsPackage
	bundleItemsCount := 0

	if p.IsPackage {
		isBundle = true
		bundleItemsCount = len(p.PackageItems)
		if p.OriginalPackagePrice != nil && *p.OriginalPackagePrice > p.Price {
			isDiscounted = true
			orig := *p.OriginalPackagePrice
			originalPrice = &orig
			discountAmount = *p.OriginalPackagePrice - p.Price
			if p.PackageSavingsPercentage != nil && *p.PackageSavingsPercentage > 0 {
				discountPct = *p.PackageSavingsPercentage
			} else if *p.OriginalPackagePrice > 0 {
				discountPct = math.Round((discountAmount / *p.OriginalPackagePrice) * 10000) / 100
			}
			discountType = "nominal"
			if discountSource == "" {
				discountSource = "package"
			}
		}
		for _, item := range p.PackageItems {
			if item.ItemProduct != nil && item.ItemProduct.StockType == "out_of_stock" {
				stockStatus = "Out of Stock"
			}
		}
	}

	return SimpleProductResponse{
		ProductID:        p.ProductID,
		ProductName:      p.ProductName,
		ProductPreview:   p.ProductPreview,
		Price:            currentPrice,
		OriginalPrice:    originalPrice,
		IsDiscounted:     isDiscounted,
		DiscountType:     discountType,
		DiscountAmount:   discountAmount,
		DiscountPct:      discountPct,
		AverageRating:    p.AverageRating,
		ReviewCount:      p.ReviewCount,
		StockStatus:      stockStatus,
		IsPromoted:       isPromoted,
		PromotionID:      promoID,
		PromotionName:    promoName,
		DiscountSource:   discountSource,
		IsBundle:         isBundle,
		IsPackage:        isPackage,
		BundleItemsCount: bundleItemsCount,
	}
}

func (s *service) GetAllProducts(businessID string, groupID string) ([]SimpleProductResponse, error) {
	products, err := s.repo.GetAllProducts(businessID, groupID)
	if err != nil {
		return nil, err
	}

	activePromos, _ := s.repo.GetActivePromotions(businessID)
	promoMap := buildProductPromoMap(activePromos)

	var simpleProducts []SimpleProductResponse

	// Jika melihat grup "Promosi" atau katalog umum, sertakan paket bundling aktif di paling atas
	if groupID == "promotion" || groupID == "" {
		bundlingPromos, _ := s.repo.GetActiveBundlingPromotions(businessID)
		for _, bp := range bundlingPromos {
			pkgName := ""
			if bp.PackageName != nil {
				pkgName = *bp.PackageName
			}
			previews := []string{}
			if bp.PackageImage != nil && *bp.PackageImage != "" {
				previews = []string{*bp.PackageImage}
			}
			stockStatus := "in_stock"
			for _, item := range bp.BundleItems {
				if item.Product != nil && len(previews) == 0 && len(item.Product.ProductPreview) > 0 {
					previews = append(previews, item.Product.ProductPreview[0])
				}
				if item.Product != nil && item.Product.StockType == "out_of_stock" {
					stockStatus = "out_of_stock"
				}
			}

			var pkgPrice, origPrice, savAmt, savPct float64
			if bp.PackagePrice != nil {
				pkgPrice = *bp.PackagePrice
			}
			if bp.OriginalTotalPrice != nil {
				origPrice = *bp.OriginalTotalPrice
			}
			if bp.SavingsAmount != nil {
				savAmt = *bp.SavingsAmount
			}
			if bp.SavingsPercentage != nil {
				savPct = *bp.SavingsPercentage
			}

			promoID := bp.PromotionID
			promoTitle := "Promo Bundling Hemat"
			if bp.PackageName != nil && strings.TrimSpace(*bp.PackageName) != "" {
				promoTitle = strings.TrimSpace(*bp.PackageName)
			}
			simpleProducts = append(simpleProducts, SimpleProductResponse{
				ProductID:        bp.PromotionID,
				ProductName:      pkgName,
				ProductPreview:   previews,
				Price:            pkgPrice,
				OriginalPrice:    &origPrice,
				IsDiscounted:     true,
				DiscountType:     "nominal",
				DiscountAmount:   savAmt,
				DiscountPct:      savPct,
				StockStatus:      stockStatus,
				IsPromoted:       true,
				PromotionID:      &promoID,
				PromotionName:    &promoTitle,
				DiscountSource:   "promotion",
				IsBundle:         true,
				BundleItemsCount: len(bp.BundleItems),
			})
		}
	}

	for _, p := range products {
		var promoItem *productPromoItem
		if promo, ok := promoMap[p.ProductID]; ok {
			promoItem = &promo
		}
		simpleProducts = append(simpleProducts, mapToSimpleProduct(p, promoItem))
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

func (s *service) GetSavedAndWishlistedShops(accountID string, filter string) (SavedAndWishlistedShopsResponse, error) {
	emptyResp := SavedAndWishlistedShopsResponse{
		SavedShopIDs:          []string{},
		WishlistedShopIDs:     []string{},
		AllShopIDs:            []string{},
		SavedMerchantIDs:      []string{},
		WishlistedMerchantIDs: []string{},
		AllMerchantIDs:        []string{},
		TotalSaved:            0,
		TotalWishlisted:       0,
		TotalShops:            0,
		TotalMerchants:        0,
		Shops:                 []SavedShopItem{},
		Merchants:             []SavedShopItem{},
	}

	if accountID == "" {
		return emptyResp, nil
	}

	raw, err := s.repo.GetSavedAndWishlistedBizIDs(accountID)
	if err != nil {
		return emptyResp, err
	}

	var savedShopIDs []string
	for id := range raw.SavedBizMap {
		savedShopIDs = append(savedShopIDs, id)
	}
	if savedShopIDs == nil {
		savedShopIDs = []string{}
	}

	var wishlistedShopIDs []string
	for id := range raw.WishlistedBizMap {
		wishlistedShopIDs = append(wishlistedShopIDs, id)
	}
	if wishlistedShopIDs == nil {
		wishlistedShopIDs = []string{}
	}

	// Filter logic: "all", "saved", "wishlisted"
	targetBizIDMap := make(map[string]bool)
	switch filter {
	case "saved", "bookmark", "bookmarks":
		for _, id := range savedShopIDs {
			targetBizIDMap[id] = true
		}
	case "wishlisted", "wishlist", "wishlists":
		for _, id := range wishlistedShopIDs {
			targetBizIDMap[id] = true
		}
	default: // "all"
		for _, id := range savedShopIDs {
			targetBizIDMap[id] = true
		}
		for _, id := range wishlistedShopIDs {
			targetBizIDMap[id] = true
		}
	}

	var allBizIDs []string
	for id := range targetBizIDMap {
		allBizIDs = append(allBizIDs, id)
	}
	if allBizIDs == nil {
		allBizIDs = []string{}
	}

	if len(allBizIDs) == 0 {
		return SavedAndWishlistedShopsResponse{
			SavedShopIDs:          savedShopIDs,
			WishlistedShopIDs:     wishlistedShopIDs,
			AllShopIDs:            allBizIDs,
			SavedMerchantIDs:      savedShopIDs,
			WishlistedMerchantIDs: wishlistedShopIDs,
			AllMerchantIDs:        allBizIDs,
			TotalSaved:            len(savedShopIDs),
			TotalWishlisted:       len(wishlistedShopIDs),
			TotalShops:            0,
			TotalMerchants:        0,
			Shops:                 []SavedShopItem{},
			Merchants:             []SavedShopItem{},
		}, nil
	}

	businesses, err := s.repo.GetBusinessesByIDs(allBizIDs)
	if err != nil {
		return emptyResp, err
	}

	verifMap, _ := s.repo.BatchGetVerificationStatus(allBizIDs)

	var shops []SavedShopItem
	for _, business := range businesses {
		category := "General"
		if len(business.Categories) > 0 {
			category = business.Categories[0].Name
		} else if business.CustomCategory != nil && strings.TrimSpace(*business.CustomCategory) != "" {
			category = *business.CustomCategory
		}

		isOpenNow := utils.CalculateStoreStatus(business.IsManuallyClosed, business.Hours, business.ConfirmedOpenDate, business.ConfirmedOvertimeDate)

		bName := ""
		if business.BusinessName != nil {
			bName = *business.BusinessName
		}

		var isVerif, hasNIB bool
		if v, ok := verifMap[business.BusinessID]; ok {
			isVerif = v.IsVerified
			hasNIB = v.HasNIB
		}

		isLive := business.IsLive
		var livePlatform *string
		if isLive {
			if business.LiveStartedAt != nil && time.Since(*business.LiveStartedAt) > 4*time.Hour {
				isLive = false
			} else {
				livePlatform = business.LivePlatform
			}
		}

		savedTime, isSaved := raw.SavedBizMap[business.BusinessID]
		wCount, isWishlisted := raw.WishlistedBizMap[business.BusinessID]

		source := "bookmark"
		if isSaved && isWishlisted {
			source = "both"
		} else if isWishlisted {
			source = "wishlist"
		}

		var savedAtPtr *time.Time
		if isSaved {
			savedAtPtr = &savedTime
		} else if latestW, ok := raw.WishlistLatestAt[business.BusinessID]; ok {
			savedAtPtr = &latestW
		}

		shops = append(shops, SavedShopItem{
			BusinessID:              business.BusinessID,
			ShopID:                  business.BusinessID,
			ID:                      business.BusinessID,
			BusinessName:            bName,
			Name:                    bName,
			LogoURL:                 business.LogoURL,
			Category:                category,
			CustomCategory:          business.CustomCategory,
			City:                    business.City,
			Province:                business.Province,
			AddressDetail:           business.AddressDetail,
			Latitude:                business.Latitude,
			Longitude:               business.Longitude,
			AverageRating:           business.AverageRating,
			Rating:                  business.AverageRating,
			TotalReviews:            business.ReviewCount,
			ReviewCount:             business.ReviewCount,
			TotalLikes:              business.LikeCount,
			LikeCount:               business.LikeCount,
			IsOpenNow:               isOpenNow,
			IsManuallyClosed:        business.IsManuallyClosed,
			IsVerified:              isVerif,
			HasNIB:                  hasNIB,
			IsLive:                  isLive,
			LivePlatform:            livePlatform,
			IsSaved:                 isSaved,
			IsBookmarked:            isSaved,
			IsWishlisted:            isWishlisted,
			WishlistedProductsCount: wCount,
			SavedAt:                 savedAtPtr,
			Source:                  source,
		})
	}

	if shops == nil {
		shops = []SavedShopItem{}
	} else {
		// Sort: toko yang memiliki SavedAt terbaru di urutan pertama
		sort.Slice(shops, func(i, j int) bool {
			if shops[i].SavedAt == nil && shops[j].SavedAt == nil {
				return shops[i].AverageRating > shops[j].AverageRating
			}
			if shops[i].SavedAt == nil {
				return false
			}
			if shops[j].SavedAt == nil {
				return true
			}
			return shops[i].SavedAt.After(*shops[j].SavedAt)
		})
	}

	return SavedAndWishlistedShopsResponse{
		SavedShopIDs:          savedShopIDs,
		WishlistedShopIDs:     wishlistedShopIDs,
		AllShopIDs:            allBizIDs,
		SavedMerchantIDs:      savedShopIDs,
		WishlistedMerchantIDs: wishlistedShopIDs,
		AllMerchantIDs:        allBizIDs,
		TotalSaved:            len(savedShopIDs),
		TotalWishlisted:       len(wishlistedShopIDs),
		TotalShops:            len(shops),
		TotalMerchants:        len(shops),
		Shops:                 shops,
		Merchants:             shops,
	}, nil
}
