package explore

import (
	"testing"
	"time"

	"toentoen-backend/internal/domain/shared/models"
	myMeili "toentoen-backend/pkg/meilisearch"
)

func TestHaversineDistance(t *testing.T) {
	// Monas Jakarta (-6.1754, 106.8272) to Bundaran HI (-6.1950, 106.8230) ~ 2.2 km
	lat1, lon1 := -6.1754, 106.8272
	lat2, lon2 := -6.1950, 106.8230

	dist := haversineDistance(lat1, lon1, lat2, lon2)
	if dist < 2.0 || dist > 2.5 {
		t.Errorf("Expected distance around 2.2 km, got %f km", dist)
	}
}

func TestScoringAndRankingCalculation(t *testing.T) {
	now := time.Now()

	// Post 1: Near, Recent, Good engagement
	post1 := models.BusinessPost{
		PostID:       "POST-1",
		BusinessID:   "BIZ-1",
		LikeCount:    50,
		CommentCount: 10,
		CreatedAt:    now.Add(-2 * time.Hour),
	}

	// Post 2: Far away, Old, Low engagement
	post2 := models.BusinessPost{
		PostID:       "POST-2",
		BusinessID:   "BIZ-2",
		LikeCount:    1,
		CommentCount: 0,
		CreatedAt:    now.Add(-72 * time.Hour),
	}

	hoursOld1 := now.Sub(post1.CreatedAt).Hours()
	recencyScore1 := 1.0 / (1.0 + (hoursOld1 / 36.0))

	hoursOld2 := now.Sub(post2.CreatedAt).Hours()
	recencyScore2 := 1.0 / (1.0 + (hoursOld2 / 36.0))

	if recencyScore1 <= recencyScore2 {
		t.Errorf("Recent post should have higher recency score: %f vs %f", recencyScore1, recencyScore2)
	}
}

func TestStoreTypeNormalizationAndLabels(t *testing.T) {
	cases := []struct {
		input         string
		expectedType  string
		expectedLabel string
	}{
		{"Rumah", models.StoreTypeRumah, "Rumah / Dapur Rumahan"},
		{"toko", models.StoreTypeToko, "Toko / Warung Kelontong"},
		{"KELILING", models.StoreTypeKeliling, "Pedagang Keliling"},
		{"gerobak", models.StoreTypeGerobak, "Gerobak / Kaki Lima"},
		{"Alun-Alun", models.StoreTypeAlunAlun, "Alun-Alun / Lapangan Terbuka"},
		{"pusat_belanja", models.StoreTypePusatBelanja, "Pusat Perbelanjaan / Mall"},
		{"unknown_type", "regular", "Regular"},
	}

	for _, c := range cases {
		norm := models.NormalizeStoreType(c.input)
		if norm != c.expectedType {
			t.Errorf("NormalizeStoreType(%q) = %q; expected %q", c.input, norm, c.expectedType)
		}
		label := models.GetStoreTypeLabel(norm)
		if label != c.expectedLabel {
			t.Errorf("GetStoreTypeLabel(%q) = %q; expected %q", norm, label, c.expectedLabel)
		}
	}
}

func TestSynonymExpansion(t *testing.T) {
	// Keyword "eskrim" should expand to contain "es krim" and "ice cream"
	expanded := myMeili.ExpandKeywordWithSynonyms("eskrim")
	foundEsKrim := false
	for _, term := range expanded {
		if term == "es krim" || term == "ice cream" {
			foundEsKrim = true
			break
		}
	}
	if !foundEsKrim {
		t.Errorf("ExpandKeywordWithSynonyms(\"eskrim\") did not include 'es krim': got %v", expanded)
	}

	// Keyword "ngopi" should expand to "kopi"
	kopiExpanded := myMeili.ExpandKeywordWithSynonyms("ngopi")
	foundKopi := false
	for _, term := range kopiExpanded {
		if term == "kopi" {
			foundKopi = true
			break
		}
	}
	if !foundKopi {
		t.Errorf("ExpandKeywordWithSynonyms(\"ngopi\") did not include 'kopi': got %v", kopiExpanded)
	}
}

func TestTieredRadiusRankingAndHashtagBooster(t *testing.T) {
	// Radius 5 km
	req := SearchRequest{
		Lat:      -6.2000,
		Lng:      106.8000,
		Radius:   5000, // 5 km
		Sort:     "nearest",
		Hashtags: []string{"pedas"},
	}

	docs := []MeiliBusinessDocument{
		{
			ID:       "BIZ-FAR-BOOSTED",
			Name:     "Warung Jauh Pedas",
			Distance: 8.0, // Outside radius (8 km)
			Hashtags: []string{"pedas"},
		},
		{
			ID:       "BIZ-NEAR-REGULAR",
			Name:     "Warung Dekat Biasa",
			Distance: 1.5, // Inside radius (1.5 km), NO hashtag
			Hashtags: []string{"manis"},
		},
		{
			ID:       "BIZ-NEAR-BOOSTED",
			Name:     "Warung Dekat Pedas",
			Distance: 3.0, // Inside radius (3.0 km), HAS hashtag "pedas"
			Hashtags: []string{"pedas"},
		},
		{
			ID:       "BIZ-FAR-REGULAR",
			Name:     "Warung Jauh Biasa",
			Distance: 12.0, // Outside radius (12 km)
			Hashtags: []string{},
		},
	}

	ranked, withinCount, outsideCount := rankBusinessDocuments(docs, req, 5.0)

	if withinCount != 2 {
		t.Errorf("Expected 2 items within radius, got %d", withinCount)
	}
	if outsideCount != 2 {
		t.Errorf("Expected 2 items outside radius, got %d", outsideCount)
	}

	// Rule 1: All Tier 1 (within radius) items MUST appear before Tier 2 (outside radius)
	// BIZ-FAR-BOOSTED has the hashtag, but it is OUTSIDE 5 km, so it MUST NOT jump over Tier 1 items!
	if !ranked[0].IsWithinRadius || !ranked[1].IsWithinRadius {
		t.Errorf("First two items must be within radius. Got item[0]: %s (within=%v), item[1]: %s (within=%v)",
			ranked[0].Name, ranked[0].IsWithinRadius, ranked[1].Name, ranked[1].IsWithinRadius)
	}

	// Rule 2: Inside Tier 1, BIZ-NEAR-BOOSTED (3.0 km with #pedas) should be boosted ABOVE BIZ-NEAR-REGULAR (1.5 km without hashtag)
	if ranked[0].ID != "BIZ-NEAR-BOOSTED" {
		t.Errorf("Expected BIZ-NEAR-BOOSTED to be ranked #1 due to hashtag booster, got %s", ranked[0].ID)
	}
	if ranked[1].ID != "BIZ-NEAR-REGULAR" {
		t.Errorf("Expected BIZ-NEAR-REGULAR to be ranked #2, got %s", ranked[1].ID)
	}

	// Rule 3: Inside Tier 2, BIZ-FAR-BOOSTED (8.0 km with #pedas) should be boosted ABOVE BIZ-FAR-REGULAR (12.0 km without hashtag)
	if ranked[2].ID != "BIZ-FAR-BOOSTED" {
		t.Errorf("Expected BIZ-FAR-BOOSTED to be ranked #3 in Tier 2, got %s", ranked[2].ID)
	}
	if ranked[3].ID != "BIZ-FAR-REGULAR" {
		t.Errorf("Expected BIZ-FAR-REGULAR to be ranked #4 in Tier 2, got %s", ranked[3].ID)
	}
}

func TestProductTieredRadiusAndPriceSort(t *testing.T) {
	req := SearchRequest{
		Lat:    -6.2000,
		Lng:    106.8000,
		Radius: 5000, // 5 km
		Sort:   "price_asc",
	}

	prods := []MeiliProductDocument{
		{
			ProductID:  "P-NEAR-EXPENSIVE",
			Distance:   2.0, // Inside radius
			FinalPrice: 35000,
		},
		{
			ProductID:  "P-NEAR-CHEAP",
			Distance:   4.0, // Inside radius
			FinalPrice: 15000,
		},
		{
			ProductID:  "P-FAR-SUPERCHEAP",
			Distance:   15.0, // Outside radius
			FinalPrice: 5000,
		},
	}

	ranked, withinCount, outsideCount := rankProductDocuments(prods, req, 5.0)

	if withinCount != 2 || outsideCount != 1 {
		t.Errorf("Expected 2 within and 1 outside, got within=%d, outside=%d", withinCount, outsideCount)
	}

	// Within radius tier: P-NEAR-CHEAP (15000) should come before P-NEAR-EXPENSIVE (35000)
	if ranked[0].ProductID != "P-NEAR-CHEAP" {
		t.Errorf("Expected ranked[0] to be P-NEAR-CHEAP, got %s", ranked[0].ProductID)
	}
	if ranked[1].ProductID != "P-NEAR-EXPENSIVE" {
		t.Errorf("Expected ranked[1] to be P-NEAR-EXPENSIVE, got %s", ranked[1].ProductID)
	}

	// Outside radius tier: P-FAR-SUPERCHEAP (5000) must stay in Tier 2 despite cheaper price
	if ranked[2].ProductID != "P-FAR-SUPERCHEAP" {
		t.Errorf("Expected ranked[2] to be P-FAR-SUPERCHEAP, got %s", ranked[2].ProductID)
	}
}

// Mock Repository for Service Testing
type mockExploreRepo struct {
	ExploreRepository
	businesses []MeiliBusinessDocument
	products   []MeiliProductDocument
}

func (m *mockExploreRepo) SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, SearchSummary, error) {
	cloned := make([]MeiliBusinessDocument, len(m.businesses))
	for i, b := range m.businesses {
		cloned[i] = b
		if b.MatchedProducts != nil {
			cloned[i].MatchedProducts = append([]MatchedProductPreview{}, b.MatchedProducts...)
		}
	}
	return cloned, SearchSummary{
		WithinRadiusCount:  int64(len(cloned)),
		OutsideRadiusCount: 0,
		TotalHits:          int64(len(cloned)),
	}, nil
}

func (m *mockExploreRepo) SearchProducts(req SearchRequest) ([]MeiliProductDocument, SearchSummary, error) {
	return m.products, SearchSummary{
		WithinRadiusCount:  int64(len(m.products)),
		OutsideRadiusCount: 0,
		TotalHits:          int64(len(m.products)),
	}, nil
}

func TestExploreServiceScopes(t *testing.T) {
	mockRepo := &mockExploreRepo{
		businesses: []MeiliBusinessDocument{
			{
				ID:   "BIZ-1",
				Name: "Toko Sinar",
				MatchedProducts: []MatchedProductPreview{
					{ProductID: "PROD-1", ProductName: "Kopi Susu"},
				},
			},
		},
		products: []MeiliProductDocument{
			{ProductID: "PROD-1", ProductName: "Kopi Susu", FinalPrice: 18000},
		},
	}

	svc := NewExploreService(mockRepo)

	// 1. Test Scope "store" (MatchedProducts should be stripped)
	resStore, err := svc.GetExploreBusinesses(SearchRequest{Scope: "store"})
	if err != nil {
		t.Fatalf("Scope store error: %v", err)
	}
	if resStore.Scope != "store" {
		t.Errorf("Expected scope 'store', got %q", resStore.Scope)
	}
	if len(resStore.Businesses) != 1 {
		t.Fatalf("Expected 1 business, got %d", len(resStore.Businesses))
	}
	if len(resStore.Businesses[0].MatchedProducts) != 0 {
		t.Errorf("Expected MatchedProducts to be stripped in scope 'store', got %v", resStore.Businesses[0].MatchedProducts)
	}

	// 2. Test Scope "all" (MatchedProducts preserved)
	resAll, err := svc.GetExploreBusinesses(SearchRequest{Scope: "all"})
	if err != nil {
		t.Fatalf("Scope all error: %v", err)
	}
	if resAll.Scope != "all" {
		t.Errorf("Expected scope 'all', got %q", resAll.Scope)
	}
	if len(resAll.Businesses[0].MatchedProducts) != 1 {
		t.Errorf("Expected MatchedProducts to be preserved in scope 'all', got %v", resAll.Businesses[0].MatchedProducts)
	}

	// 3. Test Scope "product" (Returns Products array)
	resProd, err := svc.GetExploreBusinesses(SearchRequest{Scope: "product"})
	if err != nil {
		t.Fatalf("Scope product error: %v", err)
	}
	if resProd.Scope != "product" {
		t.Errorf("Expected scope 'product', got %q", resProd.Scope)
	}
	if len(resProd.Products) != 1 || resProd.Products[0].ProductID != "PROD-1" {
		t.Errorf("Expected product PROD-1 in products array, got %v", resProd.Products)
	}
	if len(resProd.Businesses) != 0 {
		t.Errorf("Expected businesses array to be empty in scope 'product', got %d", len(resProd.Businesses))
	}
}

func TestBundlingPromotionDocumentMapping(t *testing.T) {
	pkgName := "Paket Duo Mantap"
	pkgImg := "https://example.com/duo.jpg"
	origPrice := 100000.0
	pkgPrice := 75000.0
	savingsAmount := 25000.0
	savingsPct := 25.0

	prm := &models.Promotion{
		PromotionID:        "PRM-TEST-BUNDLING",
		BusinessID:         "BUS-TEST",
		Caption:            "Paket Spesial",
		PromotionType:      models.PromotionTypeBundling,
		PackageName:        &pkgName,
		PackageImage:       &pkgImg,
		OriginalTotalPrice: &origPrice,
		PackagePrice:       &pkgPrice,
		SavingsAmount:      &savingsAmount,
		SavingsPercentage:  &savingsPct,
		CreatedAt:          time.Now(),
	}

	bizName := "Toko Enak"
	storeType := "toko"
	biz := &models.Business{
		BusinessID:   "BUS-TEST",
		BusinessName: &bizName,
		StoreType:    &storeType,
	}

	doc := myMeili.BuildBundlingDocument(nil, prm, biz)

	if doc.ID != "PRM-TEST-BUNDLING" || doc.ProductID != "PRM-TEST-BUNDLING" {
		t.Errorf("Expected ID and ProductID to be 'PRM-TEST-BUNDLING', got %s / %s", doc.ID, doc.ProductID)
	}
	if doc.ProductName != pkgName || doc.PackageName != pkgName {
		t.Errorf("Expected ProductName and PackageName to be %q, got %q / %q", pkgName, doc.ProductName, doc.PackageName)
	}
	if doc.Price != origPrice {
		t.Errorf("Expected Price to be %f, got %f", origPrice, doc.Price)
	}
	if doc.FinalPrice != pkgPrice {
		t.Errorf("Expected FinalPrice to be %f, got %f", pkgPrice, doc.FinalPrice)
	}
	if doc.DiscountPct != savingsPct {
		t.Errorf("Expected DiscountPct to be %f, got %f", savingsPct, doc.DiscountPct)
	}
	if !doc.IsDiscounted || !doc.HasPromo || !doc.IsPackage {
		t.Errorf("Expected IsDiscounted=true, HasPromo=true, IsPackage=true; got %v, %v, %v", doc.IsDiscounted, doc.HasPromo, doc.IsPackage)
	}
	if doc.PromotionType != "bundling" || doc.PromotionID != "PRM-TEST-BUNDLING" {
		t.Errorf("Expected PromotionType='bundling' and PromotionID='PRM-TEST-BUNDLING', got %s / %s", doc.PromotionType, doc.PromotionID)
	}
	if len(doc.ProductPreview) != 1 || doc.ProductPreview[0] != pkgImg {
		t.Errorf("Expected ProductPreview to contain package image, got %v", doc.ProductPreview)
	}
	if doc.BusinessName != bizName {
		t.Errorf("Expected BusinessName %q, got %q", bizName, doc.BusinessName)
	}
}

func TestProductDocumentActivePromotionMapping(t *testing.T) {
	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(24 * time.Hour)

	prod := &models.Product{
		ProductID:       "PROD-DISCOUNT-1",
		BusinessID:      "BUS-TEST",
		ProductName:     "Ayam Geprek Diskon",
		Price:           50000.0,
		Discount:        20.0,
		DiscountType:    "percentage",
		DiscountStartAt: &start,
		DiscountEndAt:   &end,
		StockType:       "in_stock",
		CreatedAt:       now,
	}

	doc := myMeili.BuildProductDocument(nil, prod, nil)

	if !doc.IsDiscounted || !doc.HasPromo {
		t.Errorf("Expected IsDiscounted=true and HasPromo=true, got %v / %v", doc.IsDiscounted, doc.HasPromo)
	}
	if doc.FinalPrice != 40000.0 {
		t.Errorf("Expected FinalPrice=40000.0, got %f", doc.FinalPrice)
	}
	if doc.DiscountPct != 20.0 {
		t.Errorf("Expected DiscountPct=20.0, got %f", doc.DiscountPct)
	}
	if doc.IsPackage {
		t.Errorf("Expected IsPackage=false for regular product, got true")
	}
}

