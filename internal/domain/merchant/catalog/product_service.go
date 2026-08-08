package catalog

import (
	"errors"
	"sort"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductService interface {
	CreateProduct(requesterID, role, businessID string, input models.CreateProductInput) error
	// Fitur Varian
	AddVariant(requesterID, role, businessID, productID string, input models.CreateVariantInput) error
	UpdateVariant(requesterID, role, businessID, productID, variantID string, input models.UpdateVariantInput) error
	RemoveVariant(requesterID, role, businessID, productID, variantID string) error

	UpdateProduct(requesterID, role, businessID, productID string, input models.UpdateProductInput) error
	DeleteProduct(requesterID, role, businessID, productID string) error
	GetProductDetail(requesterID, role, businessID, productID string) (*models.ProductDetailResponse, error)

	GetCatalog(requesterID, role, businessID string) ([]models.CatalogGroupResponse, error)

	// FUNGSI INI DIKEMBALIKAN: Menggunakan Cursor untuk Infinite Scroll & Multi-Filter
	GetAllProducts(requesterID, role, businessID string, search, productGroupID, stockType string, isActive *bool, sort string, cursor string, limit int) (*models.GetProductsResponse, error)

	BulkUpdateStatus(requesterID, role, businessID string, productIDs []string, isActive bool) error
	BulkDelete(requesterID, role, businessID string, productIDs []string) error
}

type productService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{repo}
}

// 1. CREATE PRODUCT
func (s *productService) CreateProduct(requesterID, role, businessID string, input models.CreateProductInput) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: you don't have access to manage products for this business")
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	product := models.Product{
		ProductID:          utils.GenerateUUID("PRD"),
		BusinessID:         businessID,
		ProductGroupID:     input.ProductGroupID,
		ProductImages:      input.ProductImages,
		ProductName:        input.ProductName,
		ProductDescription: input.ProductDescription,
		Price:              input.Price,
		StockType:          input.StockType,
		StockQuantity:      input.StockQuantity,
		LowStockThreshold:  input.LowStockThreshold,
		Tags:               input.Tags,
		IsActive:           isActive,
	}

	if err := s.repo.CreateProduct(&product); err != nil {
		return errors.New("failed to create product")
	}
	return nil
}

// ==========================================
// IMPLEMENTASI PRODUCT VARIANTS
// ==========================================

func (s *productService) AddVariant(requesterID, role, businessID, productID string, input models.CreateVariantInput) error {
	if _, err := s.GetProductDetail(requesterID, role, businessID, productID); err != nil {
		return err
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	variant := models.ProductVariant{
		VariantID:       utils.GenerateUUID("VAR"),
		ProductID:       productID,
		SKU:             input.SKU,
		VariantName:     input.VariantName,
		AdditionalPrice: input.AdditionalPrice,
		StockQuantity:   input.StockQuantity,
		IsActive:        isActive,
	}

	if err := s.repo.CreateVariant(&variant); err != nil {
		return errors.New("failed to add variant")
	}
	return nil
}

func (s *productService) UpdateVariant(requesterID, role, businessID, productID, variantID string, input models.UpdateVariantInput) error {
	if _, err := s.GetProductDetail(requesterID, role, businessID, productID); err != nil {
		return err
	}

	variant, err := s.repo.GetVariantByID(variantID, productID)
	if err != nil {
		return errors.New("variant not found")
	}

	if input.SKU != nil {
		variant.SKU = *input.SKU
	}
	if input.VariantName != nil {
		variant.VariantName = *input.VariantName
	}
	if input.AdditionalPrice != nil {
		variant.AdditionalPrice = *input.AdditionalPrice
	}
	if input.StockQuantity != nil {
		variant.StockQuantity = *input.StockQuantity
	}
	if input.IsActive != nil {
		variant.IsActive = *input.IsActive
	}

	if err := s.repo.UpdateVariant(variant); err != nil {
		return errors.New("failed to update variant")
	}
	return nil
}

func (s *productService) RemoveVariant(requesterID, role, businessID, productID, variantID string) error {
	if _, err := s.GetProductDetail(requesterID, role, businessID, productID); err != nil {
		return err
	}

	if err := s.repo.DeleteVariant(variantID, productID); err != nil {
		return errors.New("failed to delete variant")
	}
	return nil
}

// 2. UPDATE PRODUCT
func (s *productService) UpdateProduct(requesterID, role, businessID, productID string, input models.UpdateProductInput) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: you don't have access to manage products for this business")
	}

	product, err := s.repo.GetProductByID(productID, businessID)
	if err != nil {
		return errors.New("product not found in this business")
	}

	if input.ProductGroupID != nil {
		product.ProductGroupID = input.ProductGroupID
	}
	if input.ProductImages != nil {
		product.ProductImages = *input.ProductImages
	}
	if input.ProductName != nil {
		product.ProductName = *input.ProductName
	}
	if input.ProductDescription != nil {
		product.ProductDescription = *input.ProductDescription
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.Discount != nil {
		product.Discount = *input.Discount

		if *input.Discount == 0 {
			product.DiscountStartAt = nil
			product.DiscountEndAt = nil
		}
	}
	if input.Discount != nil {
		product.Discount = *input.Discount
	}
	if input.DiscountStartAt != nil {
		product.DiscountStartAt = input.DiscountStartAt
	}
	if input.DiscountEndAt != nil {
		product.DiscountEndAt = input.DiscountEndAt
	}
	if input.StockType != nil {
		product.StockType = *input.StockType
	}
	if input.StockQuantity != nil {
		product.StockQuantity = *input.StockQuantity
	}
	if input.IsActive != nil {
		product.IsActive = *input.IsActive
	}

	if err := s.repo.UpdateProduct(product); err != nil {
		return errors.New("failed to update product")
	}
	return nil
}

// 3. DELETE PRODUCT
func (s *productService) DeleteProduct(requesterID, role, businessID, productID string) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: you don't have access to manage products for this business")
	}

	if err := s.repo.DeleteProduct(productID, businessID); err != nil {
		return errors.New("failed to delete product")
	}
	return nil
}

// 4. GET DETAIL PRODUCT (Single)
// Ganti fungsi ini di internal/services/product_service.go
func (s *productService) GetProductDetail(requesterID, role, businessID, productID string) (*models.ProductDetailResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: you don't have access to view products for this business")
	}

	p, err := s.repo.GetProductByID(productID, businessID)
	if err != nil {
		return nil, errors.New("product not found")
	}

	// --- 🛠️ MESIN KALKULATOR DISKON & HARGA VARIAN ---
	now := time.Now()

	originalMinPrice := p.MinPrice
	originalMaxPrice := p.MaxPrice
	
	// Kebijakan fallback jika kolom virtual bernilai nol
	if originalMinPrice == 0 && originalMaxPrice == 0 {
		originalMinPrice = p.Price
		originalMaxPrice = p.Price
	}

	finalMinPrice := originalMinPrice
	finalMaxPrice := originalMaxPrice
	isDiscounted := false

	// Cek apakah ada nominal diskon yang dipasang
	if p.Discount > 0 {
		isDiscountActive := true
		
		// Validasi apakah waktu promo di dunia nyata sudah masuk jadwal atau belum
		if p.DiscountStartAt != nil && p.DiscountEndAt != nil {
			if now.Before(*p.DiscountStartAt) || now.After(*p.DiscountEndAt) {
				isDiscountActive = false // Diskon tidak aktif (belum mulai / kedaluwarsa)
			}
		}

		if isDiscountActive {
			isDiscounted = true
			finalMinPrice = originalMinPrice - p.Discount
			finalMaxPrice = originalMaxPrice - p.Discount

			// Safety Guard: Mencegah harga jebol ke angka minus/negatif
			if finalMinPrice < 0 {
				finalMinPrice = 0
			}
			if finalMaxPrice < 0 {
				finalMaxPrice = 0
			}
		}
	}

	// --- MAPPING KE DTO OUTPUT ---
	response := &models.ProductDetailResponse{
		ProductID:          p.ProductID,
		BusinessID:         p.BusinessID,
		ProductGroupID:     p.ProductGroupID,
		ProductImages:      p.ProductImages,
		ProductName:        p.ProductName,
		ProductDescription: p.ProductDescription,
		
		Price:           p.Price,
		Discount:        p.Discount,
		DiscountStartAt: p.DiscountStartAt,
		DiscountEndAt:   p.DiscountEndAt,

		HasVariants:      p.HasVariants,
		OriginalMinPrice: originalMinPrice,
		OriginalMaxPrice: originalMaxPrice,
		FinalMinPrice:    finalMinPrice,
		FinalMaxPrice:    finalMaxPrice,
		IsDiscounted:     isDiscounted,

		StockType:         p.StockType,
		StockQuantity:     p.StockQuantity,
		LowStockThreshold: p.LowStockThreshold,
		IsLowStock:        (p.StockType == "in_stock" && p.StockQuantity <= p.LowStockThreshold),
		
		Tags:         p.Tags,
		IsActive:     p.IsActive,
		Rating:       p.AverageRating,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		Group:        p.Group,
		Variants:     p.Variants,
	}

	return response, nil
}

// ==========================================
// PENGAMBILAN DATA (CATALOG & LIST)
// ==========================================

func (s *productService) GetCatalog(requesterID, role, businessID string) ([]models.CatalogGroupResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden access")
	}

	isActive := true
	products, err := s.repo.GetProducts(businessID, "", "", "in_stock", &isActive, "display_order", "", 1000)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]*models.CatalogGroupResponse)

	uncategorizedGroup := &models.CatalogGroupResponse{
		ProductGroupID: "uncategorized",
		Name:           "Lainnya",
		DisplayOrder:   999,
		Items:          []models.ProductListItem{},
	}

	now := time.Now()

	for _, p := range products {
		var thumbnail string
		if len(p.ProductImages) > 0 {
			thumbnail = p.ProductImages[0]
		}

		// 🪄 LOGIKA HARGA MIN/MAX & DISKON
		originalMinPrice := p.MinPrice
		originalMaxPrice := p.MaxPrice
		if originalMinPrice == 0 && originalMaxPrice == 0 {
			originalMinPrice = p.Price
			originalMaxPrice = p.Price
		}

		finalMinPrice := originalMinPrice
		finalMaxPrice := originalMaxPrice
		isDiscounted := false

		if p.Discount > 0 {
			isDiscountActive := true
			if p.DiscountStartAt != nil && p.DiscountEndAt != nil {
				if now.Before(*p.DiscountStartAt) || now.After(*p.DiscountEndAt) {
					isDiscountActive = false
				}
			}

			if isDiscountActive {
				isDiscounted = true
				finalMinPrice = originalMinPrice - p.Discount
				finalMaxPrice = originalMaxPrice - p.Discount

				// Guard Clause: Mencegah harga minus
				if finalMinPrice < 0 {
					finalMinPrice = 0
				}
				if finalMaxPrice < 0 {
					finalMaxPrice = 0
				}
			}
		}

		item := models.ProductListItem{
			ProductID:   p.ProductID,
			ProductName: p.ProductName,
			ImageURL:    thumbnail,
			HasVariants: p.HasVariants,

			// Mapping Harga
			OriginalMinPrice: originalMinPrice,
			OriginalMaxPrice: originalMaxPrice,
			FinalMinPrice:    finalMinPrice,
			FinalMaxPrice:    finalMaxPrice,
			IsDiscounted:     isDiscounted,

			StockType:     p.StockType,
			StockQuantity: p.StockQuantity,
			IsLowStock:    (p.StockType == "in_stock" && p.StockQuantity <= p.LowStockThreshold),
			Rating:        p.AverageRating,
			LikeCount:     p.LikeCount,
			CommentCount:  p.CommentCount,
		}

		if p.Group != nil {
			if existingGroup, exists := groupMap[p.Group.ProductGroupID]; exists {
				existingGroup.Items = append(existingGroup.Items, item)
			} else {
				groupMap[p.Group.ProductGroupID] = &models.CatalogGroupResponse{
					ProductGroupID: p.Group.ProductGroupID,
					BusinessID:     p.Group.BusinessID,
					Name:           p.Group.Name,
					DisplayOrder:   p.Group.DisplayOrder,
					IsActive:       p.Group.IsActive,
					CreatedAt:      p.Group.CreatedAt,
					UpdatedAt:      p.Group.UpdatedAt,
					Items:          []models.ProductListItem{item},
				}
			}
		} else {
			uncategorizedGroup.Items = append(uncategorizedGroup.Items, item)
		}
	}

	var result []models.CatalogGroupResponse
	for _, g := range groupMap {
		result = append(result, *g)
	}

	if len(uncategorizedGroup.Items) > 0 {
		result = append(result, *uncategorizedGroup)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DisplayOrder < result[j].DisplayOrder
	})

	return result, nil
}

func (s *productService) GetAllProducts(requesterID, role, businessID string, search, productGroupID, stockType string, isActive *bool, sort string, cursor string, limit int) (*models.GetProductsResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: you don't have access to view products for this business")
	}

	products, err := s.repo.GetProducts(businessID, search, productGroupID, stockType, isActive, sort, cursor, limit+1)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetProductSummary(businessID, search, productGroupID, isActive)
	if err != nil {
		return nil, err
	}

	var responseList []models.ProductListItem
	hasMore := false
	nextCursor := ""

	if len(products) > limit {
		hasMore = true
		products = products[:limit]
	}

	now := time.Now()

	for _, p := range products {
		var thumbnail string
		if len(p.ProductImages) > 0 {
			thumbnail = p.ProductImages[0]
		}

		// 🪄 LOGIKA HARGA MIN/MAX & DISKON (Sama dengan GetCatalog)
		originalMinPrice := p.MinPrice
		originalMaxPrice := p.MaxPrice
		if originalMinPrice == 0 && originalMaxPrice == 0 {
			originalMinPrice = p.Price
			originalMaxPrice = p.Price
		}

		finalMinPrice := originalMinPrice
		finalMaxPrice := originalMaxPrice
		isDiscounted := false

		if p.Discount > 0 {
			isDiscountActive := true
			if p.DiscountStartAt != nil && p.DiscountEndAt != nil {
				if now.Before(*p.DiscountStartAt) || now.After(*p.DiscountEndAt) {
					isDiscountActive = false
				}
			}

			if isDiscountActive {
				isDiscounted = true
				finalMinPrice = originalMinPrice - p.Discount
				finalMaxPrice = originalMaxPrice - p.Discount

				if finalMinPrice < 0 {
					finalMinPrice = 0
				}
				if finalMaxPrice < 0 {
					finalMaxPrice = 0
				}
			}
		}

		responseList = append(responseList, models.ProductListItem{
			ProductID:   p.ProductID,
			ProductName: p.ProductName,
			ImageURL:    thumbnail,
			HasVariants: p.HasVariants,

			// Mapping Harga
			OriginalMinPrice: originalMinPrice,
			OriginalMaxPrice: originalMaxPrice,
			FinalMinPrice:    finalMinPrice,
			FinalMaxPrice:    finalMaxPrice,
			IsDiscounted:     isDiscounted,

			StockType:     p.StockType,
			StockQuantity: p.StockQuantity,
			IsLowStock:    (p.StockType == "in_stock" && p.StockQuantity <= p.LowStockThreshold),
			Rating:        p.AverageRating,
			LikeCount:     p.LikeCount,
			CommentCount:  p.CommentCount,
		})
	}

	if len(responseList) > 0 && hasMore {
		nextCursor = responseList[len(responseList)-1].ProductID
	}

	response := &models.GetProductsResponse{
		Products: responseList,
		Summary:  *summary,
		Pagination: models.PaginationMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}

	return response, nil
}

// 6. BULK UPDATE STATUS
func (s *productService) BulkUpdateStatus(requesterID, role, businessID string, productIDs []string, isActive bool) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: you don't have access to manage products for this business")
	}
	return s.repo.BulkUpdateStatus(businessID, productIDs, isActive)
}

// 7. BULK DELETE
func (s *productService) BulkDelete(requesterID, role, businessID string, productIDs []string) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: you don't have access to manage products for this business")
	}
	return s.repo.BulkDelete(businessID, productIDs)
}
