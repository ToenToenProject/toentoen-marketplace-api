package product

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductDetailResponse struct {
	ProductID          string   `json:"product_id"`
	ProductName        string   `json:"product_name"`
	ProductOrigin      string   `json:"product_origin"`
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
	BusinessID         string   `json:"business_id"`
	BusinessName       string   `json:"business_name"`
	BusinessAvatar     *string  `json:"business_avatar,omitempty"`
	StockStatus        string   `json:"stock_status"`

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

	// Calculate prices
	finalPrice := prod.Price
	var originalPrice *float64
	var discountAmount float64
	var discountPct float64
	isDiscounted := false

	if prod.Discount > 0 {
		cut, pct := utils.CalculateProductDiscount(prod.Price, prod.DiscountType, prod.Discount, prod.MaxDiscount)
		if cut > 0 {
			isDiscounted = true
			finalPrice = prod.Price - cut
			original := prod.Price
			originalPrice = &original
			discountAmount = cut
			discountPct = pct
		}
	}

	stockStatus := "In Stock"
	if prod.StockType == "out_of_stock" {
		stockStatus = "Out of Stock"
	}

	bName := ""
	if bus.BusinessName != nil {
		bName = *bus.BusinessName
	}
	
	res := ProductDetailResponse{
		ProductID:          prod.ProductID,
		ProductName:        prod.ProductName,
		ProductOrigin:      prod.ProductOrigin,
		ProductDescription: prod.ProductDescription,
		ProductPreview:     prod.ProductPreview,
		Rating:             prod.AverageRating,
		ReviewCount:        prod.ReviewCount,
		Price:              finalPrice,
		OriginalPrice:      originalPrice,
		IsDiscounted:       isDiscounted,
		DiscountType:       prod.DiscountType,
		DiscountAmount:     discountAmount,
		DiscountPct:        discountPct,
		Tags:               prod.Tags,
		BusinessID:         bus.BusinessID,
		BusinessName:       bName,
		BusinessAvatar:     bus.LogoURL,
		StockStatus:        stockStatus,
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

		if sim.Discount > 0 {
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
