package database

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	hasPromotionTypeMu        sync.RWMutex
	hasPromotionTypeCol       bool
	hasPromotionTypeLastCheck time.Time

	hasBundleItemsMu        sync.RWMutex
	hasBundleItemsTab       bool
	hasBundleItemsLastCheck time.Time

	hasSignalCategoriesMu        sync.RWMutex
	hasSignalCategoriesTab       bool
	hasSignalCategoriesLastCheck time.Time
)

// HasPromotionTypeColumn memeriksa apakah kolom promotion_type sudah ada di tabel promotions.
// Hasil di-cache selama 30 detik untuk efisiensi performa query.
func HasPromotionTypeColumn(db *gorm.DB) bool {
	if db == nil {
		return false
	}

	hasPromotionTypeMu.RLock()
	if time.Since(hasPromotionTypeLastCheck) < 30*time.Second {
		val := hasPromotionTypeCol
		hasPromotionTypeMu.RUnlock()
		return val
	}
	hasPromotionTypeMu.RUnlock()

	hasPromotionTypeMu.Lock()
	defer hasPromotionTypeMu.Unlock()
	if time.Since(hasPromotionTypeLastCheck) < 30*time.Second {
		return hasPromotionTypeCol
	}

	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_schema = 'public' 
			  AND table_name = 'promotions' 
			  AND column_name = 'promotion_type'
		)
	`).Scan(&exists).Error

	hasPromotionTypeCol = (err == nil && exists)
	hasPromotionTypeLastCheck = time.Now()
	return hasPromotionTypeCol
}

// HasBundleItemsTable memeriksa apakah tabel promotion_bundle_items sudah ada di database.
func HasBundleItemsTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}

	hasBundleItemsMu.RLock()
	if time.Since(hasBundleItemsLastCheck) < 30*time.Second {
		val := hasBundleItemsTab
		hasBundleItemsMu.RUnlock()
		return val
	}
	hasBundleItemsMu.RUnlock()

	hasBundleItemsMu.Lock()
	defer hasBundleItemsMu.Unlock()
	if time.Since(hasBundleItemsLastCheck) < 30*time.Second {
		return hasBundleItemsTab
	}

	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' 
			  AND table_name = 'promotion_bundle_items'
		)
	`).Scan(&exists).Error

	hasBundleItemsTab = (err == nil && exists)
	hasBundleItemsLastCheck = time.Now()
	return hasBundleItemsTab
}

// HasSignalCategoriesTable memeriksa apakah tabel signal_categories sudah ada di database.
func HasSignalCategoriesTable(db *gorm.DB) bool {
	if db == nil {
		return false
	}

	hasSignalCategoriesMu.RLock()
	if time.Since(hasSignalCategoriesLastCheck) < 30*time.Second {
		val := hasSignalCategoriesTab
		hasSignalCategoriesMu.RUnlock()
		return val
	}
	hasSignalCategoriesMu.RUnlock()

	hasSignalCategoriesMu.Lock()
	defer hasSignalCategoriesMu.Unlock()
	if time.Since(hasSignalCategoriesLastCheck) < 30*time.Second {
		return hasSignalCategoriesTab
	}

	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' 
			  AND table_name = 'signal_categories'
		)
	`).Scan(&exists).Error

	hasSignalCategoriesTab = (err == nil && exists)
	hasSignalCategoriesLastCheck = time.Now()
	return hasSignalCategoriesTab
}
