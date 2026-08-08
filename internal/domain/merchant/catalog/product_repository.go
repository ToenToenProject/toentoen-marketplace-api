package catalog

import (
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductRepository interface {
	HasAccessToBusiness(accountID, role, businessID string) bool
	CreateProduct(product *models.Product) error

	// Fitur Varian
	CreateVariant(variant *models.ProductVariant) error
	GetVariantByID(variantID, productID string) (*models.ProductVariant, error)
	UpdateVariant(variant *models.ProductVariant) error
	DeleteVariant(variantID, productID string) error

	GetProductByID(productID, businessID string) (*models.Product, error)

	// FUNGSI DIPERBARUI: Menggunakan Cursor dan Preload
	GetProducts(businessID, search, productGroupID, stockType string, isActive *bool, sort string, cursor string, limit int) ([]models.Product, error)
	GetProductSummary(businessID, search, productGroupID string, isActive *bool) (*models.ProductSummary, error)

	UpdateProduct(product *models.Product) error
	DeleteProduct(productID, businessID string) error

	// FUNGSI BARU: Bulk Operations
	BulkUpdateStatus(businessID string, productIDs []string, isActive bool) error
	BulkDelete(businessID string, productIDs []string) error
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db}
}

// 1. OTORISASI
func (r *productRepository) HasAccessToBusiness(accountID, role, businessID string) bool {
	var count int64

	if role == "owner" {
		r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count)
	} else if role == "business_admin" {
		r.db.Model(&models.BusinessAdminAssignment{}).Where("business_id = ? AND admin_id = ? AND is_active = ?", businessID, accountID, true).Count(&count)
	}

	return count > 0
}

// 2. CREATE
func (r *productRepository) CreateProduct(product *models.Product) error {
	return r.db.Create(product).Error
}

// ==========================================
// IMPLEMENTASI PRODUCT VARIANTS
// ==========================================

func (r *productRepository) CreateVariant(variant *models.ProductVariant) error {
	return r.db.Create(variant).Error
}

func (r *productRepository) GetVariantByID(variantID, productID string) (*models.ProductVariant, error) {
	var variant models.ProductVariant
	err := r.db.Where("variant_id = ? AND product_id = ?", variantID, productID).First(&variant).Error
	return &variant, err
}

func (r *productRepository) UpdateVariant(variant *models.ProductVariant) error {
	return r.db.Save(variant).Error
}

func (r *productRepository) DeleteVariant(variantID, productID string) error {
	return r.db.Where("variant_id = ? AND product_id = ?", variantID, productID).Delete(&models.ProductVariant{}).Error
}

// 3. READ (Single)
func (r *productRepository) GetProductByID(productID, businessID string) (*models.Product, error) {
	var product models.Product
	
	err := r.db.
		Preload("Group").
		Preload("Variants").
		Select(`
			products.*, 
			
			-- 1. Cek keberadaan varian aktif
			EXISTS(
				SELECT 1 FROM product_variants 
				WHERE product_variants.product_id = products.product_id 
				AND product_variants.is_active = true
			) AS has_variants,
			
			-- 2. Hitung batas harga termurah
			COALESCE(
				(SELECT MIN(products.price + product_variants.additional_price) 
				 FROM product_variants 
				 WHERE product_variants.product_id = products.product_id 
				 AND product_variants.is_active = true), 
				products.price
			) AS min_price,

			-- 3. Hitung batas harga termahal
			COALESCE(
				(SELECT MAX(products.price + product_variants.additional_price) 
				 FROM product_variants 
				 WHERE product_variants.product_id = products.product_id 
				 AND product_variants.is_active = true), 
				products.price
			) AS max_price
		`).
		Where("product_id = ? AND business_id = ?", productID, businessID).
		First(&product).Error

	return &product, err
}

// 4. ADVANCED READ (Cursor Pagination + Preload Group)
func (r *productRepository) GetProducts(businessID, search, productGroupID, stockType string, isActive *bool, sort string, cursor string, limit int) ([]models.Product, error) {
	var products []models.Product

	// 🪄 PERBAIKAN 1: Hapus Preload("Variants") di sini agar payload tetap ringan
	query := r.db.Model(&models.Product{}).Preload("Group").Where("business_id = ?", businessID)

	// Subquery untuk HasVariants dan MinPrice
	query = query.Select(`
		products.*, 
		
		-- 1. Cek apakah punya varian aktif
		EXISTS(
			SELECT 1 FROM product_variants 
			WHERE product_variants.product_id = products.product_id 
			AND product_variants.is_active = true
		) AS has_variants,
		
		-- 2. Hitung harga termurah
		COALESCE(
			(SELECT MIN(products.price + product_variants.additional_price) 
			 FROM product_variants 
			 WHERE product_variants.product_id = products.product_id 
			 AND product_variants.is_active = true), 
			products.price
		) AS min_price,

        -- 3. Hitung harga termahal (TAMBAHAN BARU)
		COALESCE(
			(SELECT MAX(products.price + product_variants.additional_price) 
			 FROM product_variants 
			 WHERE product_variants.product_id = products.product_id 
			 AND product_variants.is_active = true), 
			products.price
		) AS max_price
	`)

	// --- 1. FILTERING ---
	if search != "" {
		searchTerms := strings.Fields(search)
		searchQuery := r.db.Where("")

		for _, term := range searchTerms {
			wildcardTerm := "%" + term + "%"
			searchQuery = searchQuery.Or(
				"product_name ILIKE ? OR product_description ILIKE ? OR tags::text ILIKE ?",
				wildcardTerm, wildcardTerm, wildcardTerm,
			)
		}
		query = query.Where(searchQuery)
	}

	if productGroupID != "" {
		query = query.Where("product_group_id = ?", productGroupID)
	}

	if stockType != "" {
		query = query.Where("stock_type = ?", stockType)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	// --- 2. LOGIKA KURSOR & SORTING ---
	if cursor != "" {
		var cursorProduct models.Product
		if err := r.db.Select("price", "created_at").Where("product_id = ?", cursor).First(&cursorProduct).Error; err == nil {
			if sort == "lowest_price" {
				query = query.Where("price > ? OR (price = ? AND product_id > ?)", cursorProduct.Price, cursorProduct.Price, cursor)
			} else if sort == "highest_price" {
				query = query.Where("price < ? OR (price = ? AND product_id < ?)", cursorProduct.Price, cursorProduct.Price, cursor)
			} else {
				query = query.Where("created_at < ? OR (created_at = ? AND product_id < ?)", cursorProduct.CreatedAt, cursorProduct.CreatedAt, cursor)
			}
		}
	}

	// Eksekusi Sorting 
	if sort == "lowest_price" {
		query = query.Order("price ASC, product_id ASC")
	} else if sort == "highest_price" {
		query = query.Order("price DESC, product_id DESC")
	} else {
		query = query.Order("created_at DESC, product_id DESC")
	}

	// --- 3. EKSEKUSI FINAL ---
	// 🪄 PERBAIKAN 2: Hapus query.Select(...) yang duplikat di sini
	err := query.Limit(limit).Find(&products).Error

	return products, err
}

// 5. UPDATE
func (r *productRepository) UpdateProduct(product *models.Product) error {
	return r.db.Save(product).Error
}

// 6. DELETE
func (r *productRepository) DeleteProduct(productID, businessID string) error {
	return r.db.Where("product_id = ? AND business_id = ?", productID, businessID).Delete(&models.Product{}).Error
}

// 7. BULK UPDATE STATUS (Aktif/Nonaktif masal)
func (r *productRepository) BulkUpdateStatus(businessID string, productIDs []string, isActive bool) error {
	return r.db.Model(&models.Product{}).
		Where("business_id = ? AND product_id IN ?", businessID, productIDs).
		Update("is_active", isActive).Error
}

// 8. BULK DELETE (Hapus masal)
func (r *productRepository) BulkDelete(businessID string, productIDs []string) error {
	// Menggunakan fitur Soft-Delete GORM (hanya mengisi deleted_at, tidak menghapus permanen dari DB)
	return r.db.Where("business_id = ? AND product_id IN ?", businessID, productIDs).
		Delete(&models.Product{}).Error
}

func (r *productRepository) GetProductSummary(businessID, search, productGroupID string, isActive *bool) (*models.ProductSummary, error) {
	var summary models.ProductSummary

	query := r.db.Model(&models.Product{}).Where("business_id = ?", businessID)

	if search != "" {
		searchTerms := strings.Fields(search)
		searchQuery := r.db.Where("")

		for _, term := range searchTerms {
			wildcardTerm := "%" + term + "%"
			searchQuery = searchQuery.Or(
				"product_name ILIKE ? OR product_description ILIKE ? OR tags::text ILIKE ?",
				wildcardTerm, wildcardTerm, wildcardTerm,
			)
		}
		query = query.Where(searchQuery)
	}

	if productGroupID != "" && productGroupID != "all" {
		query = query.Where("product_group_id = ?", productGroupID)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	err := query.Select(`
		COUNT(product_id) as total_all,
		SUM(CASE WHEN stock_type = 'in_stock' THEN 1 ELSE 0 END) as total_in_stock,
		SUM(CASE WHEN stock_type = 'pre_order' THEN 1 ELSE 0 END) as total_pre_order,
		SUM(CASE WHEN stock_type = 'limited' THEN 1 ELSE 0 END) as total_limited,
		SUM(CASE WHEN stock_type = 'out_of_stock' THEN 1 ELSE 0 END) as total_out_of_stock
	`).Scan(&summary).Error

	return &summary, err
}
