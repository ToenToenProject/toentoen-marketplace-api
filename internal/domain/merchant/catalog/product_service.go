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
	lowStockThreshold := 5

	var productGroupID *string
	if input.ProductGroupID != nil {
		cleanGroupID := strings.TrimSpace(*input.ProductGroupID)
		if cleanGroupID != "" {
			if !s.repo.CheckProductGroupExists(businessID, cleanGroupID) {
				return fmt.Errorf("grup produk dengan ID '%s' tidak ditemukan pada bisnis ini", cleanGroupID)
			}
			productGroupID = &cleanGroupID
		}
	}

	// Origin & Country handling (English: "domestic" | "imported")
	origin := strings.TrimSpace(input.ProductOrigin)
	var originCountry *string
	if origin == "" || strings.EqualFold(origin, "domestic") || strings.EqualFold(origin, "local") || strings.EqualFold(origin, "dalam_negeri") || strings.EqualFold(origin, "indonesia") {
		origin = "domestic"
		indo := "Indonesia"
		originCountry = &indo
	} else if strings.EqualFold(origin, "imported") || strings.EqualFold(origin, "luar_negeri") {
		origin = "imported"
		if input.OriginCountry != nil && strings.TrimSpace(*input.OriginCountry) != "" {
			country := strings.TrimSpace(*input.OriginCountry)
			originCountry = &country
		} else {
			return errors.New("negara asal (origin_country) wajib diisi untuk produk luar negeri / impor")
		}
	} else {
		// Jika FE langsung mengirim nama negara
		origin = "imported"
		if input.OriginCountry != nil && strings.TrimSpace(*input.OriginCountry) != "" {
			country := strings.TrimSpace(*input.OriginCountry)
			originCountry = &country
		} else {
			originCountry = &origin
		}
	}

	var rawHashtags []string
	if len(input.Hashtags) > 0 {
		rawHashtags = input.Hashtags
	} else if len(input.Tags) > 0 {
		rawHashtags = input.Tags
	}

	if len(rawHashtags) == 0 {
		return errors.New("hashtags produk wajib diisi minimal 1 hashtag")
	}

	normalizedHashtags := utils.NormalizeHashtags(rawHashtags)
	if len(normalizedHashtags) == 0 {
		return errors.New("hashtags produk tidak boleh kosong setelah dinormalisasi")
	}

	product := models.Product{
		ProductID:          utils.GenerateUUID("PRD"),
		BusinessID:         businessID,
		ProductGroupID:     productGroupID,
		ProductPreview:     input.ProductPreview,
		ProductName:        input.ProductName,
		ProductOrigin:      origin,
		OriginCountry:      originCountry,
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
		Tags:               normalizedHashtags,
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
			input.OriginCountry != nil || input.ProductDescription != nil || input.Price != nil || input.Discount != nil ||
			input.DiscountType != nil || input.MaxDiscount != nil || input.StockType != nil || input.Hashtags != nil || input.Tags != nil || input.IsActive != nil ||
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
		cleanGroupID := strings.TrimSpace(*input.ProductGroupID)
		if cleanGroupID != "" {
			if !s.repo.CheckProductGroupExists(businessID, cleanGroupID) {
				return fmt.Errorf("grup produk dengan ID '%s' tidak ditemukan pada bisnis ini", cleanGroupID)
			}
			product.ProductGroupID = &cleanGroupID
		} else {
			product.ProductGroupID = nil
		}
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
		if strings.EqualFold(cleanOrigin, "domestic") || strings.EqualFold(cleanOrigin, "local") || strings.EqualFold(cleanOrigin, "dalam_negeri") || strings.EqualFold(cleanOrigin, "indonesia") {
			product.ProductOrigin = "domestic"
			indo := "Indonesia"
			product.OriginCountry = &indo
		} else if strings.EqualFold(cleanOrigin, "imported") || strings.EqualFold(cleanOrigin, "luar_negeri") {
			product.ProductOrigin = "imported"
			if input.OriginCountry != nil && strings.TrimSpace(*input.OriginCountry) != "" {
				country := strings.TrimSpace(*input.OriginCountry)
				product.OriginCountry = &country
			} else if product.OriginCountry == nil || *product.OriginCountry == "" || strings.EqualFold(*product.OriginCountry, "Indonesia") {
				return errors.New("negara asal (origin_country) wajib diisi untuk produk luar negeri / impor")
			}
		} else if cleanOrigin != "" {
			product.ProductOrigin = "imported"
			product.OriginCountry = &cleanOrigin
		}
	}
	if input.OriginCountry != nil {
		cleanCountry := strings.TrimSpace(*input.OriginCountry)
		if cleanCountry != "" {
			product.OriginCountry = &cleanCountry
			if strings.EqualFold(cleanCountry, "Indonesia") {
				product.ProductOrigin = "domestic"
			} else {
				product.ProductOrigin = "imported"
			}
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
	if input.Hashtags != nil {
		product.Tags = utils.NormalizeHashtags(*input.Hashtags)
	} else if input.Tags != nil {
		product.Tags = utils.NormalizeHashtags(*input.Tags)
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
		OriginCountry:      p.OriginCountry,
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
		Hashtags:          p.Tags,
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

			StockType:     p.StockType,
			ProductOrigin: p.ProductOrigin,
			OriginCountry: p.OriginCountry,
			Rating:        p.AverageRating,
			LikeCount:     p.LikeCount,
			ReviewCount:   p.ReviewCount,
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
