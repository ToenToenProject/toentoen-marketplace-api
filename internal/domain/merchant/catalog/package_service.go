package catalog

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"
)

type PackageService interface {
	CalculatePackagePrice(requesterID, role, businessID string, input models.CalculatePackageInput) (*models.CalculatePackageResponse, error)
	CreatePackage(requesterID, role, businessID string, input models.CreatePackageInput) (*models.PackageDetailResponse, error)
	UpdatePackage(requesterID, role, businessID, packageID string, input models.UpdatePackageInput) (*models.PackageDetailResponse, error)
	DeletePackage(requesterID, role, businessID, packageID string) error
	GetPackages(requesterID, role, businessID string) ([]models.PackageDetailResponse, error)
	GetPackageDetail(requesterID, role, businessID, packageID string) (*models.PackageDetailResponse, error)
}

type packageService struct {
	repo         PackageRepository
	notifService notification.NotificationService
}

func NewPackageService(repo PackageRepository, notifService notification.NotificationService) PackageService {
	return &packageService{repo, notifService}
}

func (s *packageService) checkAccess(requesterID, role, businessID, permKey string) error {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}
	if role == "business_admin" && permKey != "" && !s.repo.HasPermission(requesterID, businessID, permKey) {
		return fmt.Errorf("forbidden: Anda tidak memiliki izin untuk %s", permKey)
	}
	return nil
}

func (s *packageService) CalculatePackagePrice(
	requesterID, role, businessID string,
	input models.CalculatePackageInput,
) (*models.CalculatePackageResponse, error) {
	if err := s.checkAccess(requesterID, role, businessID, ""); err != nil {
		return nil, err
	}

	if len(input.Products) < 2 {
		return nil, errors.New("minimal 2 produk yang harus dipilih untuk membuat paket")
	}
	if len(input.Products) > 20 {
		return nil, errors.New("maksimal 20 produk yang dapat diikutsertakan dalam paket")
	}

	productIDs := make([]string, 0, len(input.Products))
	qtyMap := make(map[string]int)
	for _, p := range input.Products {
		cleanID := strings.TrimSpace(p.ProductID)
		if cleanID == "" {
			return nil, errors.New("product_id tidak boleh kosong")
		}
		if _, exists := qtyMap[cleanID]; exists {
			return nil, fmt.Errorf("produk %s dimasukkan lebih dari satu kali, gunakan kuantitas untuk menambah porsi", cleanID)
		}
		qty := p.Quantity
		if qty <= 0 {
			qty = 1
		}
		qtyMap[cleanID] = qty
		productIDs = append(productIDs, cleanID)
	}

	products, err := s.repo.GetProductsByIDs(businessID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data produk: %w", err)
	}

	prodMap := make(map[string]models.Product)
	for _, p := range products {
		prodMap[p.ProductID] = p
	}

	now := time.Now()
	var totalNormalPrice, totalEffectivePrice float64
	items := make([]models.CalculatePackageItemDetail, 0, len(productIDs))

	for _, pid := range productIDs {
		prod, exists := prodMap[pid]
		if !exists {
			return nil, fmt.Errorf("produk dengan ID %s tidak ditemukan atau tidak aktif di bisnis ini", pid)
		}
		qty := qtyMap[pid]
		normalPrice := prod.Price
		subtotalNormal := normalPrice * float64(qty)
		totalNormalPrice += subtotalNormal

		effectivePrice := normalPrice
		hasExistingDiscount := false
		var discountType string
		var discountValue float64

		if utils.IsDiscountActive(prod.Discount, prod.DiscountStartAt, prod.DiscountEndAt, now) {
			cut, pct := utils.CalculateProductDiscount(prod.Price, prod.DiscountType, prod.Discount, prod.MaxDiscount)
			if cut > 0 {
				hasExistingDiscount = true
				effectivePrice = normalPrice - cut
				discountType = prod.DiscountType
				if prod.DiscountType == "percentage" {
					discountValue = pct
				} else {
					discountValue = cut
				}
			}
		}

		subtotalEffective := effectivePrice * float64(qty)
		totalEffectivePrice += subtotalEffective

		imgURL := ""
		if len(prod.ProductPreview) > 0 {
			imgURL = prod.ProductPreview[0]
		}

		items = append(items, models.CalculatePackageItemDetail{
			ProductID:              pid,
			ProductName:            prod.ProductName,
			ImageURL:               imgURL,
			Quantity:               qty,
			NormalPrice:            normalPrice,
			SubtotalNormalPrice:    subtotalNormal,
			HasExistingDiscount:    hasExistingDiscount,
			DiscountType:           discountType,
			DiscountValue:          discountValue,
			EffectivePrice:         effectivePrice,
			SubtotalEffectivePrice: subtotalEffective,
		})
	}

	potentialSavings := totalNormalPrice - totalEffectivePrice
	if potentialSavings < 0 {
		potentialSavings = 0
	}

	var packageImageURL string
	var packagePreview []string
	for _, pid := range productIDs {
		if p, ok := prodMap[pid]; ok && len(p.ProductPreview) > 0 {
			packageImageURL = p.ProductPreview[0]
			packagePreview = p.ProductPreview
			break
		}
	}

	return &models.CalculatePackageResponse{
		TotalNormalPrice:    totalNormalPrice,
		TotalEffectivePrice: totalEffectivePrice,
		DefaultPackagePrice: totalEffectivePrice,
		PotentialSavings:    potentialSavings,
		PackageImageURL:     packageImageURL,
		PackagePreview:      packagePreview,
		Items:               items,
	}, nil
}

func (s *packageService) CreatePackage(
	requesterID, role, businessID string,
	input models.CreatePackageInput,
) (*models.PackageDetailResponse, error) {
	if err := s.checkAccess(requesterID, role, businessID, models.PermAddProduct); err != nil {
		return nil, err
	}

	packageName := strings.TrimSpace(input.PackageName)
	if len(packageName) < 3 || len(packageName) > 100 {
		return nil, errors.New("nama paket wajib diisi antara 3 hingga 100 karakter")
	}

	if len(input.Products) < 2 {
		return nil, errors.New("minimal 2 produk yang harus dipilih untuk membuat paket")
	}
	if len(input.Products) > 20 {
		return nil, errors.New("maksimal 20 produk yang dapat diikutsertakan dalam paket")
	}

	if input.ProductGroupID != nil && *input.ProductGroupID != "" {
		if !s.repo.CheckProductGroupExists(businessID, *input.ProductGroupID) {
			return nil, errors.New("product_group_id tidak valid atau tidak ditemukan")
		}
	}

	productIDs := make([]string, 0, len(input.Products))
	qtyMap := make(map[string]int)
	for _, p := range input.Products {
		cleanID := strings.TrimSpace(p.ProductID)
		if cleanID == "" {
			return nil, errors.New("product_id tidak boleh kosong")
		}
		if _, exists := qtyMap[cleanID]; exists {
			return nil, fmt.Errorf("produk %s dimasukkan lebih dari satu kali", cleanID)
		}
		qty := p.Quantity
		if qty <= 0 {
			qty = 1
		}
		qtyMap[cleanID] = qty
		productIDs = append(productIDs, cleanID)
	}

	products, err := s.repo.GetProductsByIDs(businessID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data produk: %w", err)
	}

	prodMap := make(map[string]models.Product)
	for _, p := range products {
		prodMap[p.ProductID] = p
	}

	now := time.Now()
	var totalNormalPrice, totalEffectivePrice float64
	packageID := utils.GenerateUUID("PRD")
	packageItems := make([]models.ProductPackageItem, 0, len(productIDs))

	for _, pid := range productIDs {
		prod, exists := prodMap[pid]
		if !exists {
			return nil, fmt.Errorf("produk dengan ID %s tidak ditemukan atau tidak aktif di bisnis ini", pid)
		}
		qty := qtyMap[pid]
		normalPrice := prod.Price
		totalNormalPrice += normalPrice * float64(qty)

		effectivePrice := normalPrice
		hasExistingDiscount := false
		if utils.IsDiscountActive(prod.Discount, prod.DiscountStartAt, prod.DiscountEndAt, now) {
			cut, _ := utils.CalculateProductDiscount(prod.Price, prod.DiscountType, prod.Discount, prod.MaxDiscount)
			if cut > 0 {
				hasExistingDiscount = true
				effectivePrice = normalPrice - cut
			}
		}

		totalEffectivePrice += effectivePrice * float64(qty)

		packageItems = append(packageItems, models.ProductPackageItem{
			PackageProductID:   packageID,
			ItemProductID:      pid,
			Quantity:           qty,
			ItemNormalPrice:    normalPrice,
			ItemEffectivePrice: effectivePrice,
			HasDiscount:        hasExistingDiscount,
		})
	}

	if input.PackagePrice <= 0 {
		return nil, errors.New("harga paket harus lebih besar dari 0")
	}

	// Validasi: harga paket tidak boleh melebihi total harga produk setelah diskon
	if input.PackagePrice > totalEffectivePrice {
		return nil, fmt.Errorf("harga paket (Rp %.0f) tidak boleh melebihi total harga produk setelah diskon (Rp %.0f)", input.PackagePrice, totalEffectivePrice)
	}

	// Hitung penghematan paket
	savingsAmount := totalEffectivePrice - input.PackagePrice
	savingsPercentage := 0.0
	if totalEffectivePrice > 0 && savingsAmount > 0 {
		savingsPercentage = math.Round((savingsAmount/totalEffectivePrice)*10000) / 100
	}

	// Gambar paket: otomatis mengambil foto dari item pertama dari produk yang dipilih
	var previews []string
	for _, pid := range productIDs {
		if p, ok := prodMap[pid]; ok && len(p.ProductPreview) > 0 {
			previews = p.ProductPreview
			break
		}
	}
	// Jika merchant secara eksplisit menyertakan custom preview, gunakan sebagai override
	if len(input.ProductPreview) > 0 {
		previews = input.ProductPreview
	}

	stockType := strings.TrimSpace(input.StockType)
	if stockType == "" {
		stockType = "in_stock"
	}

	pkgProduct := models.Product{
		ProductID:                packageID,
		BusinessID:               businessID,
		ProductGroupID:           input.ProductGroupID,
		ProductName:              packageName,
		ProductOrigin:            "domestic",
		ProductDescription:       strings.TrimSpace(input.ProductDescription),
		ProductPreview:           previews,
		Price:                    input.PackagePrice,
		StockType:                stockType,
		IsPackage:                true,
		OriginalPackagePrice:     &totalNormalPrice,
		BaseEffectivePrice:       &totalEffectivePrice,
		PackageSavingsAmount:     &savingsAmount,
		PackageSavingsPercentage: &savingsPercentage,
		IsActive:                 true,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if err := s.repo.CreatePackage(&pkgProduct, packageItems); err != nil {
		return nil, fmt.Errorf("gagal menyimpan paket produk: %w", err)
	}

	go meilisearch.SyncBusiness(database.DB, businessID)

	return s.GetPackageDetail(requesterID, role, businessID, packageID)
}

func (s *packageService) UpdatePackage(
	requesterID, role, businessID, packageID string,
	input models.UpdatePackageInput,
) (*models.PackageDetailResponse, error) {
	if err := s.checkAccess(requesterID, role, businessID, models.PermEditProduct); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetPackageDetail(businessID, packageID)
	if err != nil {
		return nil, errors.New("paket produk tidak ditemukan")
	}

	if input.PackageName != nil {
		cleanName := strings.TrimSpace(*input.PackageName)
		if len(cleanName) < 3 || len(cleanName) > 100 {
			return nil, errors.New("nama paket wajib diisi antara 3 hingga 100 karakter")
		}
		existing.ProductName = cleanName
	}

	if input.ProductDescription != nil {
		existing.ProductDescription = strings.TrimSpace(*input.ProductDescription)
	}

	if input.ProductGroupID != nil {
		if *input.ProductGroupID != "" && !s.repo.CheckProductGroupExists(businessID, *input.ProductGroupID) {
			return nil, errors.New("product_group_id tidak valid")
		}
		existing.ProductGroupID = input.ProductGroupID
	}

	if input.ProductPreview != nil {
		existing.ProductPreview = *input.ProductPreview
	}

	if input.StockType != nil && strings.TrimSpace(*input.StockType) != "" {
		existing.StockType = strings.TrimSpace(*input.StockType)
	}

	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	now := time.Now()
	existing.UpdatedAt = now

	// Jika produk komponen diupdate
	var newPackageItems []models.ProductPackageItem
	if input.Products != nil && len(*input.Products) > 0 {
		prodInputs := *input.Products
		if len(prodInputs) < 2 {
			return nil, errors.New("minimal 2 produk yang harus dipilih untuk paket")
		}
		if len(prodInputs) > 20 {
			return nil, errors.New("maksimal 20 produk yang dapat diikutsertakan dalam paket")
		}

		productIDs := make([]string, 0, len(prodInputs))
		qtyMap := make(map[string]int)
		for _, p := range prodInputs {
			cleanID := strings.TrimSpace(p.ProductID)
			if cleanID == "" {
				return nil, errors.New("product_id tidak boleh kosong")
			}
			if _, exists := qtyMap[cleanID]; exists {
				return nil, fmt.Errorf("produk %s dimasukkan lebih dari satu kali", cleanID)
			}
			qty := p.Quantity
			if qty <= 0 {
				qty = 1
			}
			qtyMap[cleanID] = qty
			productIDs = append(productIDs, cleanID)
		}

		products, err := s.repo.GetProductsByIDs(businessID, productIDs)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil data produk: %w", err)
		}

		prodMap := make(map[string]models.Product)
		for _, p := range products {
			prodMap[p.ProductID] = p
		}

		var totalNormalPrice, totalEffectivePrice float64
		newPackageItems = make([]models.ProductPackageItem, 0, len(productIDs))

		for _, pid := range productIDs {
			prod, exists := prodMap[pid]
			if !exists {
				return nil, fmt.Errorf("produk dengan ID %s tidak ditemukan atau tidak aktif di bisnis ini", pid)
			}
			qty := qtyMap[pid]
			normalPrice := prod.Price
			totalNormalPrice += normalPrice * float64(qty)

			effectivePrice := normalPrice
			hasExistingDiscount := false
			if utils.IsDiscountActive(prod.Discount, prod.DiscountStartAt, prod.DiscountEndAt, now) {
				cut, _ := utils.CalculateProductDiscount(prod.Price, prod.DiscountType, prod.Discount, prod.MaxDiscount)
				if cut > 0 {
					hasExistingDiscount = true
					effectivePrice = normalPrice - cut
				}
			}

			totalEffectivePrice += effectivePrice * float64(qty)

			newPackageItems = append(newPackageItems, models.ProductPackageItem{
				PackageProductID:   packageID,
				ItemProductID:      pid,
				Quantity:           qty,
				ItemNormalPrice:    normalPrice,
				ItemEffectivePrice: effectivePrice,
				HasDiscount:        hasExistingDiscount,
			})
		}

		existing.OriginalPackagePrice = &totalNormalPrice
		existing.BaseEffectivePrice = &totalEffectivePrice

		// Jika merchant tidak mengirimkan custom ProductPreview, sinkronkan gambar ke produk pertama yang baru
		if input.ProductPreview == nil {
			for _, pid := range productIDs {
				if p, ok := prodMap[pid]; ok && len(p.ProductPreview) > 0 {
					existing.ProductPreview = p.ProductPreview
					break
				}
			}
		}
	}

	// Update harga paket
	if input.PackagePrice != nil && *input.PackagePrice > 0 {
		existing.Price = *input.PackagePrice
	}

	baseEffective := existing.Price
	if existing.BaseEffectivePrice != nil && *existing.BaseEffectivePrice > 0 {
		baseEffective = *existing.BaseEffectivePrice
	}

	if existing.Price > baseEffective {
		return nil, fmt.Errorf("harga paket (Rp %.0f) tidak boleh melebihi total harga produk setelah diskon (Rp %.0f)", existing.Price, baseEffective)
	}

	savingsAmount := baseEffective - existing.Price
	savingsPercentage := 0.0
	if baseEffective > 0 && savingsAmount > 0 {
		savingsPercentage = math.Round((savingsAmount/baseEffective)*10000) / 100
	}
	existing.PackageSavingsAmount = &savingsAmount
	existing.PackageSavingsPercentage = &savingsPercentage

	if err := s.repo.UpdatePackage(existing, newPackageItems); err != nil {
		return nil, fmt.Errorf("gagal memperbarui paket produk: %w", err)
	}

	go meilisearch.SyncBusiness(database.DB, businessID)

	return s.GetPackageDetail(requesterID, role, businessID, packageID)
}

func (s *packageService) DeletePackage(requesterID, role, businessID, packageID string) error {
	if err := s.checkAccess(requesterID, role, businessID, models.PermDeleteProduct); err != nil {
		return err
	}

	if err := s.repo.DeletePackage(businessID, packageID); err != nil {
		return err
	}

	go meilisearch.SyncBusiness(database.DB, businessID)
	return nil
}

func (s *packageService) GetPackages(requesterID, role, businessID string) ([]models.PackageDetailResponse, error) {
	if err := s.checkAccess(requesterID, role, businessID, ""); err != nil {
		return nil, err
	}

	pkgs, err := s.repo.GetPackages(businessID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar paket: %w", err)
	}

	responses := make([]models.PackageDetailResponse, 0, len(pkgs))
	for _, p := range pkgs {
		responses = append(responses, formatPackageDetail(&p))
	}

	return responses, nil
}

func (s *packageService) GetPackageDetail(requesterID, role, businessID, packageID string) (*models.PackageDetailResponse, error) {
	if err := s.checkAccess(requesterID, role, businessID, ""); err != nil {
		return nil, err
	}

	pkg, err := s.repo.GetPackageDetail(businessID, packageID)
	if err != nil {
		return nil, errors.New("paket produk tidak ditemukan")
	}

	res := formatPackageDetail(pkg)
	return &res, nil
}

func formatPackageDetail(p *models.Product) models.PackageDetailResponse {
	var origPrice, baseEff, savAmt, savPct float64
	if p.OriginalPackagePrice != nil {
		origPrice = *p.OriginalPackagePrice
	}
	if p.BaseEffectivePrice != nil {
		baseEff = *p.BaseEffectivePrice
	}
	if p.PackageSavingsAmount != nil {
		savAmt = *p.PackageSavingsAmount
	}
	if p.PackageSavingsPercentage != nil {
		savPct = *p.PackageSavingsPercentage
	}

	items := make([]models.PackageItemDetailResponse, 0, len(p.PackageItems))
	for _, pi := range p.PackageItems {
		name := ""
		imgURL := ""
		stockType := "in_stock"
		isActive := true

		if pi.ItemProduct != nil {
			name = pi.ItemProduct.ProductName
			stockType = pi.ItemProduct.StockType
			isActive = pi.ItemProduct.IsActive
			if len(pi.ItemProduct.ProductPreview) > 0 {
				imgURL = pi.ItemProduct.ProductPreview[0]
			}
		}

		items = append(items, models.PackageItemDetailResponse{
			ProductID:              pi.ItemProductID,
			ProductName:            name,
			ImageURL:               imgURL,
			Quantity:               pi.Quantity,
			NormalPrice:            pi.ItemNormalPrice,
			EffectivePrice:         pi.ItemEffectivePrice,
			SubtotalEffectivePrice: pi.ItemEffectivePrice * float64(pi.Quantity),
			StockType:              stockType,
			IsActive:               isActive,
		})
	}

	return models.PackageDetailResponse{
		ProductID:                p.ProductID,
		PackageID:                p.ProductID,
		BusinessID:               p.BusinessID,
		PackageName:              p.ProductName,
		ProductDescription:       p.ProductDescription,
		ProductGroupID:           p.ProductGroupID,
		ProductPreview:           p.ProductPreview,
		PackagePrice:             p.Price,
		OriginalPackagePrice:     origPrice,
		BaseEffectivePrice:       baseEff,
		SavingsAmount:            savAmt,
		SavingsPercentage:        savPct,
		IsDiscounted:             savAmt > 0,
		StockType:                p.StockType,
		IsActive:                 p.IsActive,
		ItemsCount:               len(items),
		Items:                    items,
		CreatedAt:                p.CreatedAt,
		UpdatedAt:                p.UpdatedAt,
	}
}
