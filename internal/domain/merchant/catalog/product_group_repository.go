package catalog

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"

	"gorm.io/gorm"
)

type ProductGroupRepository interface {
	HasPermission(accountID, businessID, permKey string) bool
	CheckBusinessExists(businessID string) bool
	CheckNameExists(businessID, name string) bool
	GetGroups(businessID string) ([]models.ProductGroup, error)
	HasActivePromotion(businessID string) bool
	CreateGroup(group *models.ProductGroup) error
	GetMaxDisplayOrder(businessID string) int
	UpdateGroup(group *models.ProductGroup) error
	UpdateDisplayOrder(businessID, groupID string, newOrder int) error
	GetGroupByID(businessID, productGroupID string) (*models.ProductGroup, error)
	DeleteGroup(businessID, productGroupID string) error
	CountProductsInGroup(businessID, productGroupID string) (int64, error)

	// Untuk memanipulasi isi grup
	AddProductsToGroup(businessID, productGroupID string, productIDs []string) error
	RemoveProductsFromGroup(businessID string, productIDs []string) error
}

type productGroupRepository struct {
	db *gorm.DB
}

func NewProductGroupRepository(db *gorm.DB) ProductGroupRepository {
	return &productGroupRepository{db}
}

func (r *productGroupRepository) HasPermission(accountID, businessID, permKey string) bool {
	return permissions.HasPermission(r.db, accountID, businessID, permKey)
}

func (r *productGroupRepository) CheckBusinessExists(businessID string) bool {
	var count int64
	r.db.Model(&models.Business{}).Where("business_id = ?", businessID).Count(&count)
	return count > 0
}

func (r *productGroupRepository) CheckNameExists(businessID, name string) bool {
	var count int64
	r.db.Model(&models.ProductGroup{}).Where("business_id = ? AND name = ?", businessID, name).Count(&count)
	return count > 0
}

func (r *productGroupRepository) GetGroups(businessID string) ([]models.ProductGroup, error) {
	var groups []models.ProductGroup
	// Diurutkan berdasarkan display_order terkecil ke terbesar, lalu ditapis berdasarkan yang aktif saja jika diperlukan
	err := r.db.Where("business_id = ?", businessID).Order("display_order ASC, created_at DESC").Find(&groups).Error
	return groups, err
}

func (r *productGroupRepository) HasActivePromotion(businessID string) bool {
	var count int64
	r.db.Table("promotions").
		Where("business_id = ? AND status = ?", businessID, models.PromotionStatusActive).
		Count(&count)
	return count > 0
}

func (r *productGroupRepository) CreateGroup(group *models.ProductGroup) error {
	return r.db.Create(group).Error
}

func (r *productGroupRepository) GetMaxDisplayOrder(businessID string) int {
	var maxOrder int
	// Cari nilai maksimum display_order. Jika toko belum punya grup sama sekali, kembalikan 0.
	r.db.Model(&models.ProductGroup{}).
		Where("business_id = ?", businessID).
		Select("COALESCE(MAX(display_order), 0)").
		Scan(&maxOrder)

	return maxOrder
}

func (r *productGroupRepository) GetGroupByID(businessID, productGroupID string) (*models.ProductGroup, error) {
	var group models.ProductGroup
	err := r.db.Where("business_id = ? AND product_group_id = ?", businessID, productGroupID).First(&group).Error
	return &group, err
}

func (r *productGroupRepository) UpdateGroup(group *models.ProductGroup) error {
	return r.db.Save(group).Error
}

func (r *productGroupRepository) UpdateDisplayOrder(businessID, productGroupID string, newOrder int) error {
	return r.db.Model(&models.ProductGroup{}).
		Where("business_id = ? AND product_group_id = ?", businessID, productGroupID).
		Update("display_order", newOrder).Error
}

func (r *productGroupRepository) CountProductsInGroup(businessID, productGroupID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Product{}).Where("business_id = ? AND product_group_id = ?", businessID, productGroupID).Count(&count).Error
	return count, err
}

func (r *productGroupRepository) DeleteGroup(businessID, productGroupID string) error {
	return r.db.Where("business_id = ? AND product_group_id = ?", businessID, productGroupID).Delete(&models.ProductGroup{}).Error
}

// Memasukkan produk ke grup ini
func (r *productGroupRepository) AddProductsToGroup(businessID, productGroupID string, productIDs []string) error {
	if len(productIDs) == 0 {
		return nil
	}
	return r.db.Model(&models.Product{}).
		Where("business_id = ? AND product_id IN ?", businessID, productIDs).
		Update("product_group_id", productGroupID).Error
}

func (r *productGroupRepository) RemoveProductsFromGroup(businessID string, productIDs []string) error {
	if len(productIDs) == 0 {
		return nil
	}

	// Gunakan Debug() untuk melihat query asli
	// Gunakan Updates dengan map untuk memastikan NULL terkirim
	result := r.db.Debug().Model(&models.Product{}).
		Where("business_id = ? AND product_id IN ?", businessID, productIDs).
		Updates(map[string]interface{}{"product_group_id": nil})

	if result.Error != nil {
		return result.Error
	}

	// Cek apakah ada data yang berhasil di-update
	if result.RowsAffected == 0 {
		return errors.New("tidak ada produk ditemukan dengan ID tersebut di toko ini")
	}

	return nil
}
