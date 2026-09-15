package catalog

import (
	"errors"
	"fmt"
	"strings"
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
		return errors.New("forbidden: Anda tidak memiliki akses untuk mengelola produk di bisnis ini")
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermAddDeleteProducts) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menambah produk")
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	discountType := "nominal"
	if input.DiscountType != nil {
		cleanType := strings.ToLower(strings.TrimSpace(*input.DiscountType))
		if cleanType != "" {
			if cleanType != "nominal" && cleanType != "percentage" {
				return errors.New("tipe diskon tidak valid, pilih 'nominal' atau 'percentage'")
			}
			discountType = cleanType
		}
	}

	if input.Discount < 0 {
		return errors.New("nilai diskon tidak boleh bernilai negatif")
	}

	if discountType == "percentage" && input.Discount > 100 {
		return errors.New("persentase diskon tidak boleh melebihi 100%")
	}

	if discountType == "nominal" && input.Discount > input.Price {
		return errors.New("nominal diskon tidak boleh melebihi harga produk")
	}

	if input.MaxDiscount != nil && *input.MaxDiscount < 0 {
		return errors.New("maksimal diskon tidak boleh bernilai negatif")
	}

	discountStartAt := input.DiscountStartAt
	discountEndAt := input.DiscountEndAt
	maxDiscount := input.MaxDiscount
	if input.Discount == 0 {
		discountStartAt = nil
		discountEndAt = nil
		maxDiscount = nil
	}

	stockQuantity := 0
	if input.StockQuantity != nil {
		if *input.StockQuantity < 0 {
			return errors.New("jumlah stok tidak boleh bernilai negatif")
		}
		stockQuantity = *input.StockQuantity
	}

	lowStockThreshold := 5
	if input.LowStockThreshold != nil {
		if *input.LowStockThreshold < 0 {
			return errors.New("ambang batas stok menipis tidak boleh bernilai negatif")
		}
		lowStockThreshold = *input.LowStockThreshold
	}

	product := models.Product{
		ProductID:          utils.GenerateUUID("PRD"),
		BusinessID:         businessID,
		ProductGroupID:     input.ProductGroupID,
		ProductPreview:     input.ProductPreview,
		ProductName:        input.ProductName,
		ProductOrigin:      input.ProductOrigin,
		ProductDescription: input.ProductDescription,
		Price:              input.Price,
		DiscountType:       discountType,
		Discount:           input.Discount,
		MaxDiscount:        maxDiscount,
		DiscountStartAt:    discountStartAt,
		DiscountEndAt:      discountEndAt,
		StockType:          input.StockType,
		StockQuantity:      stockQuantity,
		LowStockThreshold:  lowStockThreshold,
		Tags:               input.Tags,
		IsActive:           isActive,
	}

	if err := s.repo.CreateProduct(&product); err != nil {
		return fmt.Errorf("gagal membuat produk: %w", err)
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

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermEditProductDetails) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengedit varian produk")
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
		return errors.New("gagal menambahkan varian")
	}
	return nil
}

func (s *productService) UpdateVariant(requesterID, role, businessID, productID, variantID string, input models.UpdateVariantInput) error {
	if _, err := s.GetProductDetail(requesterID, role, businessID, productID); err != nil {
		return err
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermEditProductDetails) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengedit varian produk")
	}

	variant, err := s.repo.GetVariantByID(variantID, productID)
	if err != nil {
		return errors.New("varian tidak ditemukan")
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
		return errors.New("gagal memperbarui varian")
	}
	return nil
}

func (s *productService) RemoveVariant(requesterID, role, businessID, productID, variantID string) error {
	if _, err := s.GetProductDetail(requesterID, role, businessID, productID); err != nil {
		return err
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermAddDeleteProducts) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menghapus varian produk")
	}

	if err := s.repo.DeleteVariant(variantID, productID); err != nil {
		return errors.New("gagal menghapus varian")
	}
	return nil
}

// 2. UPDATE PRODUCT
func (s *productService) UpdateProduct(requesterID, role, businessID, productID string, input models.UpdateProductInput) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses untuk mengelola produk di bisnis ini")
	}

	if role == "business_admin" {
		if input.ProductPreview != nil && !s.repo.HasPermission(requesterID, businessID, models.PermChangeProductImage) {
			return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah gambar/media produk")
		}

		hasOtherDetails := input.ProductGroupID != nil || input.ProductName != nil || input.ProductOrigin != nil ||
			input.ProductDescription != nil || input.Price != nil || input.Discount != nil ||
			input.DiscountType != nil || input.MaxDiscount != nil ||
			input.StockType != nil || input.StockQuantity != nil || input.LowStockThreshold != nil || input.IsActive != nil ||
			input.DiscountStartAt != nil || input.DiscountEndAt != nil
		if hasOtherDetails && !s.repo.HasPermission(requesterID, businessID, models.PermEditProductDetails) {
			return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah detail produk")
		}
	}

	product, err := s.repo.GetProductByID(productID, businessID)
	if err != nil {
		return errors.New("produk tidak ditemukan di bisnis ini")
	}

	if input.ProductGroupID != nil {
		product.ProductGroupID = input.ProductGroupID
	}
	if input.ProductPreview != nil {
		product.ProductPreview = *input.ProductPreview
	}
	if input.ProductName != nil {
		cleanName := strings.TrimSpace(*input.ProductName)
		if cleanName == "" {
			return errors.New("nama produk wajib diisi dan tidak boleh kosong")
		}
		product.ProductName = cleanName
	}
	if input.ProductOrigin != nil {
		cleanOrigin := strings.TrimSpace(*input.ProductOrigin)
		if cleanOrigin == "" {
			product.ProductOrigin = "domestic"
		} else {
			product.ProductOrigin = cleanOrigin
		}
	}
	if input.ProductDescription != nil {
		product.ProductDescription = strings.TrimSpace(*input.ProductDescription)
	}
	if input.Price != nil {
		if *input.Price < 0 {
			return errors.New("harga produk tidak boleh bernilai negatif")
		}
		product.Price = *input.Price
	}
	if input.DiscountType != nil {
		cleanType := strings.ToLower(strings.TrimSpace(*input.DiscountType))
		if cleanType != "nominal" && cleanType != "percentage" {
			return errors.New("tipe diskon tidak valid, pilih 'nominal' atau 'percentage'")
		}
		product.DiscountType = cleanType
	}
	if input.MaxDiscount != nil {
		if *input.MaxDiscount < 0 {
			return errors.New("maksimal diskon tidak boleh bernilai negatif")
		}
		product.MaxDiscount = input.MaxDiscount
	}
	if input.Discount != nil {
		if *input.Discount < 0 {
			return errors.New("nilai diskon tidak boleh bernilai negatif")
		}
		product.Discount = *input.Discount

		if *input.Discount == 0 {
			product.DiscountStartAt = nil
			product.DiscountEndAt = nil
			product.MaxDiscount = nil
		}
	}

	// Validasi silang tipe diskon vs nilai & harga
	if product.DiscountType == "percentage" && product.Discount > 100 {
		return errors.New("persentase diskon tidak boleh melebihi 100%")
	}
	if product.DiscountType == "nominal" && product.Discount > product.Price {
		return errors.New("nominal diskon tidak boleh melebihi harga produk")
	}

	if input.DiscountStartAt != nil {
		product.DiscountStartAt = input.DiscountStartAt
	}
	if input.DiscountEndAt != nil {
		product.DiscountEndAt = input.DiscountEndAt
	}
	if input.StockType != nil {
		cleanStock := strings.TrimSpace(*input.StockType)
		if cleanStock == "" {
			return errors.New("tipe stok wajib diisi dan tidak boleh kosong")
		}
		product.StockType = cleanStock
	}
	if input.StockQuantity != nil {
		if *input.StockQuantity < 0 {
			return errors.New("jumlah stok tidak boleh bernilai negatif")
		}
		product.StockQuantity = *input.StockQuantity
	}
	if input.LowStockThreshold != nil {
		if *input.LowStockThreshold < 0 {
			return errors.New("ambang batas stok menipis tidak boleh bernilai negatif")
		}
		product.LowStockThreshold = *input.LowStockThreshold
	}
	if input.IsActive != nil {
		product.IsActive = *input.IsActive
	}

	if err := s.repo.UpdateProduct(product); err != nil {
		return errors.New("gagal memperbarui produk")
	}
	return nil
}

// 3. DELETE PRODUCT
func (s *productService) DeleteProduct(requesterID, role, businessID, productID string) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses untuk mengelola produk di bisnis ini")
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermAddDeleteProducts) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menghapus produk")
	}

	if err := s.repo.DeleteProduct(productID, businessID); err != nil {
		return errors.New("gagal menghapus produk")
	}
	return nil
}

// 4. GET DETAIL PRODUCT (Single)
// Ganti fungsi ini di internal/services/product_service.go
func (s *productService) GetProductDetail(requesterID, role, businessID, productID string) (*models.ProductDetailResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses untuk melihat produk di bisnis ini")
	}

	p, err := s.repo.GetProductByID(productID, businessID)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
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
	var discountAmount float64
	var discountPct float64

	// Cek apakah ada nominal/persen diskon yang dipasang
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
			minCut, pct := utils.CalculateProductDiscount(originalMinPrice, p.DiscountType, p.Discount, p.MaxDiscount)
			maxCut, _ := utils.CalculateProductDiscount(originalMaxPrice, p.DiscountType, p.Discount, p.MaxDiscount)

			finalMinPrice = originalMinPrice - minCut
			finalMaxPrice = originalMaxPrice - maxCut
			discountAmount = minCut
			discountPct = pct
		}
	}

	// --- MAPPING KE DTO OUTPUT ---
	response := &models.ProductDetailResponse{
		ProductID:          p.ProductID,
		BusinessID:         p.BusinessID,
		ProductGroupID:     p.ProductGroupID,
		ProductPreview:     p.ProductPreview,
		ProductName:        p.ProductName,
		ProductOrigin:      p.ProductOrigin,
		ProductDescription: p.ProductDescription,
		Price:              p.Price,
		DiscountType:       p.DiscountType,
		Discount:           p.Discount,
		DiscountAmount:     discountAmount,
		DiscountPct:        discountPct,
		MaxDiscount:        p.MaxDiscount,
		DiscountStartAt:    p.DiscountStartAt,
		DiscountEndAt:      p.DiscountEndAt,

		HasVariants:      p.HasVariants,
		OriginalMinPrice: originalMinPrice,
		OriginalMaxPrice: originalMaxPrice,
		FinalMinPrice:    finalMinPrice,
		FinalMaxPrice:    finalMaxPrice,
		IsDiscounted:     isDiscounted,

		StockType:         p.StockType,
		StockQuantity:     p.StockQuantity,
		LowStockThreshold: p.LowStockThreshold,
		Tags:              p.Tags,
		IsActive:          p.IsActive,
		Rating:            p.AverageRating,
		LikeCount:         p.LikeCount,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		Group:             p.Group,
		Variants:          p.Variants,
	}

	return response, nil
}

// ==========================================
// PENGAMBILAN DATA PRODUK (LIST & FILTER)
// ==========================================

func (s *productService) GetAllProducts(requesterID, role, businessID string, search, productGroupID, stockType string, isActive *bool, sort string, cursor string, limit int) (*models.GetProductsResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses untuk melihat produk di bisnis ini")
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
		if len(p.ProductPreview) > 0 {
			thumbnail = p.ProductPreview[0]
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
		var discountAmount float64
		var discountPct float64

		if p.Discount > 0 {
			isDiscountActive := true
			if p.DiscountStartAt != nil && p.DiscountEndAt != nil {
				if now.Before(*p.DiscountStartAt) || now.After(*p.DiscountEndAt) {
					isDiscountActive = false
				}
			}

			if isDiscountActive {
				isDiscounted = true
				minCut, pct := utils.CalculateProductDiscount(originalMinPrice, p.DiscountType, p.Discount, p.MaxDiscount)
				maxCut, _ := utils.CalculateProductDiscount(originalMaxPrice, p.DiscountType, p.Discount, p.MaxDiscount)

				finalMinPrice = originalMinPrice - minCut
				finalMaxPrice = originalMaxPrice - maxCut
				discountAmount = minCut
				discountPct = pct
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

			// Mapping Diskon
			IsDiscounted:    isDiscounted,
			DiscountType:    p.DiscountType,
			Discount:        p.Discount,
			DiscountAmount:  discountAmount,
			DiscountPct:     discountPct,
			MaxDiscount:     p.MaxDiscount,
			DiscountStartAt: p.DiscountStartAt,
			DiscountEndAt:   p.DiscountEndAt,

			StockType:   p.StockType,
			Rating:      p.AverageRating,
			LikeCount:   p.LikeCount,
			ReviewCount: p.ReviewCount,
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
		return errors.New("forbidden: Anda tidak memiliki akses untuk mengelola produk di bisnis ini")
	}
	return s.repo.BulkUpdateStatus(businessID, productIDs, isActive)
}

// 7. BULK DELETE
func (s *productService) BulkDelete(requesterID, role, businessID string, productIDs []string) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses untuk mengelola produk di bisnis ini")
	}
	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermAddDeleteProducts) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menghapus produk")
	}
	return s.repo.BulkDelete(businessID, productIDs)
}
