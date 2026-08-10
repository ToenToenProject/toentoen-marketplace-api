package product

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
)

type ProductDetailResponse struct {
	ProductID          string   `json:"product_id"`
	ProductName        string   `json:"product_name"`
	ProductOrigin      string   `json:"product_origin"`
	ImportedFrom       string   `json:"imported_from"`
	ProductImages      []string `json:"product_images"`
	Rating             float64  `json:"rating"`
	ReviewCount        int      `json:"review_count"`
	Price              float64  `json:"price"`
	OriginalPrice      *float64 `json:"original_price,omitempty"` // Harga coret (hanya jika diskon)
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
		return nil, errors.New("product not found")
	}

	// Calculate prices
	finalPrice := prod.Price
	var originalPrice *float64
	if prod.Discount > 0 {
		finalPrice = prod.Price - (prod.Price * (prod.Discount / 100))
		original := prod.Price
		originalPrice = &original
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
		ImportedFrom:       prod.ImportedFrom,
		ProductImages:      prod.ProductImages,
		Rating:             prod.AverageRating,
		ReviewCount:        prod.ReviewCount,
		Price:              finalPrice,
		OriginalPrice:      originalPrice,
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
		if sim.Discount > 0 {
			simOriginal = sim.Price
			simPrice = sim.Price - (sim.Price * (sim.Discount / 100))
		}
		
		img := ""
		if len(sim.ProductImages) > 0 {
			img = sim.ProductImages[0]
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
			IsDiscounted:     sim.Discount > 0,
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
