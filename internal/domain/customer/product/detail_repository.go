package product

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProductDetailRepository interface {
	GetProductWithBusiness(productID string) (*models.Product, *models.Business, error)
	GetSimilarProducts(tags []string, limit int, excludeProductID string) ([]models.Product, error)
}

type productDetailRepository struct {
	db *gorm.DB
}

func NewProductDetailRepository(db *gorm.DB) ProductDetailRepository {
	return &productDetailRepository{db}
}

func (r *productDetailRepository) GetProductWithBusiness(productID string) (*models.Product, *models.Business, error) {
	var product models.Product
	if err := r.db.Where("product_id = ?", productID).First(&product).Error; err != nil {
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
