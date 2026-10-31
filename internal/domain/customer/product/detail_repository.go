package product

import (
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductDetailRepository interface {
	GetProductWithBusiness(productID string) (*models.Product, *models.Business, error)
	GetSimilarProducts(tags []string, limit int, excludeProductID string) ([]models.Product, error)
	GetActivePromotionForProduct(businessID, productID string) (*models.Promotion, *models.PromotionProduct, error)
}

type productDetailRepository struct {
	db *gorm.DB
}

func NewProductDetailRepository(db *gorm.DB) ProductDetailRepository {
	return &productDetailRepository{db}
}

func (r *productDetailRepository) GetProductWithBusiness(productID string) (*models.Product, *models.Business, error) {
	var product models.Product
	if err := r.db.Preload("PackageItems.ItemProduct").Where("product_id = ?", productID).First(&product).Error; err != nil {
		return nil, nil, err
	}

	var business models.Business
	if err := r.db.Where("business_id = ?", product.BusinessID).First(&business).Error; err != nil {
		return nil, nil, err
	}

	return &product, &business, nil
}

func (r *productDetailRepository) GetSimilarProducts(tags []string, limit int, excludeProductID string) ([]models.Product, error) {
	var products []models.Product
	query := r.db.Where("product_id != ? AND is_active = ?", excludeProductID, true)

	if len(tags) > 0 {
		// PostgreSQL JSONB array contains operator ?|
		// Find products that have ANY of these tags
		query = query.Where("tags ?| array[?]", tags)
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&products).Error
	if err != nil {
		// Fallback if the jsonb array query fails or returns empty, just get random products (or latest)
		r.db.Where("product_id != ? AND is_active = ?", excludeProductID, true).Order("created_at DESC").Limit(limit).Find(&products)
	}

	return products, nil
}

func (r *productDetailRepository) GetActivePromotionForProduct(businessID, productID string) (*models.Promotion, *models.PromotionProduct, error) {
	now := time.Now()

	// 1. Cek dari tabel promotion_products & promotions
	var pp models.PromotionProduct
	err := r.db.
		Preload("Product").
		Joins("JOIN promotions p ON p.promotion_id = promotion_products.promotion_id").
		Where("p.business_id = ? AND promotion_products.product_id = ? AND p.status = ? AND p.start_at <= ? AND p.end_at >= ?",
			businessID, productID, models.PromotionStatusActive, now, now).
		Order("p.created_at DESC").
		First(&pp).Error

	if err == nil {
		var promo models.Promotion
		if errPromo := r.db.Preload("Post").Preload("MainProduct").Where("promotion_id = ?", pp.PromotionID).First(&promo).Error; errPromo == nil {
			return &promo, &pp, nil
		}
	}

	// 2. Cek jika produk merupakan main_product_id dari promosi aktif
	var promo models.Promotion
	errMain := r.db.
		Preload("Post").
		Preload("MainProduct").
		Where("business_id = ? AND main_product_id = ? AND status = ? AND start_at <= ? AND end_at >= ?",
			businessID, productID, models.PromotionStatusActive, now, now).
		Order("created_at DESC").
		First(&promo).Error

	if errMain == nil {
		return &promo, nil, nil
	}

	return nil, nil, gorm.ErrRecordNotFound
}

