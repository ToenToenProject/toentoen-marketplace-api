package catalog

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"

	"gorm.io/gorm"
)

type PackageRepository interface {
	HasAccessToBusiness(accountID, role, businessID string) bool
	HasPermission(accountID, businessID, permKey string) bool
	CheckProductGroupExists(businessID, productGroupID string) bool
	GetProductsByIDs(businessID string, productIDs []string) ([]models.Product, error)
	CreatePackage(pkgProduct *models.Product, items []models.ProductPackageItem) error
	UpdatePackage(pkgProduct *models.Product, items []models.ProductPackageItem) error
	DeletePackage(businessID, productID string) error
	GetPackages(businessID string) ([]models.Product, error)
	GetPackageDetail(businessID, productID string) (*models.Product, error)
}

type packageRepository struct {
	db *gorm.DB
}

func NewPackageRepository(db *gorm.DB) PackageRepository {
	return &packageRepository{db}
}

func (r *packageRepository) HasAccessToBusiness(accountID, role, businessID string) bool {
	var count int64
	if role == "owner" {
		r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count)
	} else if role == "business_admin" {
		r.db.Model(&models.BusinessAdminAssignment{}).Where("business_id = ? AND admin_id = ? AND status IN ('active', 'accepted', 'scheduled_for_deletion')", businessID, accountID).Count(&count)
	}
	return count > 0
}

func (r *packageRepository) HasPermission(accountID, businessID, permKey string) bool {
	return permissions.HasPermission(r.db, accountID, businessID, permKey)
}

func (r *packageRepository) CheckProductGroupExists(businessID, productGroupID string) bool {
	var count int64
	r.db.Model(&models.ProductGroup{}).Where("business_id = ? AND product_group_id = ?", businessID, productGroupID).Count(&count)
	return count > 0
}

func (r *packageRepository) GetProductsByIDs(businessID string, productIDs []string) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("business_id = ? AND product_id IN (?) AND deleted_at IS NULL", businessID, productIDs).Find(&products).Error
	return products, err
}

func (r *packageRepository) CreatePackage(pkgProduct *models.Product, items []models.ProductPackageItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pkgProduct).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *packageRepository) UpdatePackage(pkgProduct *models.Product, items []models.ProductPackageItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(pkgProduct).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Where("package_product_id = ?", pkgProduct.ProductID).Delete(&models.ProductPackageItem{}).Error; err != nil {
				return err
			}
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *packageRepository) DeletePackage(businessID, productID string) error {
	res := r.db.Where("business_id = ? AND product_id = ? AND is_package = ?", businessID, productID, true).Delete(&models.Product{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("paket tidak ditemukan atau bukan milik bisnis ini")
	}
	return nil
}

func (r *packageRepository) GetPackages(businessID string) ([]models.Product, error) {
	var products []models.Product
	err := r.db.
		Preload("Group").
		Preload("PackageItems.ItemProduct").
		Where("business_id = ? AND is_package = ?", businessID, true).
		Order("created_at DESC").
		Find(&products).Error
	return products, err
}

func (r *packageRepository) GetPackageDetail(businessID, productID string) (*models.Product, error) {
	var product models.Product
	err := r.db.
		Preload("Group").
		Preload("PackageItems.ItemProduct").
		Where("business_id = ? AND product_id = ? AND is_package = ?", businessID, productID, true).
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}
