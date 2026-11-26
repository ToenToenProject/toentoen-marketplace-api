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

// HasPromotionTypeColumn checks whether column promotion_type exists in the promotions table.
// Results are cached for 30 seconds to minimize redundant information_schema lookups.
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

// HasBundleItemsTable checks whether table promotion_bundle_items exists in the database.
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

// HasSignalCategoriesTable checks whether table signal_categories exists in the database.
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
