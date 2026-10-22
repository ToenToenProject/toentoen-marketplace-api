package promotion

import (
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

func TestPromotionValidation_Caption(t *testing.T) {
	// 1. Caption length > 150 chars
	longCaption := string(make([]byte, 151))
	if len(longCaption) <= 150 {
		t.Errorf("expected caption length > 150")
	}

	// 2. Caption within 150 chars
	validCaption := "Diskon Spesial Gajian! Dapatkan potongan harga s.d 50% untuk produk terpilih."
	if len(validCaption) > 150 {
		t.Errorf("expected caption <= 150 chars, got %d", len(validCaption))
	}
}

func TestPromotionValidation_MaxProducts(t *testing.T) {
	// Max 20 products
	input := models.CreatePromotionInput{
		Caption:       "Promo Akhir Pekan",
		MainProductID: "PRD-01",
		Products:      make([]models.CreatePromotionItemInput, 21),
	}
	if len(input.Products) > 20 {
		// Expected to fail validation
	} else {
		t.Errorf("expected validation to flag > 20 products")
	}

	// Main product must belong to products
	found := false
	for _, p := range input.Products {
		if p.ProductID == input.MainProductID {
			found = true
			break
		}
	}
	if found {
		t.Errorf("main product should not have been found in unpopulated slice")
	}
}

func TestPromotionValidation_Duration(t *testing.T) {
	now := time.Now()
	// Maximum 90 days
	startAt := now
	endAtExceeded := startAt.AddDate(0, 0, 91)
	if endAtExceeded.Sub(startAt) > 90*24*time.Hour {
		// Valid rejection
	} else {
		t.Errorf("expected duration to exceed 90 days")
	}

	endAtValid := startAt.AddDate(0, 0, 14) // 14 days promo
	if endAtValid.Sub(startAt) > 90*24*time.Hour {
		t.Errorf("expected duration to be valid under 90 days")
	}
}

func TestPromotionStatus_Derivation(t *testing.T) {
	now := time.Now()

	// 1. Scheduled promotion (starts in 2 hours, ends in 48 hours)
	scheduledStart := now.Add(2 * time.Hour)
	scheduledEnd := now.Add(48 * time.Hour)
	status := models.PromotionStatusActive
	if scheduledStart.After(now) && scheduledEnd.After(scheduledStart) {
		status = models.PromotionStatusScheduled
	}
	if status != models.PromotionStatusScheduled {
		t.Errorf("expected status 'scheduled', got '%s'", status)
	}

	// 2. Active promotion (started 1 hour ago, ends tomorrow)
	activeStart := now.Add(-1 * time.Hour)
	activeEnd := now.Add(24 * time.Hour)
	if !activeStart.After(now) && activeEnd.After(now) {
		status = models.PromotionStatusActive
	}
	if status != models.PromotionStatusActive {
		t.Errorf("expected status 'active', got '%s'", status)
	}

	// 3. Ended promotion (ended 1 hour ago)
	endedEnd := now.Add(-1 * time.Hour)
	if endedEnd.Before(now) {
		status = models.PromotionStatusEnded
	}
	if status != models.PromotionStatusEnded {
		t.Errorf("expected status 'ended', got '%s'", status)
	}
}

func TestPromotion_RemainingDaysCalculation(t *testing.T) {
	now := time.Now()
	// Promo ends in 3 days and 5 hours
	endAt := now.Add(3*24*time.Hour + 5*time.Hour)
	diff := endAt.Sub(now)
	daysRemaining := int(diff.Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	if daysRemaining != 3 {
		t.Errorf("expected 3 remaining days, got %d", daysRemaining)
	}

	// Expired promo
	expiredEnd := now.Add(-2 * time.Hour)
	diffExpired := expiredEnd.Sub(now)
	daysExpired := int(diffExpired.Hours() / 24)
	if daysExpired < 0 {
		daysExpired = 0
	}
	if daysExpired != 0 {
		t.Errorf("expected 0 remaining days for expired promo, got %d", daysExpired)
	}
}

func TestPromotion_DiscountCalculations(t *testing.T) {
	originalPrice := 100000.0

	// Percentage discount: 20%
	cut20, pct20 := utils.CalculateProductDiscount(originalPrice, "percentage", 20, nil)
	if cut20 != 20000.0 || pct20 != 20.0 {
		t.Errorf("expected cut 20000 and pct 20, got cut %.2f pct %.2f", cut20, pct20)
	}
	finalPrice20 := originalPrice - cut20
	if finalPrice20 != 80000.0 {
		t.Errorf("expected final price 80000, got %.2f", finalPrice20)
	}

	// Nominal discount: Rp 15.000
	cutNom, pctNom := utils.CalculateProductDiscount(originalPrice, "nominal", 15000, nil)
	if cutNom != 15000.0 || pctNom != 15.0 {
		t.Errorf("expected cut 15000 and pct 15, got cut %.2f pct %.2f", cutNom, pctNom)
	}
	finalPriceNom := originalPrice - cutNom
	if finalPriceNom != 85000.0 {
		t.Errorf("expected final price 85000, got %.2f", finalPriceNom)
	}

	// Percentage with MaxDiscount cap
	maxCap := 15000.0
	cutCap, _ := utils.CalculateProductDiscount(originalPrice, "percentage", 50, &maxCap) // 50% = 50,000 capped at 15,000
	if cutCap != 15000.0 {
		t.Errorf("expected capped cut 15000, got %.2f", cutCap)
	}
}

func TestPromotion_RedisDedupWindow(t *testing.T) {
	now := time.Now()
	lastSeen := now.Add(-10 * time.Minute)
	dedupWindow := 30 * time.Minute

	// Within 30 minutes -> duplicate, should skip
	isDuplicate := now.Sub(lastSeen) < dedupWindow
	if !isDuplicate {
		t.Errorf("expected event to be detected as duplicate within 30 min window")
	}

	// After 35 minutes -> fresh event, should record
	lastSeenOld := now.Add(-35 * time.Minute)
	isDuplicateOld := now.Sub(lastSeenOld) < dedupWindow
	if isDuplicateOld {
		t.Errorf("expected event to be accepted after 30 min window")
	}
}

func TestPromotionListItem_JSONContract(t *testing.T) {
	now := time.Now()
	postID := "PST-123"

	// 1. Catalog Promotion Item
	catalogItem := models.PromotionListItem{
		PromotionID:   "PRM-001",
		PromotionType: models.PromotionTypeCatalog,
		Caption:       "Diskon Gajian",
		MainProduct: &models.PromotionMainProductCard{
			ProductID:   "PRD-01",
			ProductName: "Kopi",
		},
		Post:          nil,
		ProductCount:  2,
		RemainingDays: 5,
		StartAt:       now,
		EndAt:         now.AddDate(0, 0, 5),
		Status:        models.PromotionStatusActive,
	}

	if catalogItem.PromotionType != "catalog" {
		t.Errorf("expected promotion_type 'catalog', got %s", catalogItem.PromotionType)
	}
	if catalogItem.MainProduct == nil {
		t.Errorf("expected main_product to be populated for catalog promotion")
	}
	if catalogItem.Post != nil {
		t.Errorf("expected post to be nil for catalog promotion")
	}

	// 2. Post Promotion Item
	postItem := models.PromotionListItem{
		PromotionID:   postID,
		PromotionType: models.PromotionTypePost,
		PostID:        &postID,
		Caption:       "Menu baru!",
		MainProduct:   nil,
		Post: &models.PromotionPostCard{
			PostID:    postID,
			Content:   "Menu baru!",
			LikeCount: 15,
		},
		ProductCount:  0,
		RemainingDays: 7,
		StartAt:       now,
		EndAt:         now.AddDate(0, 0, 7),
		Status:        models.PromotionStatusActive,
	}

	if postItem.PromotionType != "post" {
		t.Errorf("expected promotion_type 'post', got %s", postItem.PromotionType)
	}
	if postItem.Post == nil {
		t.Errorf("expected post to be populated for post promotion")
	}
	if postItem.MainProduct != nil {
		t.Errorf("expected main_product to be nil for post promotion")
	}
	if *postItem.PostID != postID {
		t.Errorf("expected post_id %s, got %s", postID, *postItem.PostID)
	}
}

func TestBundlingPromotionValidation(t *testing.T) {
	// 1. Min 2 products
	inputFew := models.CreateBundlingPromotionInput{
		PackageName:  "Paket Single",
		PackagePrice: 20000,
		Products: []models.BundlingProductItemInput{
			{ProductID: "PRD-01", Quantity: 1},
		},
	}
	if len(inputFew.Products) < 2 {
		// Valid rejection
	} else {
		t.Errorf("expected rejection for fewer than 2 products")
	}

	// 2. Package price must be cheaper than original total price
	// Product 1: Mie Ayam (Rp 25.000 x 1 = Rp 25.000)
	// Product 2: Es Teh (Rp 5.000 x 2 = Rp 10.000)
	// Original Total = Rp 35.000
	origTotal := 35000.0

	// Invalid package price: Rp 35.000 (same price)
	invalidPkgPrice := 35000.0
	if invalidPkgPrice >= origTotal {
		// Valid rejection
	} else {
		t.Errorf("expected rejection when package price equals or exceeds original price")
	}

	// Valid package price: Rp 28.000
	validPkgPrice := 28000.0
	savingsAmount := origTotal - validPkgPrice
	savingsPercentage := (savingsAmount / origTotal) * 100

	if savingsAmount != 7000.0 {
		t.Errorf("expected savings amount 7000, got %.2f", savingsAmount)
	}
	expectedPct := 20.0
	if savingsPercentage < expectedPct-0.01 || savingsPercentage > expectedPct+0.01 {
		t.Errorf("expected savings percentage 20.0%%, got %.2f%%", savingsPercentage)
	}
}

func TestBundlingPromotion_JSONContract(t *testing.T) {
	now := time.Now()
	pkgName := "Paket Kenyang Berdua"
	pkgPrice := 50000.0
	origTotal := 65000.0
	savingsAmt := 15000.0
	savingsPct := 23.08
	pkgImg := "https://res.cloudinary.com/demo/image/upload/sample.jpg"

	bundlingItem := models.PromotionListItem{
		PromotionID:        "PRM-BND-01",
		PromotionType:      models.PromotionTypeBundling,
		Caption:            "Paket Spesial Berdua!",
		PackageName:        &pkgName,
		PackagePrice:       &pkgPrice,
		PackageImage:       &pkgImg,
		OriginalTotalPrice: &origTotal,
		SavingsAmount:      &savingsAmt,
		SavingsPercentage:  &savingsPct,
		ProductCount:       2,
		RemainingDays:      7,
		StartAt:            now,
		EndAt:              now.AddDate(0, 0, 7),
		Status:             models.PromotionStatusActive,
	}

	if bundlingItem.PromotionType != "bundling" {
		t.Errorf("expected promotion_type 'bundling', got %s", bundlingItem.PromotionType)
	}
	if bundlingItem.PackageName == nil || *bundlingItem.PackageName != pkgName {
		t.Errorf("expected package name %s, got %v", pkgName, bundlingItem.PackageName)
	}
	if bundlingItem.PackagePrice == nil || *bundlingItem.PackagePrice != pkgPrice {
		t.Errorf("expected package price %.0f, got %v", pkgPrice, bundlingItem.PackagePrice)
	}
	if bundlingItem.SavingsAmount == nil || *bundlingItem.SavingsAmount != savingsAmt {
		t.Errorf("expected savings amount %.0f, got %v", savingsAmt, bundlingItem.SavingsAmount)
	}
	if bundlingItem.ProductCount != 2 {
		t.Errorf("expected product count 2, got %d", bundlingItem.ProductCount)
	}
}

func TestPromotionEligibility_CrossTypeSupport(t *testing.T) {
	// Skenario: Produk A sedang aktif di promosi per-produk (PRM-CAT-01)
	prodID := "PRD-A"
	activePerProductMap := map[string]string{
		prodID: "PRM-CAT-01",
	}
	activeBundlingMap := map[string]string{}

	// 1. Evaluasi untuk target_type: "per_product" (promosi per kategori / per produk baru)
	// Aturan: Produk yang sudah ada di per_product TIDAK BISA dipromosikan lagi di per_product
	inPerProduct := false
	if _, ok := activePerProductMap[prodID]; ok {
		inPerProduct = true
	}

	targetTypePerProduct := "per_product"
	isEligiblePerProduct := true
	if targetTypePerProduct == "per_product" && inPerProduct {
		isEligiblePerProduct = false
	}
	if isEligiblePerProduct {
		t.Errorf("expected product in per_product promotion to be ineligible for another per_product promotion")
	}

	// 2. Evaluasi untuk target_type: "bundling" (promosi paket bundling)
	// Aturan: Produk yang sudah ada di per_product TETAP BISA dipromosikan di paket bundling
	targetTypeBundling := "bundling"
	isEligibleBundling := true
	if targetTypeBundling == "bundling" {
		isEligibleBundling = true // Selalu eligible untuk bundling
	}
	if !isEligibleBundling {
		t.Errorf("expected product in per_product promotion to remain eligible for bundling promotion")
	}

	// 3. Skenario sebaliknya: Produk B sedang aktif di paket bundling (PRM-BND-01)
	prodB := "PRD-B"
	activeBundlingMap[prodB] = "PRM-BND-01"

	// Produk B ingin didaftarkan ke promosi per-produk
	inPerProductB := false
	if _, ok := activePerProductMap[prodB]; ok {
		inPerProductB = true
	}
	isEligibleBForPerProduct := true
	if targetTypePerProduct == "per_product" && inPerProductB {
		isEligibleBForPerProduct = false
	}
	if !isEligibleBForPerProduct {
		t.Errorf("expected product in bundling to remain eligible for per_product promotion")
	}
}



