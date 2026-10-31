package product

import (
	"errors"
	"math"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductDetailResponse struct {
	ProductID          string   `json:"product_id"`
	ProductName        string   `json:"product_name"`
	ProductOrigin      string   `json:"product_origin"`
	OriginCountry      *string  `json:"origin_country,omitempty"`
	ProductDescription string   `json:"product_description"`
	ProductPreview     []string `json:"product_preview"`
	Rating             float64  `json:"rating"`
	ReviewCount        int      `json:"review_count"`
	Price              float64  `json:"price"`
	OriginalPrice      *float64 `json:"original_price,omitempty"` // Harga coret (hanya jika diskon)
	IsDiscounted       bool     `json:"is_discounted"`
	DiscountType       string   `json:"discount_type"`
	DiscountAmount     float64  `json:"discount_amount"` // Potongan harga (Rupiah)
	DiscountPct        float64  `json:"discount_pct"`    // Besaran diskon (%)
	Tags               []string `json:"tags"`
	Hashtags           []string `json:"hashtags"`
	BusinessID         string   `json:"business_id"`
	BusinessName       string   `json:"business_name"`
	BusinessAvatar     *string  `json:"business_avatar,omitempty"`
	StockStatus        string   `json:"stock_status"`

	IsPromoted      bool                               `json:"is_promoted"`
	PromotionID     *string                            `json:"promotion_id,omitempty"`
	PromotionName   *string                            `json:"promotion_name,omitempty"`
	DiscountSource  string                             `json:"discount_source,omitempty"`
	ActivePromotion *models.BusinessActivePromotionItem `json:"active_promotion,omitempty"`

	// Fitur Paket Produk (Standalone Package)
	IsPackage    bool                               `json:"is_package"`
	PackageItems []models.PackageItemDetailResponse `json:"package_items,omitempty"`

	PremiumSimilarProducts []models.ProductListItem `json:"premium_similar_products"`
	SimilarProducts        []models.ProductListItem `json:"similar_products"`
}

type ProductDetailService interface {
	GetProductDetail(productID string) (*ProductDetailResponse, error)
}

type productDetailService struct {
	repo ProductDetailRepository
}

func NewProductDetailService(repo ProductDetailRepository) ProductDetailService {
	return &productDetailService{repo}
}

func (s *productDetailService) GetProductDetail(productID string) (*ProductDetailResponse, error) {
	prod, bus, err := s.repo.GetProductWithBusiness(productID)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}

	// Calculate prices & promotions
	finalPrice := prod.Price
	var originalPrice *float64
	var discountAmount float64
	var discountPct float64
	isDiscounted := false
	discountType := prod.DiscountType
	isPromoted := false
	var promoID *string
	var promoName *string
	discountSource := ""
	var activePromoItem *models.BusinessActivePromotionItem

	promo, promoProd, _ := s.repo.GetActivePromotionForProduct(bus.BusinessID, prod.ProductID)
	if promo != nil {
		isPromoted = true
		promoID = &promo.PromotionID
		title := "Promo Spesial Toko"
		if promo.PackageName != nil && strings.TrimSpace(*promo.PackageName) != "" {
			title = strings.TrimSpace(*promo.PackageName)
		} else if strings.TrimSpace(promo.Caption) != "" {
			title = strings.TrimSpace(promo.Caption)
		} else if promo.Post != nil && strings.TrimSpace(promo.Post.Content) != "" {
			title = strings.TrimSpace(promo.Post.Content)
		}
		promoName = &title

		var banner *string
		if promo.PackageImage != nil && *promo.PackageImage != "" {
			banner = promo.PackageImage
		} else if promo.Post != nil && len(promo.Post.Images) > 0 {
			banner = &promo.Post.Images[0]
		} else if len(prod.ProductPreview) > 0 {
			banner = &prod.ProductPreview[0]
		}

		activePromoItem = &models.BusinessActivePromotionItem{
			PromotionID:   promo.PromotionID,
			Title:         title,
			Caption:       promo.Caption,
			BannerURL:     banner,
			PromotionType: promo.PromotionType,
			DiscountPct:   promo.SavingsPercentage,
			StartAt:       promo.StartAt,
			EndAt:         promo.EndAt,
		}

		if promoProd != nil {
			discountSource = promoProd.DiscountSource
			if discountSource == "" {
				discountSource = "promotion"
			}
			if promoProd.DiscountType != "" && promoProd.Discount > 0 {
				origPrice := prod.Price
				cut, pct := utils.CalculateProductDiscount(origPrice, promoProd.DiscountType, promoProd.Discount, promoProd.MaxDiscount)
				if cut > 0 {
					isDiscounted = true
					finalPrice = origPrice - cut
					originalPrice = &origPrice
					discountAmount = cut
					discountPct = pct
					discountType = promoProd.DiscountType
				}
			}
		} else {
			discountSource = "promotion"
		}
	}

	if !isDiscounted && utils.IsDiscountActive(prod.Discount, prod.DiscountStartAt, prod.DiscountEndAt, time.Now()) {
		cut, pct := utils.CalculateProductDiscount(prod.Price, prod.DiscountType, prod.Discount, prod.MaxDiscount)
		if cut > 0 {
			isDiscounted = true
			finalPrice = prod.Price - cut
			original := prod.Price
			originalPrice = &original
			discountAmount = cut
			discountPct = pct
			discountType = prod.DiscountType
			if discountSource == "" {
				discountSource = "product"
			}
		}
	}

	stockStatus := "In Stock"
	if prod.StockType == "out_of_stock" {
		stockStatus = "Out of Stock"
	}

	var packageItems []models.PackageItemDetailResponse
	if prod.IsPackage {
		if prod.OriginalPackagePrice != nil && *prod.OriginalPackagePrice > prod.Price {
			isDiscounted = true
			orig := *prod.OriginalPackagePrice
			originalPrice = &orig
			discountAmount = *prod.OriginalPackagePrice - prod.Price
			if prod.PackageSavingsPercentage != nil && *prod.PackageSavingsPercentage > 0 {
				discountPct = *prod.PackageSavingsPercentage
			} else if *prod.OriginalPackagePrice > 0 {
				discountPct = math.Round((discountAmount / *prod.OriginalPackagePrice) * 10000) / 100
			}
			discountType = "nominal"
			if discountSource == "" {
				discountSource = "package"
			}
		}

		for _, pi := range prod.PackageItems {
			name := ""
			imgURL := ""
			st := "in_stock"
			act := true
			if pi.ItemProduct != nil {
				name = pi.ItemProduct.ProductName
				st = pi.ItemProduct.StockType
				act = pi.ItemProduct.IsActive
				if len(pi.ItemProduct.ProductPreview) > 0 {
					imgURL = pi.ItemProduct.ProductPreview[0]
				}
				if st == "out_of_stock" {
					stockStatus = "Out of Stock"
				}
			}

			packageItems = append(packageItems, models.PackageItemDetailResponse{
				ProductID:              pi.ItemProductID,
				ProductName:            name,
				ImageURL:               imgURL,
				Quantity:               pi.Quantity,
				NormalPrice:            pi.ItemNormalPrice,
				EffectivePrice:         pi.ItemEffectivePrice,
				SubtotalEffectivePrice: pi.ItemEffectivePrice * float64(pi.Quantity),
				StockType:              st,
				IsActive:               act,
			})
		}
	}

	bName := ""
	if bus.BusinessName != nil {
		bName = *bus.BusinessName
	}
	
	res := ProductDetailResponse{
		ProductID:          prod.ProductID,
		ProductName:        prod.ProductName,
		ProductOrigin:      prod.ProductOrigin,
		OriginCountry:      prod.OriginCountry,
		ProductDescription: prod.ProductDescription,
		ProductPreview:     prod.ProductPreview,
		Rating:             prod.AverageRating,
		ReviewCount:        prod.ReviewCount,
		Price:              finalPrice,
		OriginalPrice:      originalPrice,
		IsDiscounted:       isDiscounted,
		DiscountType:       discountType,
		DiscountAmount:     discountAmount,
		DiscountPct:        discountPct,
		Tags:               prod.Tags,
		Hashtags:           prod.Tags,
		BusinessID:         bus.BusinessID,
		BusinessName:       bName,
		BusinessAvatar:     bus.LogoURL,
		StockStatus:        stockStatus,
		IsPromoted:         isPromoted,
		PromotionID:        promoID,
		PromotionName:      promoName,
		DiscountSource:     discountSource,
		ActivePromotion:    activePromoItem,
		IsPackage:          prod.IsPackage,
		PackageItems:       packageItems,
	}

	// Fetch similar products
	similars, _ := s.repo.GetSimilarProducts(prod.Tags, 10, prod.ProductID)
	
	// Convert to ProductListItem
	var similarItems []models.ProductListItem
	for _, sim := range similars {
		simPrice := sim.Price
		var simOriginal float64
		var discountAmount float64
		var discountPct float64
		isDiscounted := false

		if utils.IsDiscountActive(sim.Discount, sim.DiscountStartAt, sim.DiscountEndAt, time.Now()) {
			cut, pct := utils.CalculateProductDiscount(sim.Price, sim.DiscountType, sim.Discount, sim.MaxDiscount)
			if cut > 0 {
				isDiscounted = true
				simOriginal = sim.Price
				simPrice = sim.Price - cut
				discountAmount = cut
				discountPct = pct
			}
		}
		
		img := ""
		if len(sim.ProductPreview) > 0 {
			img = sim.ProductPreview[0]
		}
		
		st := "In Stock"
		if sim.StockType == "out_of_stock" {
			st = "Out of Stock"
		}
		
		similarItems = append(similarItems, models.ProductListItem{
			ProductID:        sim.ProductID,
			ProductName:      sim.ProductName,
			ImageURL:         img,
			HasVariants:      sim.HasVariants,
			OriginalMinPrice: simOriginal, // Simplified
			FinalMinPrice:    simPrice,
			IsDiscounted:     isDiscounted,
			DiscountType:     sim.DiscountType,
			Discount:         sim.Discount,
			DiscountAmount:   discountAmount,
			DiscountPct:      discountPct,
			MaxDiscount:      sim.MaxDiscount,
			StockType:        st,
			Rating:           sim.AverageRating,
			LikeCount:        sim.LikeCount,
		})
	}

	// Just duplicate for now as requested
	res.PremiumSimilarProducts = similarItems
	res.SimilarProducts = similarItems

	return &res, nil
}
