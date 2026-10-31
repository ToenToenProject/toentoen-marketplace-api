package catalog

import (
	"testing"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

type mockPackageRepo struct {
	hasAccess  bool
	hasPerm    bool
	products   map[string]models.Product
	packages   map[string]*models.Product
	pkgItems   map[string][]models.ProductPackageItem
}

func newMockPackageRepo() *mockPackageRepo {
	return &mockPackageRepo{
		hasAccess: true,
		hasPerm:   true,
		products:  make(map[string]models.Product),
		packages:  make(map[string]*models.Product),
		pkgItems:  make(map[string][]models.ProductPackageItem),
	}
}

func (m *mockPackageRepo) HasAccessToBusiness(accountID, role, businessID string) bool {
	return m.hasAccess
}

func (m *mockPackageRepo) HasPermission(accountID, businessID, permKey string) bool {
	return m.hasPerm
}

func (m *mockPackageRepo) CheckProductGroupExists(businessID, productGroupID string) bool {
	return true
}

func (m *mockPackageRepo) GetProductsByIDs(businessID string, productIDs []string) ([]models.Product, error) {
	var res []models.Product
	for _, id := range productIDs {
		if p, ok := m.products[id]; ok {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockPackageRepo) CreatePackage(pkgProduct *models.Product, items []models.ProductPackageItem) error {
	m.packages[pkgProduct.ProductID] = pkgProduct
	m.pkgItems[pkgProduct.ProductID] = items
	return nil
}

func (m *mockPackageRepo) UpdatePackage(pkgProduct *models.Product, items []models.ProductPackageItem) error {
	m.packages[pkgProduct.ProductID] = pkgProduct
	if len(items) > 0 {
		m.pkgItems[pkgProduct.ProductID] = items
	}
	return nil
}

func (m *mockPackageRepo) DeletePackage(businessID, productID string) error {
	delete(m.packages, productID)
	delete(m.pkgItems, productID)
	return nil
}

func (m *mockPackageRepo) GetPackages(businessID string) ([]models.Product, error) {
	var res []models.Product
	for _, p := range m.packages {
		copyP := *p
		copyP.PackageItems = m.pkgItems[p.ProductID]
		res = append(res, copyP)
	}
	return res, nil
}

func (m *mockPackageRepo) GetPackageDetail(businessID, productID string) (*models.Product, error) {
	p, ok := m.packages[productID]
	if !ok {
		return nil, gormErrNotFound()
	}
	copyP := *p
	copyP.PackageItems = m.pkgItems[productID]
	return &copyP, nil
}

func gormErrNotFound() error {
	return nil // Handled as error in service
}

func TestCalculatePackagePrice(t *testing.T) {
	repo := newMockPackageRepo()
	svc := NewPackageService(repo, nil)

	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(24 * time.Hour)

	// Product A: 12.000 dengan diskon nominal 2.000 (harga efektif 10.000)
	repo.products["PRD-A"] = models.Product{
		ProductID:       "PRD-A",
		BusinessID:      "BUS-1",
		ProductName:     "Product A",
		Price:           12000,
		Discount:        2000,
		DiscountType:    "nominal",
		DiscountStartAt: &start,
		DiscountEndAt:   &end,
		IsActive:        true,
	}

	// Product B: 10.000 tanpa diskon (harga efektif 10.000)
	repo.products["PRD-B"] = models.Product{
		ProductID:   "PRD-B",
		BusinessID:  "BUS-1",
		ProductName: "Product B",
		Price:       10000,
		IsActive:    true,
	}

	// 1. Validasi minimal 2 produk
	_, errFew := svc.CalculatePackagePrice("ACC-1", "owner", "BUS-1", models.CalculatePackageInput{
		Products: []models.PackageProductItemInput{
			{ProductID: "PRD-A", Quantity: 1},
		},
	})
	if errFew == nil {
		t.Errorf("expected error for less than 2 products")
	}

	// 2. Kalkulasi benar
	res, err := svc.CalculatePackagePrice("ACC-1", "owner", "BUS-1", models.CalculatePackageInput{
		Products: []models.PackageProductItemInput{
			{ProductID: "PRD-A", Quantity: 1},
			{ProductID: "PRD-B", Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Total normal: 12.000 + 10.000 = 22.000
	if res.TotalNormalPrice != 22000 {
		t.Errorf("expected TotalNormalPrice 22000, got %v", res.TotalNormalPrice)
	}

	// Total efektif: 10.000 + 10.000 = 20.000
	if res.TotalEffectivePrice != 20000 {
		t.Errorf("expected TotalEffectivePrice 20000, got %v", res.TotalEffectivePrice)
	}

	if res.DefaultPackagePrice != 20000 {
		t.Errorf("expected DefaultPackagePrice 20000, got %v", res.DefaultPackagePrice)
	}

	if res.PotentialSavings != 2000 {
		t.Errorf("expected PotentialSavings 2000, got %v", res.PotentialSavings)
	}
}

func TestCreatePackagePricingRules(t *testing.T) {
	repo := newMockPackageRepo()
	svc := NewPackageService(repo, nil)

	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(24 * time.Hour)

	repo.products["PRD-A"] = models.Product{
		ProductID:       "PRD-A",
		BusinessID:      "BUS-1",
		ProductName:     "Product A",
		Price:           12000,
		Discount:        2000,
		DiscountType:    "nominal",
		DiscountStartAt: &start,
		DiscountEndAt:   &end,
		IsActive:        true,
	}
	repo.products["PRD-B"] = models.Product{
		ProductID:   "PRD-B",
		BusinessID:  "BUS-1",
		ProductName: "Product B",
		Price:       10000,
		IsActive:    true,
	}

	// Skenario 1: Harga paket melebihi total efektif (20.001 > 20.000) -> DITOLAK
	_, errOver := svc.CreatePackage("ACC-1", "owner", "BUS-1", models.CreatePackageInput{
		PackageName:  "Paket Mahal",
		PackagePrice: 21000,
		Products: []models.PackageProductItemInput{
			{ProductID: "PRD-A", Quantity: 1},
			{ProductID: "PRD-B", Quantity: 1},
		},
	})
	if errOver == nil {
		t.Errorf("expected rejection when package price exceeds total effective price")
	}

	// Skenario 2: Lanjut dengan harga total efektif (20.000) -> SUKSES
	pkgDef, errDef := svc.CreatePackage("ACC-1", "owner", "BUS-1", models.CreatePackageInput{
		PackageName:  "Paket Normal 20rb",
		PackagePrice: 20000,
		Products: []models.PackageProductItemInput{
			{ProductID: "PRD-A", Quantity: 1},
			{ProductID: "PRD-B", Quantity: 1},
		},
	})
	if errDef != nil {
		t.Fatalf("unexpected error: %v", errDef)
	}
	if pkgDef.PackagePrice != 20000 {
		t.Errorf("expected price 20000, got %v", pkgDef.PackagePrice)
	}
	if pkgDef.SavingsAmount != 0 {
		t.Errorf("expected savings 0, got %v", pkgDef.SavingsAmount)
	}

	// Skenario 3: Merchant memberi diskon tambahan menjadi 10.000 (diskon 50%) -> SUKSES
	pkgDiscounted, errDisc := svc.CreatePackage("ACC-1", "owner", "BUS-1", models.CreatePackageInput{
		PackageName:  "Paket Murah 10rb",
		PackagePrice: 10000,
		Products: []models.PackageProductItemInput{
			{ProductID: "PRD-A", Quantity: 1},
			{ProductID: "PRD-B", Quantity: 1},
		},
	})
	if errDisc != nil {
		t.Fatalf("unexpected error: %v", errDisc)
	}
	if pkgDiscounted.PackagePrice != 10000 {
		t.Errorf("expected price 10000, got %v", pkgDiscounted.PackagePrice)
	}
	if pkgDiscounted.SavingsAmount != 10000 {
		t.Errorf("expected savings 10000, got %v", pkgDiscounted.SavingsAmount)
	}
	if pkgDiscounted.SavingsPercentage != 50.0 {
		t.Errorf("expected savings 50.0%%, got %v", pkgDiscounted.SavingsPercentage)
	}
	if !pkgDiscounted.IsDiscounted {
		t.Errorf("expected is_discounted to be true")
	}
}
