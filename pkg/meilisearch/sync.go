package meilisearch

import (
	"log"
	"math"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"gorm.io/gorm"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type MatchedProductPreview struct {
	ProductID      string   `json:"product_id"`
	ProductName    string   `json:"product_name"`
	ProductPreview []string `json:"product_preview,omitempty"`
	Price          float64  `json:"price"`
	FinalPrice     float64  `json:"final_price"`
	IsDiscounted   bool     `json:"is_discounted"`
	DiscountPct    float64  `json:"discount_pct,omitempty"`
	StockType      string   `json:"stock_type"`
}

type MeiliBusinessHour struct {
	DayOfWeek int    `json:"day_of_week"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed  bool   `json:"is_closed"`
}

type MeiliBusinessDocument struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Products         []string            `json:"products"`
	Rating           float64             `json:"rating"`
	ReviewCount      int64               `json:"review_count"`
	TotalReviews     int64               `json:"total_reviews"`
	LikeCount        int                 `json:"like_count"`
	TotalLikes       int                 `json:"total_likes"`
	CreatedAt        int64               `json:"created_at,omitempty"`
	PreviewImages    []string            `json:"preview_images"`
	IsOpen           bool                `json:"is_open"`
	IsManuallyClosed bool                `json:"is_manually_closed"`
	CloseTime        string              `json:"close_time"`
	Hours            []MeiliBusinessHour `json:"hours"`
	Geo              GeoLocation         `json:"_geo"`
	Latitude         *float64            `json:"latitude"`
	Longitude        *float64            `json:"longitude"`
	Lat              *float64            `json:"lat,omitempty"`
	Lng              *float64            `json:"lng,omitempty"`
	Distance         float64             `json:"_geoDistance,omitempty"`
	IsWithinRadius   bool                `json:"is_within_radius"`

	// Category fields
	CategoryID    int64    `json:"category_id,omitempty"`
	CategoryIDs   []int64  `json:"category_ids"`
	CategoryNames []string `json:"category_names"`
	Categories    []string `json:"categories,omitempty"` // alias untuk kompatibilitas

	// Tag & Hashtag fields
	TagIDs     []int64  `json:"tag_ids"`
	TagNames   []string `json:"tag_names"`
	Tags       []string `json:"tags,omitempty"` // alias untuk kompatibilitas
	CustomTags []string `json:"custom_tags,omitempty"`
	Hashtags   []string `json:"hashtags,omitempty"`

	// Enhanced Gatekeeping & Scope fields
	StoreType       string                  `json:"store_type"`
	StoreTypeLabel  string                  `json:"store_type_label"`
	HasPromo        bool                    `json:"has_promo"`
	MinPrice        float64                 `json:"min_price"`
	MaxPrice        float64                 `json:"max_price"`
	StockTypes      []string                `json:"stock_types"`
	City            string                  `json:"city"`
	AddressDetail   string                  `json:"address_detail"`
	MatchedProducts []MatchedProductPreview `json:"matched_products,omitempty"`
}

type MeiliProductDocument struct {
	ID             string      `json:"id,omitempty"`
	ProductID      string      `json:"product_id"`
	ProductName    string      `json:"product_name"`
	PackageName    string      `json:"package_name,omitempty"`
	ProductPreview []string    `json:"product_preview"`
	Price          float64     `json:"price"`
	FinalPrice     float64     `json:"final_price"`
	IsDiscounted   bool        `json:"is_discounted"`
	DiscountPct    float64     `json:"discount_pct"`
	DiscountAmount float64     `json:"discount_amount,omitempty"`
	StockType      string      `json:"stock_type"`
	BusinessID     string      `json:"business_id"`
	BusinessName   string      `json:"business_name"`
	StoreType      string      `json:"store_type"`
	StoreTypeLabel string      `json:"store_type_label"`
	IsOpen         bool        `json:"is_open"`
	Rating         float64     `json:"rating"`
	ReviewCount    int64       `json:"review_count"`
	Hashtags       []string    `json:"hashtags"`
	CategoryIDs    []int64     `json:"category_ids,omitempty"`
	CategoryNames  []string    `json:"category_names,omitempty"`
	Geo            GeoLocation `json:"_geo"`
	Latitude       *float64    `json:"latitude"`
	Longitude      *float64    `json:"longitude"`
	Lat            *float64    `json:"lat,omitempty"`
	Lng            *float64    `json:"lng,omitempty"`
	Distance       float64     `json:"_geoDistance,omitempty"`
	IsWithinRadius bool        `json:"is_within_radius"`
	CreatedAt      int64       `json:"created_at,omitempty"`

	// Promotional & Bundling Fields
	IsPackage     bool   `json:"is_package"`
	PromotionType string `json:"promotion_type,omitempty"`
	PromotionID   string `json:"promotion_id,omitempty"`
	HasPromo      bool   `json:"has_promo"`
}

// InitMeilisearchIndices sets filterable, sortable, searchable attributes, synonyms, and typo tolerance
func InitMeilisearchIndices() {
	if Client == nil {
		return
	}

	synonyms := GetIndonesianSynonyms()
	typoTolerance := &meilisearch.TypoTolerance{
		Enabled: true,
		MinWordSizeForTypos: meilisearch.MinWordSizeForTypos{
			OneTypo:  4,
			TwoTypos: 8,
		},
	}

	// 1. Indeks "businesses"
	bizIndex := Client.Index("businesses")
	bizFilterable := &[]interface{}{
		"_geo",
		"rating",
		"is_open",
		"is_manually_closed",
		"category_id",
		"category_ids",
		"category_names",
		"tag_ids",
		"tag_names",
		"hashtags",
		"store_type",
		"has_promo",
		"min_price",
		"max_price",
		"stock_types",
		"total_likes",
		"like_count",
	}
	bizSortable := &[]string{"_geo", "rating", "review_count", "min_price", "total_likes", "like_count", "created_at"}
	bizSearchable := &[]string{
		"name",
		"products",
		"category_names",
		"tag_names",
		"custom_tags",
		"hashtags",
		"store_type",
		"store_type_label",
		"city",
		"address_detail",
	}

	if _, err := bizIndex.UpdateFilterableAttributes(bizFilterable); err != nil {
		log.Printf("⚠️ Gagal set filterable attributes businesses: %v", err)
	}
	if _, err := bizIndex.UpdateSortableAttributes(bizSortable); err != nil {
		log.Printf("⚠️ Gagal set sortable attributes businesses: %v", err)
	}
	if _, err := bizIndex.UpdateSearchableAttributes(bizSearchable); err != nil {
		log.Printf("⚠️ Gagal set searchable attributes businesses: %v", err)
	}
	if _, err := bizIndex.UpdateSynonyms(&synonyms); err != nil {
		log.Printf("⚠️ Gagal set synonyms businesses: %v", err)
	}
	if _, err := bizIndex.UpdateTypoTolerance(typoTolerance); err != nil {
		log.Printf("⚠️ Gagal set typo tolerance businesses: %v", err)
	}

	// 2. Indeks "products"
	primaryKey := "id"
	if _, err := Client.GetIndex("products"); err != nil {
		_, _ = Client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        "products",
			PrimaryKey: primaryKey,
		})
	} else {
		idx, _ := Client.GetIndex("products")
		if idx != nil && idx.PrimaryKey == "" {
			_, _ = idx.UpdateIndex(&meilisearch.UpdateIndexRequestParams{PrimaryKey: primaryKey})
		}
	}
	prodIndex := Client.Index("products")
	prodFilterable := &[]interface{}{
		"_geo",
		"price",
		"final_price",
		"is_discounted",
		"has_promo",
		"is_package",
		"promotion_type",
		"stock_type",
		"is_open",
		"store_type",
		"business_id",
		"category_ids",
		"category_names",
		"hashtags",
		"rating",
	}
	prodSortable := &[]string{"_geo", "final_price", "price", "rating", "created_at"}
	prodSearchable := &[]string{
		"product_name",
		"package_name",
		"business_name",
		"category_names",
		"hashtags",
		"store_type_label",
	}

	if _, err := prodIndex.UpdateFilterableAttributes(prodFilterable); err != nil {
		log.Printf("⚠️ Gagal set filterable attributes products: %v", err)
	}
	if _, err := prodIndex.UpdateSortableAttributes(prodSortable); err != nil {
		log.Printf("⚠️ Gagal set sortable attributes products: %v", err)
	}
	if _, err := prodIndex.UpdateSearchableAttributes(prodSearchable); err != nil {
		log.Printf("⚠️ Gagal set searchable attributes products: %v", err)
	}
	if _, err := prodIndex.UpdateSynonyms(&synonyms); err != nil {
		log.Printf("⚠️ Gagal set synonyms products: %v", err)
	}
	if _, err := prodIndex.UpdateTypoTolerance(typoTolerance); err != nil {
		log.Printf("⚠️ Gagal set typo tolerance products: %v", err)
	}
}

// BuildBusinessDocument creates a MeiliBusinessDocument from a Business model
func BuildBusinessDocument(db *gorm.DB, b *models.Business) MeiliBusinessDocument {
	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)
	currentDayInt := int(now.Weekday())

	// Fetch active products
	var products []models.Product
	if db != nil {
		db.Where("business_id = ? AND is_active = ?", b.BusinessID, true).Find(&products)
	}

	productNames := make([]string, 0, len(products))
	matchedPreviews := make([]MatchedProductPreview, 0, len(products))
	stockTypeMap := make(map[string]bool)
	var hasPromo bool
	var minPrice, maxPrice float64

	for i, p := range products {
		if p.ProductName != "" {
			productNames = append(productNames, p.ProductName)
		}
		if p.StockType != "" {
			stockTypeMap[p.StockType] = true
		}

		currentPrice := p.Price
		isDiscounted := false
		var discountPct float64

		if utils.IsDiscountActive(p.Discount, p.DiscountStartAt, p.DiscountEndAt, now) {
			cut, pct := utils.CalculateProductDiscount(p.Price, p.DiscountType, p.Discount, p.MaxDiscount)
			if cut > 0 {
				isDiscounted = true
				currentPrice = p.Price - cut
				discountPct = pct
				hasPromo = true
			}
		}

		if i == 0 || currentPrice < minPrice {
			minPrice = currentPrice
		}
		if currentPrice > maxPrice {
			maxPrice = currentPrice
		}

		matchedPreviews = append(matchedPreviews, MatchedProductPreview{
			ProductID:      p.ProductID,
			ProductName:    p.ProductName,
			ProductPreview: p.ProductPreview,
			Price:          p.Price,
			FinalPrice:     currentPrice,
			IsDiscounted:   isDiscounted,
			DiscountPct:    discountPct,
			StockType:      p.StockType,
		})
	}

	// Check active promotions in promotions table if not already detected from products
	if !hasPromo && db != nil {
		var promoCount int64
		db.Table("promotions").Where("business_id = ? AND status = 'active'", b.BusinessID).Count(&promoCount)
		if promoCount > 0 {
			hasPromo = true
		}
	}

	// Fetch categories if not preloaded
	categories := b.Categories
	if len(categories) == 0 && db != nil {
		db.Table("business_categories").
			Joins("JOIN business_has_categories ON business_has_categories.category_id = business_categories.category_id").
			Where("business_has_categories.business_id = ?", b.BusinessID).
			Find(&categories)
	}

	categoryIDs := make([]int64, 0, len(categories))
	categoryNames := make([]string, 0, len(categories))
	for _, c := range categories {
		categoryIDs = append(categoryIDs, c.CategoryID)
		if c.Name != "" {
			categoryNames = append(categoryNames, c.Name)
		}
	}
	if b.CustomCategory != nil && strings.TrimSpace(*b.CustomCategory) != "" {
		categoryNames = append(categoryNames, strings.TrimSpace(*b.CustomCategory))
	}
	var primaryCatID int64
	if len(categoryIDs) > 0 {
		primaryCatID = categoryIDs[0]
	}

	// Fetch tags if not preloaded
	tags := b.Tags
	if len(tags) == 0 && db != nil {
		db.Table("business_tags").
			Joins("JOIN business_has_tags ON business_has_tags.tag_id = business_tags.tag_id").
			Where("business_has_tags.business_id = ?", b.BusinessID).
			Find(&tags)
	}

	tagIDs := make([]int64, 0, len(tags))
	tagNames := make([]string, 0, len(tags))
	for _, t := range tags {
		tagIDs = append(tagIDs, t.TagID)
		if t.Name != "" {
			tagNames = append(tagNames, t.Name)
		}
	}
	for _, ct := range b.CustomTags {
		if trimmed := strings.TrimSpace(ct); trimmed != "" {
			tagNames = append(tagNames, trimmed)
		}
	}
	for _, ht := range b.Hashtags {
		if trimmed := strings.TrimSpace(ht); trimmed != "" {
			tagNames = append(tagNames, trimmed)
		}
	}

	// Hours
	hours := b.Hours
	if len(hours) == 0 && db != nil {
		db.Where("business_id = ?", b.BusinessID).Order("day_of_week ASC").Find(&hours)
	}

	meiliHours := make([]MeiliBusinessHour, 0, len(hours))
	for _, h := range hours {
		openStr := ""
		if h.OpenTime != nil {
			openStr = *h.OpenTime
		}
		closeStr := ""
		if h.CloseTime != nil {
			closeStr = *h.CloseTime
		}
		isClosed := h.IsClosed || openStr == "" || closeStr == ""
		meiliHours = append(meiliHours, MeiliBusinessHour{
			DayOfWeek: h.DayOfWeek,
			OpenTime:  openStr,
			CloseTime: closeStr,
			IsClosed:  isClosed,
		})
	}

	// Calculate IsOpen and CloseTime
	isOpen := utils.CalculateStoreStatus(b.IsManuallyClosed, hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)
	closeTime := ""
	for _, h := range hours {
		if h.DayOfWeek == currentDayInt && h.CloseTime != nil {
			closeTime = *h.CloseTime
			break
		}
	}

	name := ""
	if b.BusinessName != nil {
		name = *b.BusinessName
	}

	previewImages := b.BusinessImages
	if previewImages == nil {
		previewImages = []string{}
	}

	storeType := "regular"
	if b.StoreType != nil && strings.TrimSpace(*b.StoreType) != "" {
		storeType = models.NormalizeStoreType(*b.StoreType)
	}
	storeTypeLabel := models.GetStoreTypeLabel(storeType)

	stockTypes := make([]string, 0, len(stockTypeMap))
	for st := range stockTypeMap {
		stockTypes = append(stockTypes, st)
	}

	city := ""
	if b.City != nil {
		city = *b.City
	}
	addressDetail := ""
	if b.AddressDetail != nil {
		addressDetail = *b.AddressDetail
	}

	doc := MeiliBusinessDocument{
		ID:               b.BusinessID,
		Name:             name,
		Products:         productNames,
		Rating:           b.AverageRating,
		ReviewCount:      b.ReviewCount,
		TotalReviews:     b.ReviewCount,
		LikeCount:        b.LikeCount,
		TotalLikes:       b.LikeCount,
		CreatedAt:        b.CreatedAt.Unix(),
		PreviewImages:    previewImages,
		IsOpen:           isOpen,
		IsManuallyClosed: b.IsManuallyClosed,
		CloseTime:        closeTime,
		Hours:            meiliHours,
		CategoryID:      primaryCatID,
		CategoryIDs:     categoryIDs,
		CategoryNames:   categoryNames,
		Categories:      categoryNames,
		TagIDs:          tagIDs,
		TagNames:        tagNames,
		Tags:            tagNames,
		CustomTags:      b.CustomTags,
		Hashtags:        b.Hashtags,
		StoreType:       storeType,
		StoreTypeLabel:  storeTypeLabel,
		HasPromo:        hasPromo,
		MinPrice:        minPrice,
		MaxPrice:        maxPrice,
		StockTypes:      stockTypes,
		City:            city,
		AddressDetail:   addressDetail,
		MatchedProducts: matchedPreviews,
	}

	if b.Latitude != nil && b.Longitude != nil {
		doc.Geo = GeoLocation{
			Lat: *b.Latitude,
			Lng: *b.Longitude,
		}
		doc.Latitude = b.Latitude
		doc.Longitude = b.Longitude
		doc.Lat = b.Latitude
		doc.Lng = b.Longitude
	}

	return doc
}

// BuildProductDocument creates a MeiliProductDocument from a Product and its Business
func BuildProductDocument(db *gorm.DB, p *models.Product, b *models.Business) MeiliProductDocument {
	if b == nil && db != nil {
		var biz models.Business
		if err := db.Preload("Hours").Preload("Categories").Where("business_id = ?", p.BusinessID).First(&biz).Error; err == nil {
			b = &biz
		}
	}

	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)

	currentPrice := p.Price
	isDiscounted := false
	var discountPct float64
	var discountAmount float64

	promotionID := ""
	promotionType := ""

	// 1. Cek diskon langsung di tabel products
	if utils.IsDiscountActive(p.Discount, p.DiscountStartAt, p.DiscountEndAt, now) {
		cut, pct := utils.CalculateProductDiscount(p.Price, p.DiscountType, p.Discount, p.MaxDiscount)
		if cut > 0 {
			isDiscounted = true
			currentPrice = p.Price - cut
			discountAmount = cut
			discountPct = pct
			promotionType = "catalog"
		}
	}

	// 2. Cek apakah ada program promosi aktif di tabel promotions (per_product/catalog)
	if db != nil {
		type promoResult struct {
			PromotionID   string   `gorm:"column:promotion_id"`
			PromotionType string   `gorm:"column:promotion_type"`
			DiscountType  string   `gorm:"column:discount_type"`
			Discount      float64  `gorm:"column:discount"`
			MaxDiscount   *float64 `gorm:"column:max_discount"`
		}
		var pr promoResult
		err := db.Table("promotion_products pp").
			Select("pp.promotion_id, pp.discount_type, pp.discount, pp.max_discount, prm.promotion_type").
			Joins("JOIN promotions prm ON prm.promotion_id = pp.promotion_id").
			Where("pp.product_id = ? AND prm.status = ? AND prm.start_at <= ? AND prm.end_at >= ?", p.ProductID, models.PromotionStatusActive, now, now).
			Order("prm.start_at DESC").
			Limit(1).
			Scan(&pr).Error

		if err == nil && pr.PromotionID != "" {
			cut, pct := utils.CalculateProductDiscount(p.Price, pr.DiscountType, pr.Discount, pr.MaxDiscount)
			if cut > 0 {
				isDiscounted = true
				currentPrice = p.Price - cut
				discountAmount = cut
				discountPct = pct
				promotionID = pr.PromotionID
				promotionType = "product"
			}
		}
	}

	if p.IsPackage && p.OriginalPackagePrice != nil && *p.OriginalPackagePrice > p.Price {
		isDiscounted = true
		discountAmount = *p.OriginalPackagePrice - p.Price
		if p.PackageSavingsPercentage != nil && *p.PackageSavingsPercentage > 0 {
			discountPct = *p.PackageSavingsPercentage
		} else if *p.OriginalPackagePrice > 0 {
			discountPct = math.Round((discountAmount / *p.OriginalPackagePrice) * 10000) / 100
		}
		promotionType = "package"
	}
	hasPromo := isDiscounted

	bizName := ""
	storeType := "regular"
	storeTypeLabel := "Regular"
	isOpen := false
	rating := 0.0
	var reviewCount int64
	var categoryIDs []int64
	var categoryNames []string
	var geo GeoLocation
	var latPtr, lngPtr *float64

	if b != nil {
		if b.BusinessName != nil {
			bizName = *b.BusinessName
		}
		if b.StoreType != nil {
			storeType = models.NormalizeStoreType(*b.StoreType)
			storeTypeLabel = models.GetStoreTypeLabel(storeType)
		}
		isOpen = utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)
		rating = b.AverageRating
		reviewCount = b.ReviewCount

		for _, c := range b.Categories {
			categoryIDs = append(categoryIDs, c.CategoryID)
			if c.Name != "" {
				categoryNames = append(categoryNames, c.Name)
			}
		}
		if b.CustomCategory != nil && strings.TrimSpace(*b.CustomCategory) != "" {
			categoryNames = append(categoryNames, strings.TrimSpace(*b.CustomCategory))
		}

		if b.Latitude != nil && b.Longitude != nil {
			geo = GeoLocation{
				Lat: *b.Latitude,
				Lng: *b.Longitude,
			}
			latPtr = b.Latitude
			lngPtr = b.Longitude
		}
	}

	preview := p.ProductPreview
	if preview == nil {
		preview = []string{}
	}

	hashtags := p.Tags
	if hashtags == nil {
		hashtags = []string{}
	}

	return MeiliProductDocument{
		ID:             p.ProductID,
		ProductID:      p.ProductID,
		ProductName:    p.ProductName,
		ProductPreview: preview,
		Price:          p.Price,
		FinalPrice:     currentPrice,
		IsDiscounted:   isDiscounted,
		DiscountPct:    discountPct,
		DiscountAmount: discountAmount,
		StockType:      p.StockType,
		BusinessID:     p.BusinessID,
		BusinessName:   bizName,
		StoreType:      storeType,
		StoreTypeLabel: storeTypeLabel,
		IsOpen:         isOpen,
		Rating:         rating,
		ReviewCount:    reviewCount,
		Hashtags:       hashtags,
		CategoryIDs:    categoryIDs,
		CategoryNames:  categoryNames,
		Geo:            geo,
		Latitude:       latPtr,
		Longitude:      lngPtr,
		Lat:            latPtr,
		Lng:            lngPtr,
		CreatedAt:      p.CreatedAt.Unix(),
		IsPackage:      p.IsPackage,
		PromotionType:  promotionType,
		PromotionID:    promotionID,
		HasPromo:       hasPromo,
	}
}

// BuildBundlingDocument creates a MeiliProductDocument from an active bundling promotion
func BuildBundlingDocument(db *gorm.DB, prm *models.Promotion, b *models.Business) MeiliProductDocument {
	if b == nil && db != nil {
		var biz models.Business
		if err := db.Preload("Hours").Preload("Categories").Where("business_id = ?", prm.BusinessID).First(&biz).Error; err == nil {
			b = &biz
		}
	}

	name := "Paket Promosi"
	if prm.PackageName != nil && strings.TrimSpace(*prm.PackageName) != "" {
		name = strings.TrimSpace(*prm.PackageName)
	} else if strings.TrimSpace(prm.Caption) != "" {
		name = strings.TrimSpace(prm.Caption)
	}

	price := 0.0
	if prm.OriginalTotalPrice != nil && *prm.OriginalTotalPrice > 0 {
		price = *prm.OriginalTotalPrice
	} else if prm.PackagePrice != nil {
		price = *prm.PackagePrice
	}

	finalPrice := price
	if prm.PackagePrice != nil && *prm.PackagePrice > 0 {
		finalPrice = *prm.PackagePrice
	}

	discountPct := 0.0
	if prm.SavingsPercentage != nil && *prm.SavingsPercentage > 0 {
		discountPct = *prm.SavingsPercentage
	} else if price > finalPrice && price > 0 {
		discountPct = (price - finalPrice) / price * 100
	}

	discountAmount := 0.0
	if prm.SavingsAmount != nil && *prm.SavingsAmount > 0 {
		discountAmount = *prm.SavingsAmount
	} else if price > finalPrice {
		discountAmount = price - finalPrice
	}

	var preview []string
	if prm.PackageImage != nil && strings.TrimSpace(*prm.PackageImage) != "" {
		preview = []string{strings.TrimSpace(*prm.PackageImage)}
	} else {
		if len(prm.BundleItems) == 0 && db != nil {
			_ = db.Preload("Product").Where("promotion_id = ?", prm.PromotionID).Find(&prm.BundleItems).Error
		}
		for _, item := range prm.BundleItems {
			if item.Product != nil && len(item.Product.ProductPreview) > 0 {
				preview = append(preview, item.Product.ProductPreview...)
			}
		}
	}
	if preview == nil {
		preview = []string{}
	}

	bizName := ""
	storeType := "regular"
	storeTypeLabel := "Regular"
	isOpen := false
	rating := 0.0
	var reviewCount int64
	var categoryIDs []int64
	var categoryNames []string
	var geo GeoLocation
	var latPtr, lngPtr *float64

	if b != nil {
		if b.BusinessName != nil {
			bizName = *b.BusinessName
		}
		if b.StoreType != nil {
			storeType = models.NormalizeStoreType(*b.StoreType)
			storeTypeLabel = models.GetStoreTypeLabel(storeType)
		}
		isOpen = utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)
		rating = b.AverageRating
		reviewCount = b.ReviewCount

		for _, c := range b.Categories {
			categoryIDs = append(categoryIDs, c.CategoryID)
			if c.Name != "" {
				categoryNames = append(categoryNames, c.Name)
			}
		}
		if b.CustomCategory != nil && strings.TrimSpace(*b.CustomCategory) != "" {
			categoryNames = append(categoryNames, strings.TrimSpace(*b.CustomCategory))
		}

		if b.Latitude != nil && b.Longitude != nil {
			geo = GeoLocation{
				Lat: *b.Latitude,
				Lng: *b.Longitude,
			}
			latPtr = b.Latitude
			lngPtr = b.Longitude
		}
	}

	hashtags := []string{"promo", "bundling", "paket"}
	if b != nil && len(b.Hashtags) > 0 {
		for _, h := range b.Hashtags {
			if strings.TrimSpace(h) != "" {
				hashtags = append(hashtags, strings.TrimSpace(h))
			}
		}
	}

	return MeiliProductDocument{
		ID:             prm.PromotionID,
		ProductID:      prm.PromotionID,
		ProductName:    name,
		PackageName:    name,
		ProductPreview: preview,
		Price:          price,
		FinalPrice:     finalPrice,
		IsDiscounted:   true,
		DiscountPct:    discountPct,
		DiscountAmount: discountAmount,
		StockType:      "in_stock",
		BusinessID:     prm.BusinessID,
		BusinessName:   bizName,
		StoreType:      storeType,
		StoreTypeLabel: storeTypeLabel,
		IsOpen:         isOpen,
		Rating:         rating,
		ReviewCount:    reviewCount,
		Hashtags:       hashtags,
		CategoryIDs:    categoryIDs,
		CategoryNames:  categoryNames,
		Geo:            geo,
		Latitude:       latPtr,
		Longitude:      lngPtr,
		Lat:            latPtr,
		Lng:            lngPtr,
		CreatedAt:      prm.CreatedAt.Unix(),
		IsPackage:      true,
		PromotionType:  "bundling",
		PromotionID:    prm.PromotionID,
		HasPromo:       true,
	}
}

// SyncBusiness synchronizes a single business and its products to Meilisearch
func SyncBusiness(db *gorm.DB, businessID string) error {
	if Client == nil || businessID == "" {
		return nil
	}

	var b models.Business
	err := db.Preload("Hours").Preload("Categories").Preload("Tags").Where("business_id = ?", businessID).First(&b).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = DeleteBusinessDoc(businessID)
			_ = DeleteBusinessProductsDoc(db, businessID)
			return nil
		}
		log.Printf("⚠️ Gagal query business %s untuk Meilisearch: %v", businessID, err)
		return err
	}

	// If business is inactive or soft-deleted, remove from index
	if !b.IsActive || b.Status == "deleted" {
		_ = DeleteBusinessDoc(businessID)
		_ = DeleteBusinessProductsDoc(db, businessID)
		return nil
	}

	doc := BuildBusinessDocument(db, &b)
	bizIndex := Client.Index("businesses")
	_, err = bizIndex.AddDocuments([]MeiliBusinessDocument{doc}, nil)
	if err != nil {
		log.Printf("⚠️ Gagal sync business %s (%s) ke Meilisearch: %v", businessID, doc.Name, err)
	}

	// Sync products of this business
	_ = SyncBusinessProducts(db, businessID)

	log.Printf("🔍 Berhasil sync business %s (%s) ke Meilisearch", businessID, doc.Name)
	return err
}

// SyncProduct synchronizes a single product to Meilisearch
func SyncProduct(db *gorm.DB, productID string) error {
	if Client == nil || productID == "" {
		return nil
	}

	if strings.HasPrefix(productID, "PRM-") {
		return SyncPromotion(db, productID)
	}

	var p models.Product
	if err := db.Where("product_id = ?", productID).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return DeleteProductDoc(productID)
		}
		return err
	}

	if !p.IsActive {
		return DeleteProductDoc(productID)
	}

	var b models.Business
	if err := db.Preload("Hours").Preload("Categories").Where("business_id = ?", p.BusinessID).First(&b).Error; err != nil {
		return err
	}
	if !b.IsActive || b.Status == "deleted" {
		return DeleteProductDoc(productID)
	}

	doc := BuildProductDocument(db, &p, &b)
	prodIndex := Client.Index("products")
	primaryKey := "id"
	_, err := prodIndex.AddDocuments([]MeiliProductDocument{doc}, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
	return err
}

// SyncBusinessProducts synchronizes all products of a business to Meilisearch
func SyncBusinessProducts(db *gorm.DB, businessID string) error {
	if Client == nil || businessID == "" {
		return nil
	}

	var b models.Business
	if err := db.Preload("Hours").Preload("Categories").Where("business_id = ?", businessID).First(&b).Error; err != nil {
		return err
	}

	var products []models.Product
	if err := db.Where("business_id = ?", businessID).Find(&products).Error; err != nil {
		return err
	}

	prodIndex := Client.Index("products")
	if !b.IsActive || b.Status == "deleted" {
		ids := make([]string, len(products))
		for i, p := range products {
			ids[i] = p.ProductID
		}
		if len(ids) > 0 {
			_, _ = prodIndex.DeleteDocuments(ids, nil)
		}
		return nil
	}

	docs := make([]MeiliProductDocument, 0, len(products))
	deletedIDs := make([]string, 0)
	for i := range products {
		if products[i].IsActive {
			docs = append(docs, BuildProductDocument(db, &products[i], &b))
		} else {
			deletedIDs = append(deletedIDs, products[i].ProductID)
		}
	}

	// Sync active bundling promotions of this business
	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)
	var bundlingPromos []models.Promotion
	if err := db.Preload("BundleItems.Product").Where("business_id = ? AND promotion_type = 'bundling' AND status = 'active' AND start_at <= ? AND end_at >= ?", businessID, now, now).Find(&bundlingPromos).Error; err == nil {
		for i := range bundlingPromos {
			docs = append(docs, BuildBundlingDocument(db, &bundlingPromos[i], &b))
		}
	}

	// Delete stopped/expired bundling promotions of this business
	var inactiveBundlingIDs []string
	if err := db.Model(&models.Promotion{}).Where("business_id = ? AND promotion_type = 'bundling' AND (status != 'active' OR end_at < ?)", businessID, now).Pluck("promotion_id", &inactiveBundlingIDs).Error; err == nil && len(inactiveBundlingIDs) > 0 {
		deletedIDs = append(deletedIDs, inactiveBundlingIDs...)
	}

	primaryKey := "id"
	if len(docs) > 0 {
		_, _ = prodIndex.AddDocuments(docs, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
	}
	if len(deletedIDs) > 0 {
		_, _ = prodIndex.DeleteDocuments(deletedIDs, nil)
	}
	return nil
}

// DeleteProductDoc removes a product document from Meilisearch
func DeleteProductDoc(productID string) error {
	if Client == nil || productID == "" {
		return nil
	}
	prodIndex := Client.Index("products")
	_, err := prodIndex.DeleteDocument(productID, nil)
	return err
}

// DeleteBusinessProductsDoc removes all product documents of a business from Meilisearch
func DeleteBusinessProductsDoc(db *gorm.DB, businessID string) error {
	if Client == nil || businessID == "" {
		return nil
	}
	var productIDs []string
	if db != nil {
		db.Model(&models.Product{}).Where("business_id = ?", businessID).Pluck("product_id", &productIDs)
	}
	if len(productIDs) > 0 {
		prodIndex := Client.Index("products")
		_, err := prodIndex.DeleteDocuments(productIDs, nil)
		return err
	}
	return nil
}

// DeleteBusinessDoc removes a business document from Meilisearch
func DeleteBusinessDoc(businessID string) error {
	if Client == nil || businessID == "" {
		return nil
	}
	bizIndex := Client.Index("businesses")
	_, err := bizIndex.DeleteDocument(businessID, nil)
	return err
}

// SyncAllBusinesses synchronizes all active businesses and products from PostgreSQL to Meilisearch
func SyncAllBusinesses(db *gorm.DB) (int, error) {
	if Client == nil {
		return 0, nil
	}

	var businesses []models.Business
	err := db.Preload("Hours").Preload("Categories").Preload("Tags").Where("is_active = ? AND status != ?", true, "deleted").Find(&businesses).Error
	if err != nil {
		log.Printf("⚠️ Gagal query semua businesses untuk Meilisearch: %v", err)
		return 0, err
	}

	if len(businesses) == 0 {
		log.Println("ℹ️ Tidak ada data bisnis untuk disinkronkan ke Meilisearch.")
		return 0, nil
	}

	bizDocs := make([]MeiliBusinessDocument, 0, len(businesses))
	for i := range businesses {
		bizDocs = append(bizDocs, BuildBusinessDocument(db, &businesses[i]))
	}

	bizIndex := Client.Index("businesses")
	_, err = bizIndex.AddDocuments(bizDocs, nil)
	if err != nil {
		log.Printf("⚠️ Gagal upload batch businesses ke Meilisearch: %v", err)
	}

	// Also sync all active products in batch
	var products []models.Product
	db.Where("is_active = ?", true).Find(&products)
	bizMap := make(map[string]*models.Business)
	for i := range businesses {
		bizMap[businesses[i].BusinessID] = &businesses[i]
	}

	prodDocs := make([]MeiliProductDocument, 0, len(products))
	for i := range products {
		if b, ok := bizMap[products[i].BusinessID]; ok {
			prodDocs = append(prodDocs, BuildProductDocument(db, &products[i], b))
		}
	}

	// Also sync active bundling promotions in batch
	now := time.Now().In(utils.GetWIBLocation())
	var bundlingPromos []models.Promotion
	if err := db.Preload("BundleItems.Product").Where("promotion_type = 'bundling' AND status = 'active' AND start_at <= ? AND end_at >= ?", now, now).Find(&bundlingPromos).Error; err == nil {
		for i := range bundlingPromos {
			if b, ok := bizMap[bundlingPromos[i].BusinessID]; ok {
				prodDocs = append(prodDocs, BuildBundlingDocument(db, &bundlingPromos[i], b))
			}
		}
	}

	if len(prodDocs) > 0 {
		prodIndex := Client.Index("products")
		primaryKey := "id"
		_, err = prodIndex.AddDocuments(prodDocs, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
		if err != nil {
			log.Printf("⚠️ Gagal upload batch products ke Meilisearch: %v", err)
		}
	}

	log.Printf("✅ Selesai sync! %d bisnis & %d produk telah disinkronkan ke Meilisearch", len(bizDocs), len(prodDocs))
	return len(bizDocs), nil
}

// SyncAllProducts synchronizes all active products from PostgreSQL to Meilisearch
func SyncAllProducts(db *gorm.DB) (int, error) {
	if Client == nil || db == nil {
		return 0, nil
	}

	var businesses []models.Business
	err := db.Preload("Hours").Preload("Categories").Where("is_active = ? AND status != ?", true, "deleted").Find(&businesses).Error
	if err != nil {
		log.Printf("⚠️ Gagal query businesses untuk SyncAllProducts: %v", err)
		return 0, err
	}

	bizMap := make(map[string]*models.Business, len(businesses))
	for i := range businesses {
		bizMap[businesses[i].BusinessID] = &businesses[i]
	}

	var products []models.Product
	if err := db.Where("is_active = ?", true).Find(&products).Error; err != nil {
		log.Printf("⚠️ Gagal query products untuk SyncAllProducts: %v", err)
		return 0, err
	}

	prodDocs := make([]MeiliProductDocument, 0, len(products))
	for i := range products {
		if b, ok := bizMap[products[i].BusinessID]; ok {
			prodDocs = append(prodDocs, BuildProductDocument(db, &products[i], b))
		}
	}

	// Also sync active bundling promotions
	now := time.Now().In(utils.GetWIBLocation())
	var bundlingPromos []models.Promotion
	if err := db.Preload("BundleItems.Product").Where("promotion_type = 'bundling' AND status = 'active' AND start_at <= ? AND end_at >= ?", now, now).Find(&bundlingPromos).Error; err == nil {
		for i := range bundlingPromos {
			if b, ok := bizMap[bundlingPromos[i].BusinessID]; ok {
				prodDocs = append(prodDocs, BuildBundlingDocument(db, &bundlingPromos[i], b))
			}
		}
	}

	if len(prodDocs) > 0 {
		prodIndex := Client.Index("products")
		primaryKey := "id"
		_, err = prodIndex.AddDocuments(prodDocs, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
		if err != nil {
			log.Printf("⚠️ Gagal sync all products ke Meilisearch: %v", err)
			return 0, err
		}
	}

	log.Printf("✅ Selesai sync! %d produk telah disinkronkan ke Meilisearch 'products'", len(prodDocs))
	return len(prodDocs), nil
}

// UpdateBusinessStatus sends an immediate partial update to Meilisearch for is_open, is_manually_closed, and close_time
func UpdateBusinessStatus(db *gorm.DB, businessID string) error {
	if Client == nil || db == nil || businessID == "" {
		return nil
	}

	var b models.Business
	if err := db.Preload("Hours").Where("business_id = ?", businessID).First(&b).Error; err != nil {
		return err
	}

	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)
	currentDayInt := int(now.Weekday())

	isOpen := utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)
	closeTime := ""
	for _, h := range b.Hours {
		if h.DayOfWeek == currentDayInt && h.CloseTime != nil {
			closeTime = *h.CloseTime
			break
		}
	}

	primaryKey := "id"
	// 1. Partial update index "businesses"
	bizIndex := Client.Index("businesses")
	updateDoc := map[string]interface{}{
		"id":                 b.BusinessID,
		"is_open":            isOpen,
		"is_manually_closed": b.IsManuallyClosed,
		"close_time":         closeTime,
	}
	_, err := bizIndex.UpdateDocuments([]map[string]interface{}{updateDoc}, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
	if err != nil {
		log.Printf("⚠️ Gagal update status bisnis %s di Meilisearch: %v", businessID, err)
	}

	// 2. Partial update seluruh produk milik toko ini di index "products"
	var productIDs []string
	if errProd := db.Model(&models.Product{}).Where("business_id = ? AND is_active = ?", businessID, true).Pluck("product_id", &productIDs).Error; errProd == nil && len(productIDs) > 0 {
		prodUpdates := make([]map[string]interface{}, 0, len(productIDs))
		for _, pid := range productIDs {
			prodUpdates = append(prodUpdates, map[string]interface{}{
				"id":      pid,
				"is_open": isOpen,
			})
		}
		prodIndex := Client.Index("products")
		_, _ = prodIndex.UpdateDocuments(prodUpdates, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
	}

	log.Printf("⏰ [RealTimeStatus] Status bisnis %s diperbarui di Meilisearch (is_open=%v, is_manually_closed=%v, close_time=%s)", businessID, isOpen, b.IsManuallyClosed, closeTime)
	return err
}

type BusinessStatusBatchUpdate struct {
	ID               string `json:"id"`
	IsOpen           bool   `json:"is_open"`
	IsManuallyClosed bool   `json:"is_manually_closed"`
	CloseTime        string `json:"close_time"`
}

// BatchUpdateBusinessStatuses updates multiple businesses' status in Meilisearch
func BatchUpdateBusinessStatuses(updates []BusinessStatusBatchUpdate) error {
	if Client == nil || len(updates) == 0 {
		return nil
	}
	primaryKey := "id"
	bizIndex := Client.Index("businesses")
	_, err := bizIndex.UpdateDocuments(updates, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
	return err
}

// SyncPromotion synchronizes a promotion document (bundling) or affected product discounts to Meilisearch
func SyncPromotion(db *gorm.DB, promotionID string) error {
	if Client == nil || db == nil || promotionID == "" {
		return nil
	}

	var prm models.Promotion
	if err := db.Preload("BundleItems.Product").Preload("Products").Where("promotion_id = ?", promotionID).First(&prm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return DeleteProductDoc(promotionID)
		}
		return err
	}

	now := time.Now().In(utils.GetWIBLocation())
	isActive := prm.Status == models.PromotionStatusActive && !prm.StartAt.After(now) && !prm.EndAt.Before(now)

	if prm.PromotionType == models.PromotionTypeBundling {
		if !isActive {
			return DeleteProductDoc(promotionID)
		}
		var b models.Business
		_ = db.Preload("Hours").Preload("Categories").Where("business_id = ?", prm.BusinessID).First(&b).Error
		doc := BuildBundlingDocument(db, &prm, &b)
		prodIndex := Client.Index("products")
		primaryKey := "id"
		_, err := prodIndex.AddDocuments([]MeiliProductDocument{doc}, &meilisearch.DocumentOptions{PrimaryKey: &primaryKey})
		return err
	}

	// For per_product promotion: re-sync all affected products
	for _, pp := range prm.Products {
		_ = SyncProduct(db, pp.ProductID)
	}
	return nil
}


