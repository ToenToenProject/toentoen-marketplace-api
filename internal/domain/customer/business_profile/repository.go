package business_profile

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
)

type SavedShopData struct {
	SavedBizMap      map[string]time.Time
	WishlistedBizMap map[string]int
	WishlistLatestAt map[string]time.Time
}

type Repository interface {
	GetBusinessDetail(businessID string) (models.Business, error)
	CheckBookmark(accountID, businessID string) (bool, error)
	ToggleBookmark(accountID, businessID string) (bool, error)
	GetBookmarkedBusinesses(accountID string) ([]models.Business, error)

	GetSavedAndWishlistedBizIDs(accountID string) (SavedShopData, error)
	GetBusinessesByIDs(businessIDs []string) ([]models.Business, error)

	CheckBusinessLike(accountID, businessID string) (bool, error)
	ToggleBusinessLike(accountID, businessID string) (bool, int64, error)
	GetBusinessLikeCount(businessID string) (int64, error)

	GetRecentProducts(businessID string, limit int) ([]models.Product, error)
	GetRecentPosts(businessID string, limit int) ([]models.BusinessPost, error)
	GetSimilarBusinesses(category string, city string, currentBusinessID string, limit int) ([]models.Business, error)
	GetRecentTopReviews(businessID string, limit int) ([]models.Review, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)

	GetVerificationStatus(businessID string) (isVerified bool, hasNIB bool, err error)
	BatchGetVerificationStatus(businessIDs []string) (map[string]BusinessVerificationInfo, error)

	GetAllProductGroups(businessID string) ([]models.ProductGroup, error)
	HasActivePromotion(businessID string) bool
	GetActiveBundlingPromotions(businessID string) ([]models.Promotion, error)
	GetAllProducts(businessID string, groupID string) ([]models.Product, error)
	GetAllPosts(businessID string) ([]models.BusinessPost, error)
	GetPostLikesByAccount(accountID string, postIDs []string) (map[string]bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository() Repository {
	return &repository{
		db: database.DB,
	}
}

func (r *repository) GetBusinessDetail(businessID string) (models.Business, error) {
	var business models.Business
	err := r.db.
		Preload("Categories").
		Preload("Hours").
		Where("business_id = ?", businessID).
		First(&business).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return business, errors.New("bisnis tidak ditemukan")
		}
		return business, err
	}

	return business, nil
}

func (r *repository) CheckBookmark(accountID, businessID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.BusinessBookmark{}).
		Where("account_id = ? AND business_id = ?", accountID, businessID).
		Count(&count).Error
	
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) ToggleBookmark(accountID, businessID string) (bool, error) {
	var existing models.BusinessBookmark
	err := r.db.Where("account_id = ? AND business_id = ?", accountID, businessID).First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Buat baru
			newBookmark := models.BusinessBookmark{
				AccountID:  accountID,
				BusinessID: businessID,
			}
			if errCreate := r.db.Create(&newBookmark).Error; errCreate != nil {
				return false, errCreate
			}
			return true, nil // Sekarang di-bookmark
		}
		return false, err
	}

	// Hapus (Unbookmark)
	if errDelete := r.db.Delete(&existing).Error; errDelete != nil {
		return true, errDelete // gagal hapus, masih bookmarked
	}

	return false, nil // Sudah tidak di-bookmark
}

func (r *repository) CheckBusinessLike(accountID, businessID string) (bool, error) {
	if accountID == "" || businessID == "" {
		return false, nil
	}
	var count int64
	err := r.db.Model(&models.BusinessLike{}).
		Where("account_id = ? AND business_id = ?", accountID, businessID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) GetBusinessLikeCount(businessID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.BusinessLike{}).
		Where("business_id = ?", businessID).
		Count(&count).Error
	return count, err
}

func (r *repository) ToggleBusinessLike(accountID, businessID string) (bool, int64, error) {
	var isLiked bool
	var totalLikes int64

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var b models.Business
		if err := tx.Select("business_id").Where("business_id = ?", businessID).First(&b).Error; err != nil {
			return err
		}

		var existing models.BusinessLike
		errFind := tx.Where("account_id = ? AND business_id = ?", accountID, businessID).First(&existing).Error

		if errFind != nil {
			if errors.Is(errFind, gorm.ErrRecordNotFound) {
				newLike := models.BusinessLike{
					AccountID:  accountID,
					BusinessID: businessID,
				}
				if err := tx.Create(&newLike).Error; err != nil {
					return err
				}
				isLiked = true
			} else {
				return errFind
			}
		} else {
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			isLiked = false
		}

		if err := tx.Model(&models.BusinessLike{}).Where("business_id = ?", businessID).Count(&totalLikes).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Business{}).Where("business_id = ?", businessID).UpdateColumn("like_count", totalLikes).Error; err != nil {
			return err
		}

		return nil
	})

	if err == nil {
		go meilisearch.SyncBusiness(database.DB, businessID)
	}

	return isLiked, totalLikes, err
}

func (r *repository) GetBookmarkedBusinesses(accountID string) ([]models.Business, error) {
	var businesses []models.Business
	err := r.db.
		Preload("Categories").
		Preload("Hours").
		Joins("JOIN business_bookmarks ON business_bookmarks.business_id = businesses.business_id").
		Where("business_bookmarks.account_id = ?", accountID).
		Find(&businesses).Error

	return businesses, err
}

func (r *repository) GetRecentProducts(businessID string, limit int) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("business_id = ? AND is_active = ?", businessID, true).
		Order("created_at DESC").
		Limit(limit).
		Find(&products).Error
	return products, err
}

func (r *repository) GetRecentPosts(businessID string, limit int) ([]models.BusinessPost, error) {
	var posts []models.BusinessPost
	err := r.db.Where("business_id = ?", businessID).
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *repository) GetSimilarBusinesses(category string, city string, currentBusinessID string, limit int) ([]models.Business, error) {
	var businesses []models.Business
	
	// Cari bisnis yang ada di kota yang sama, KECUALI bisnis saat ini
	query := r.db.Preload("Categories").Preload("Hours").
		Where("city = ? AND business_id != ?", city, currentBusinessID)

	// Jika kita ingin mencari berdasarkan kategori juga:
	// Di postgres_go/gorm, join ke business_category_tags agak rumit dengan gorm many2many langsung filter.
	// Untuk sederhana, kita tarik berdasarkan kota saja atau kita bisa gunakan raw query jika butuh exact category match.
	// Saat ini, ambil berdasarkan city dulu untuk "Similar in Area" sebagai awalan.
	// Jika category tidak kosong, kita cari bisnis yang tergabung dalam kategori itu
	if category != "" && category != "General" {
		query = query.Joins("JOIN business_category_tags bct ON bct.business_id = businesses.business_id").
			Joins("JOIN categories c ON c.id = bct.category_id").
			Where("c.name = ?", category)
	}

	err := query.Limit(limit).Find(&businesses).Error
	return businesses, err
}

func (r *repository) GetRecentTopReviews(businessID string, limit int) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.Preload("Account").Where("business_id = ?", businessID).
		Order("rating DESC, created_at DESC").
		Limit(limit).
		Find(&reviews).Error
	return reviews, err
}

func (r *repository) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	var summary models.ReviewSummary
	var total float64
	
	type Result struct {
		Rating int
		Count  int64
	}
	var results []Result

	err := r.db.Model(&models.Review{}).
		Select("rating, count(rating) as count").
		Where("business_id = ?", businessID).
		Group("rating").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	var totalReviews int64
	for _, res := range results {
		totalReviews += res.Count
		total += float64(res.Rating) * float64(res.Count)

		switch res.Rating {
		case 5:
			summary.Distribution.Star5 = res.Count
		case 4:
			summary.Distribution.Star4 = res.Count
		case 3:
			summary.Distribution.Star3 = res.Count
		case 2:
			summary.Distribution.Star2 = res.Count
		case 1:
			summary.Distribution.Star1 = res.Count
		}
	}

	summary.TotalReviews = totalReviews
	if totalReviews > 0 {
		summary.AverageRating = total / float64(totalReviews)
	}

	return &summary, nil
}

func (r *repository) GetAllProductGroups(businessID string) ([]models.ProductGroup, error) {
	var groups []models.ProductGroup
	err := r.db.Where("business_id = ? AND is_active = ?", businessID, true).
		Order("display_order ASC, created_at DESC").
		Find(&groups).Error
	return groups, err
}

func (r *repository) HasActivePromotion(businessID string) bool {
	var count int64
	r.db.Table("promotions").
		Where("business_id = ? AND status = ?", businessID, models.PromotionStatusActive).
		Count(&count)
	return count > 0
}

func (r *repository) GetActiveBundlingPromotions(businessID string) ([]models.Promotion, error) {
	if !database.HasPromotionTypeColumn(r.db) || !database.HasBundleItemsTable(r.db) {
		return []models.Promotion{}, nil
	}
	var promos []models.Promotion
	err := r.db.
		Preload("BundleItems.Product").
		Where("business_id = ? AND promotion_type = ? AND status = ?", businessID, models.PromotionTypeBundling, models.PromotionStatusActive).
		Order("created_at DESC").
		Find(&promos).Error
	return promos, err
}

func (r *repository) GetAllProducts(businessID string, groupID string) ([]models.Product, error) {
	var products []models.Product
	query := r.db.Where("business_id = ? AND is_active = ?", businessID, true)
	
	if groupID != "" {
		if groupID == "promotion" {
			query = query.Where("product_id IN (SELECT pp.product_id FROM promotion_products pp JOIN promotions p ON p.promotion_id = pp.promotion_id WHERE p.business_id = ? AND p.status = 'active')", businessID)
		} else {
			query = query.Where("product_group_id = ?", groupID)
		}
	}

	err := query.Order("created_at DESC").Find(&products).Error
	return products, err
}

func (r *repository) GetAllPosts(businessID string) ([]models.BusinessPost, error) {
	var posts []models.BusinessPost
	err := r.db.
		Preload("Business").
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&posts).Error
	return posts, err
}

func (r *repository) GetPostLikesByAccount(accountID string, postIDs []string) (map[string]bool, error) {
	likedMap := make(map[string]bool)
	if len(postIDs) == 0 {
		return likedMap, nil
	}

	var likes []models.BusinessPostLike
	err := r.db.Where("account_id = ? AND post_id IN ?", accountID, postIDs).Find(&likes).Error
	if err != nil {
		return likedMap, err
	}

	for _, like := range likes {
		likedMap[like.PostID] = true
	}
	return likedMap, nil
}

type BusinessVerificationInfo struct {
	IsVerified bool `gorm:"column:is_verified"`
	HasNIB     bool `gorm:"column:has_nib"`
}

func (r *repository) GetVerificationStatus(businessID string) (bool, bool, error) {
	var res BusinessVerificationInfo
	query := `
		SELECT 
			COALESCE(
				o.kyc_status = 'verified' OR EXISTS (
					SELECT 1 
					FROM business_has_tags bht 
					JOIN business_tags bt ON bt.tag_id = bht.tag_id 
					WHERE bht.business_id = b.business_id 
					  AND (LOWER(bt.name) LIKE '%verif%' OR LOWER(bt.name) LIKE '%(nib)%')
				),
				false
			) AS is_verified,
			COALESCE(b.nib IS NOT NULL AND TRIM(b.nib) != '', false) AS has_nib
		FROM businesses b
		LEFT JOIN owners o ON o.account_id = b.account_id
		WHERE b.business_id = ?
		LIMIT 1
	`
	if err := r.db.Raw(query, businessID).Scan(&res).Error; err != nil {
		return false, false, err
	}
	return res.IsVerified, res.HasNIB, nil
}

func (r *repository) BatchGetVerificationStatus(businessIDs []string) (map[string]BusinessVerificationInfo, error) {
	resMap := make(map[string]BusinessVerificationInfo)
	if len(businessIDs) == 0 {
		return resMap, nil
	}
	type Row struct {
		BusinessID string `gorm:"column:business_id"`
		IsVerified bool   `gorm:"column:is_verified"`
		HasNIB     bool   `gorm:"column:has_nib"`
	}
	var rows []Row
	query := `
		SELECT 
			b.business_id,
			COALESCE(
				o.kyc_status = 'verified' OR EXISTS (
					SELECT 1 
					FROM business_has_tags bht 
					JOIN business_tags bt ON bt.tag_id = bht.tag_id 
					WHERE bht.business_id = b.business_id 
					  AND (LOWER(bt.name) LIKE '%verif%' OR LOWER(bt.name) LIKE '%(nib)%')
				),
				false
			) AS is_verified,
			COALESCE(b.nib IS NOT NULL AND TRIM(b.nib) != '', false) AS has_nib
		FROM businesses b
		LEFT JOIN owners o ON o.account_id = b.account_id
		WHERE b.business_id IN (?)
	`
	if err := r.db.Raw(query, businessIDs).Scan(&rows).Error; err != nil {
		return resMap, err
	}
	for _, row := range rows {
		resMap[row.BusinessID] = BusinessVerificationInfo{
			IsVerified: row.IsVerified,
			HasNIB:     row.HasNIB,
		}
	}
	return resMap, nil
}

func (r *repository) GetSavedAndWishlistedBizIDs(accountID string) (SavedShopData, error) {
	data := SavedShopData{
		SavedBizMap:      make(map[string]time.Time),
		WishlistedBizMap: make(map[string]int),
		WishlistLatestAt: make(map[string]time.Time),
	}

	if accountID == "" {
		return data, nil
	}

	// 1. Query bookmarked businesses
	type bookmarkRow struct {
		BusinessID string    `gorm:"column:business_id"`
		CreatedAt  time.Time `gorm:"column:created_at"`
	}
	var bRows []bookmarkRow
	err := r.db.Table("business_bookmarks").
		Select("business_id, created_at").
		Where("account_id = ?", accountID).
		Find(&bRows).Error
	if err != nil {
		return data, err
	}
	for _, b := range bRows {
		data.SavedBizMap[b.BusinessID] = b.CreatedAt
	}

	// 2. Query wishlisted products and their business_id
	type wishlistRow struct {
		BusinessID       string    `gorm:"column:business_id"`
		Count            int       `gorm:"column:count"`
		LatestWishlistAt time.Time `gorm:"column:latest_wishlist_at"`
	}
	var wRows []wishlistRow
	err = r.db.Table("product_wishlists pw").
		Select("p.business_id, count(pw.product_id) as count, max(pw.created_at) as latest_wishlist_at").
		Joins("JOIN products p ON pw.product_id = p.product_id").
		Where("pw.account_id = ? AND p.deleted_at IS NULL", accountID).
		Group("p.business_id").
		Find(&wRows).Error
	if err != nil {
		return data, err
	}
	for _, w := range wRows {
		data.WishlistedBizMap[w.BusinessID] = w.Count
		data.WishlistLatestAt[w.BusinessID] = w.LatestWishlistAt
	}

	return data, nil
}

func (r *repository) GetBusinessesByIDs(businessIDs []string) ([]models.Business, error) {
	if len(businessIDs) == 0 {
		return []models.Business{}, nil
	}
	var businesses []models.Business
	err := r.db.
		Preload("Categories").
		Preload("Hours").
		Where("business_id IN ?", businessIDs).
		Find(&businesses).Error
	return businesses, err
}

