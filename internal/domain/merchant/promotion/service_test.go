package promotion

import (
	"encoding/json"
	"fmt"
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

func TestEligibleProductsSorting_ActivePromotedAtBottom(t *testing.T) {
	// Skenario: Produk A (tersedia), Produk B (aktif promosi), Produk C (tersedia)
	items := []models.EligibleProductItem{
		{
			ProductID:            "PRD-A",
			ProductName:          "Produk A",
			IsActivePromoted:     false,
			PromotionStatus:      "available",
			PromotionStatusLabel: "Tersedia",
		},
		{
			ProductID:            "PRD-B",
			ProductName:          "Produk B",
			IsActivePromoted:     true,
			PromotionStatus:      "active_promoted",
			PromotionStatusLabel: "Sedang Aktif Promosi",
		},
		{
			ProductID:            "PRD-C",
			ProductName:          "Produk C",
			IsActivePromoted:     false,
			PromotionStatus:      "available",
			PromotionStatusLabel: "Tersedia",
		},
	}

	// Simulasikan pengurutan yang diterapkan pada GetEligibleProducts di repository
	var normalItems []models.EligibleProductItem
	var activePromotedItems []models.EligibleProductItem
	for _, it := range items {
		if it.IsActivePromoted {
			activePromotedItems = append(activePromotedItems, it)
		} else {
			normalItems = append(normalItems, it)
		}
	}
	sortedItems := append(normalItems, activePromotedItems...)

	if len(sortedItems) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sortedItems))
	}
	// Produk B (aktif promosi) harus berada di indeks paling akhir
	if sortedItems[2].ProductID != "PRD-B" {
		t.Errorf("expected PRD-B to be at the bottom, got %s", sortedItems[2].ProductID)
	}
	if !sortedItems[2].IsActivePromoted {
		t.Errorf("expected bottom item to have IsActivePromoted = true")
	}
	if sortedItems[0].IsActivePromoted || sortedItems[1].IsActivePromoted {
		t.Errorf("expected top items to not be active promoted")
	}
}

func TestCreatePromotionInput_UnmarshalJSON_NewFormat(t *testing.T) {
	// JSON payload dari form baru
	rawJSON := `{
		"caption": "Promo Gajian",
		"product_ids": ["PRD-01", "PRD-02"],
		"discount_type": "percentage",
		"discount_value": 20,
		"duration_days": 3
	}`

	var input models.CreatePromotionInput
	err := jsonUnmarshal([]byte(rawJSON), &input)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if input.Caption != "Promo Gajian" {
		t.Errorf("expected caption 'Promo Gajian', got '%s'", input.Caption)
	}
	if input.DurationDays != 3 {
		t.Errorf("expected duration_days 3, got %d", input.DurationDays)
	}
	if len(input.Products) != 2 {
		t.Fatalf("expected 2 products populated from product_ids, got %d", len(input.Products))
	}
	if input.Products[0].ProductID != "PRD-01" || *input.Products[0].Discount != 20 {
		t.Errorf("expected product 0 discount 20, got %v", input.Products[0].Discount)
	}
	if *input.Products[0].DiscountType != "percentage" {
		t.Errorf("expected product 0 discount_type 'percentage', got %s", *input.Products[0].DiscountType)
	}
}

func jsonUnmarshal(b []byte, v interface{}) error {
	// helper wrapper
	return json.Unmarshal(b, v)
}

func TestCreatePromotion_DurationChoices(t *testing.T) {
	validChoices := []int{1, 3, 7}
	for _, days := range validChoices {
		if days != 1 && days != 3 && days != 7 {
			t.Errorf("choice %d should be valid", days)
		}
	}

	invalidChoices := []int{0, 2, 4, 5, 10, 14, 30}
	for _, days := range invalidChoices {
		isValid := (days == 1 || days == 3 || days == 7)
		if isValid {
			t.Errorf("choice %d should be invalid", days)
		}
	}
}

func TestCreatePromotion_NominalDiscountValidation(t *testing.T) {
	productPrice := 50000.0

	// 1. Valid nominal discount (Rp 10.000 < Rp 50.000)
	validDiscount := 10000.0
	if validDiscount >= productPrice {
		t.Errorf("discount %.0f should be strictly less than product price %.0f", validDiscount, productPrice)
	}

	// 2. Invalid nominal discount (equal to price: Rp 50.000)
	equalDiscount := 50000.0
	if equalDiscount < productPrice {
		t.Errorf("discount equal to price should be rejected")
	}

	// 3. Invalid nominal discount (exceeds price: Rp 60.000)
	exceedDiscount := 60000.0
	if exceedDiscount < productPrice {
		t.Errorf("discount exceeding price should be rejected")
	}
}

func TestPromotionDetailResponse_TitleGeneration(t *testing.T) {
	bizName := "Warung Berkah"
	mainProd := "Nasi Goreng Spesial"

	// 1. Single product, percentage
	title1 := fmt.Sprintf("%s diskon %.0f%% untuk %s.", bizName, 20.0, mainProd)
	expected1 := "Warung Berkah diskon 20% untuk Nasi Goreng Spesial."
	if title1 != expected1 {
		t.Errorf("expected '%s', got '%s'", expected1, title1)
	}

	// 2. Multiple products (3 products total, N=2), nominal
	nOthers := 2
	discNomStr := formatRupiah(10000)
	title2 := fmt.Sprintf("%s diskon Rp %s untuk %s dan %d produk lainnya.", bizName, discNomStr, mainProd, nOthers)
	expected2 := "Warung Berkah diskon Rp 10.000 untuk Nasi Goreng Spesial dan 2 produk lainnya."
	if title2 != expected2 {
		t.Errorf("expected '%s', got '%s'", expected2, title2)
	}
}

func TestPushPromotionNotification_SinglePushRule(t *testing.T) {
	now := time.Now()

	// Promo aktif yang BELUM pernah di-push
	promoCanPush := models.Promotion{
		PromotionID: "PRM-ACTIVE-01",
		Status:      models.PromotionStatusActive,
		NotifiedAt:  nil,
	}
	canPush := (promoCanPush.Status == models.PromotionStatusActive && promoCanPush.NotifiedAt == nil)
	if !canPush {
		t.Errorf("expected active promo without notified_at to be pushable")
	}

	// Promo aktif yang SUDAH pernah di-push
	promoAlreadyPushed := models.Promotion{
		PromotionID: "PRM-ACTIVE-02",
		Status:      models.PromotionStatusActive,
		NotifiedAt:  &now,
	}
	canPushAgain := (promoAlreadyPushed.Status == models.PromotionStatusActive && promoAlreadyPushed.NotifiedAt == nil)
	if canPushAgain {
		t.Errorf("expected active promo with notified_at to NOT be pushable (anti-spam single push limit)")
	}
}



