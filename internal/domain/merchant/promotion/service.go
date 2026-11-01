package promotion

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"
)

type PromotionService interface {
	GetEligibleProducts(requesterID, role, businessID, targetType string) ([]models.EligibleProductItem, error)
	CreatePromotion(requesterID, role, businessID string, input models.CreatePromotionInput) (*models.PromotionDetailResponse, error)
	CreateBundlingPromotion(requesterID, role, businessID string, input models.CreateBundlingPromotionInput) (*models.PromotionDetailResponse, error)
	GetPromotions(requesterID, role, businessID, tab, cursor string, limit int) (models.GetPromotionsResponse, error)
	GetPromotionDetail(requesterID, role, businessID, promotionID string) (*models.PromotionDetailResponse, error)
	GetCustomerPromotionDetail(promotionID string) (*models.PromotionDetailResponse, error)
	ExtendPromotion(requesterID, role, businessID, promotionID string, input models.ExtendPromotionInput) error
	StopPromotion(requesterID, role, businessID, promotionID string) error
	PushPromotionNotification(requesterID, role, businessID, promotionID string) (*models.PushPromotionResponse, error)

	RecordPromotionEvents(events []models.PromotionEvent) error
}

type promotionService struct {
	repo         PromotionRepository
	notifService notification.NotificationService
}

func NewPromotionService(repo PromotionRepository, notifService notification.NotificationService) PromotionService {
	return &promotionService{
		repo:         repo,
		notifService: notifService,
	}
}

func (s *promotionService) checkManageAccess(accountID, role, businessID string) error {
	if !s.repo.HasAccessToBusiness(accountID, role, businessID) {
		return errors.New("anda tidak memiliki akses ke bisnis ini")
	}
	if role == "business_admin" && !s.repo.HasPermission(accountID, businessID, models.PermManagePromotions) {
		return errors.New("anda tidak memiliki izin untuk mengelola promosi toko (manage_promotions)")
	}
	return nil
}

func (s *promotionService) GetEligibleProducts(requesterID, role, businessID, targetType string) ([]models.EligibleProductItem, error) {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return nil, err
	}
	return s.repo.GetEligibleProducts(businessID, targetType)
}

func (s *promotionService) CreatePromotion(
	requesterID, role, businessID string,
	input models.CreatePromotionInput,
) (*models.PromotionDetailResponse, error) {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return nil, err
	}

	caption := strings.TrimSpace(input.Caption)
	if caption == "" {
		caption = "Promo Spesial Toko"
	}
	if len(caption) > 150 {
		caption = caption[:150]
	}

	if len(input.Products) == 0 {
		return nil, errors.New("minimal 1 produk harus dipilih untuk promosi")
	}
	if len(input.Products) > 20 {
		return nil, errors.New("maksimal 20 produk yang dapat diikutsertakan dalam satu promosi")
	}

	now := time.Now()
	if input.DurationDays > 0 {
		if input.DurationDays != 1 && input.DurationDays != 3 && input.DurationDays != 7 {
			return nil, errors.New("pilihan periode promosi harus 1, 3, atau 7 hari")
		}
		input.StartAt = now
		input.EndAt = now.AddDate(0, 0, input.DurationDays)
	} else {
		// Berikan toleransi mundur untuk start_at agar tidak gagal karena latency atau perbedaan jam client
		if input.StartAt.IsZero() || input.StartAt.Before(now.Add(-5*time.Minute)) {
			input.StartAt = now
		}
		if input.EndAt.IsZero() || !input.EndAt.After(input.StartAt) {
			input.EndAt = input.StartAt.AddDate(0, 0, 7) // Auto-fallback default 7 hari jika end_at tidak valid
		}
	}

	totalDurationHours := input.EndAt.Sub(input.StartAt).Hours()
	if totalDurationHours > 90*24 {
		return nil, errors.New("durasi maksimal promosi adalah 90 hari")
	}

	// Cek apakah main_product_id ada di list produk, auto-fallback jika tidak ada
	mainProductFound := false
	productIDMap := make(map[string]bool)
	for _, p := range input.Products {
		cleanID := strings.TrimSpace(p.ProductID)
		if cleanID == "" {
			return nil, errors.New("product_id tidak boleh kosong")
		}
		if productIDMap[cleanID] {
			return nil, fmt.Errorf("produk dengan ID %s dimasukkan lebih dari satu kali", cleanID)
		}
		productIDMap[cleanID] = true
		if cleanID == input.MainProductID {
			mainProductFound = true
		}
	}
	if !mainProductFound && len(input.Products) > 0 {
		input.MainProductID = input.Products[0].ProductID
		mainProductFound = true
	}

	// Cek konflik: pastikan produk tidak sedang dalam promosi per-produk aktif/terjadwal lain
	activePromoMap, err := s.repo.GetActivePromotionProductIDs(businessID, models.PromotionTypePerProduct)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa status promosi produk: %w", err)
	}
	for pID := range productIDMap {
		if existingPromoID, conflict := activePromoMap[pID]; conflict {
			return nil, fmt.Errorf("produk %s sudah terdaftar dalam promosi per-produk aktif/terjadwal (%s)", pID, existingPromoID)
		}
	}

	// Ambil data produk dari DB untuk validasi dan snapshot harga
	eligibleProducts, err := s.repo.GetEligibleProducts(businessID, models.PromotionTypePerProduct)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat data produk: %w", err)
	}
	eligibleMap := make(map[string]models.EligibleProductItem)
	for _, ep := range eligibleProducts {
		eligibleMap[ep.ProductID] = ep
	}

	var mainProductItem *models.EligibleProductItem
	promotionID := utils.GenerateUUID("PRM")
	isActiveNow := true
	status := models.PromotionStatusActive
	var notifiedAt *time.Time = nil

	promoProducts := make([]models.PromotionProduct, 0, len(input.Products))
	productsToUpdate := make([]models.Product, 0)
	productIDsList := make([]string, 0, len(input.Products))

	for _, item := range input.Products {
		prodInfo, exists := eligibleMap[item.ProductID]
		if !exists {
			return nil, fmt.Errorf("produk %s tidak ditemukan atau tidak aktif di bisnis ini", item.ProductID)
		}
		if item.ProductID == input.MainProductID {
			mainProductItem = &prodInfo
		}
		productIDsList = append(productIDsList, item.ProductID)

		keepExisting := item.KeepExistingDiscount != nil && *item.KeepExistingDiscount
		// Jika merchant tidak set diskon baru dan produk memang punya diskon lama, default keepExisting
		if item.Discount == nil && prodInfo.HasExistingDiscount {
			keepExisting = true
		}

		var discSource string
		var finalDiscType string
		var finalDiscVal float64
		var finalMaxDisc *float64

		hadPrev := prodInfo.HasExistingDiscount
		var prevType *string
		var prevVal *float64
		var prevMax *float64
		var prevStart *time.Time
		var prevEnd *time.Time

		if hadPrev {
			tCopy := prodInfo.CurrentDiscountType
			prevType = &tCopy
			vCopy := prodInfo.CurrentDiscount
			prevVal = &vCopy
			prevMax = prodInfo.CurrentMaxDiscount
			prevStart = prodInfo.CurrentDiscountStartAt
			prevEnd = prodInfo.CurrentDiscountEndAt
		}

		if keepExisting {
			if !hadPrev {
				return nil, fmt.Errorf("produk %s tidak memiliki diskon sebelumnya untuk dipertahankan", item.ProductID)
			}
			discSource = models.PromotionDiscountSourceKept
			finalDiscType = prodInfo.CurrentDiscountType
			finalDiscVal = prodInfo.CurrentDiscount
			finalMaxDisc = prodInfo.CurrentMaxDiscount
		} else {
			// Merchant membuat diskon baru
			discSource = models.PromotionDiscountSourceNew

			// Auto-infer discount_type jika kosong
			if item.DiscountType == nil || strings.TrimSpace(*item.DiscountType) == "" {
				defaultType := "percentage"
				if item.Discount != nil && *item.Discount > 100 {
					defaultType = "nominal"
				}
				item.DiscountType = &defaultType
			}
			finalDiscType = strings.ToLower(strings.TrimSpace(*item.DiscountType))
			if finalDiscType != "percentage" && finalDiscType != "nominal" {
				finalDiscType = "percentage"
			}
			if item.Discount == nil || *item.Discount <= 0 {
				return nil, fmt.Errorf("nilai diskon (discount) wajib lebih dari 0 untuk produk %s", item.ProductID)
			}
			finalDiscVal = *item.Discount

			if finalDiscType == "percentage" && (finalDiscVal < 1 || finalDiscVal > 100) {
				return nil, fmt.Errorf("persentase diskon produk %s harus antara 1%% hingga 100%%", item.ProductID)
			}
			if finalDiscType == "nominal" && finalDiscVal >= prodInfo.Price {
				return nil, fmt.Errorf("nominal diskon produk %s tidak boleh sama atau melebihi harga produk (Rp %.0f)", item.ProductID, prodInfo.Price)
			}

			// Jika max_discount <= 0, ubah ke nil agar disimpan sebagai SQL NULL
			if item.MaxDiscount != nil && *item.MaxDiscount <= 0 {
				item.MaxDiscount = nil
			}
			finalMaxDisc = item.MaxDiscount
		}

		cut, _ := utils.CalculateProductDiscount(prodInfo.Price, finalDiscType, finalDiscVal, finalMaxDisc)
		snapshotFinal := prodInfo.Price - cut
		if snapshotFinal < 0 {
			snapshotFinal = 0
		}

		promoProducts = append(promoProducts, models.PromotionProduct{
			PromotionID:           promotionID,
			ProductID:             item.ProductID,
			DiscountSource:        discSource,
			DiscountType:          finalDiscType,
			Discount:              finalDiscVal,
			MaxDiscount:           finalMaxDisc,
			HadPreviousDiscount:   hadPrev,
			PrevDiscountType:      prevType,
			PrevDiscount:          prevVal,
			PrevMaxDiscount:       prevMax,
			PrevDiscountStartAt:   prevStart,
			PrevDiscountEndAt:     prevEnd,
			SnapshotOriginalPrice: prodInfo.Price,
			SnapshotFinalPrice:    snapshotFinal,
		})

		// Jika promosi langsung aktif sekarang, siapkan perubahan pada tabel products
		if isActiveNow {
			productsToUpdate = append(productsToUpdate, models.Product{
				ProductID:       item.ProductID,
				DiscountType:    finalDiscType,
				Discount:        finalDiscVal,
				MaxDiscount:     finalMaxDisc,
				DiscountStartAt: &input.StartAt,
				DiscountEndAt:   &input.EndAt,
			})
		}
	}

	promo := &models.Promotion{
		PromotionID:     promotionID,
		BusinessID:      businessID,
		Caption:         caption,
		MainProductID:   input.MainProductID,
		StartAt:         input.StartAt,
		EndAt:           input.EndAt,
		Status:          status,
		NotifiedAt:      notifiedAt,
		ImpressionCount: 0,
		ClickCount:      0,
		CreatedBy:       requesterID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Buat Auto-Post feed jika promosi langsung aktif
	var autoPost *models.BusinessPost
	if isActiveNow {
		images := []string{}
		if mainProductItem != nil && mainProductItem.ImageURL != "" {
			images = append(images, mainProductItem.ImageURL)
		}
		autoPost = &models.BusinessPost{
			PostID:         utils.GenerateUUID("POST"),
			BusinessID:     businessID,
			Content:        caption,
			Images:         images,
			IsPromotion:    true,
			PromotionID:    &promotionID,
			PromotionEndAt: &input.EndAt,
			LikeCount:      0,
			CommentCount:   0,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	if err := s.repo.CreatePromotion(promo, promoProducts, productsToUpdate, autoPost); err != nil {
		return nil, fmt.Errorf("gagal membuat promosi: %w", err)
	}

	// Promosi langsung aktif di katalog. Sinkronkan data ke Meilisearch.
	// Blast notifikasi dilakukan secara manual oleh merchant via endpoint push promotion.
	go meilisearch.SyncBusiness(database.DB, businessID)
	go meilisearch.SyncPromotion(database.DB, promotionID)

	return s.GetPromotionDetail(requesterID, role, businessID, promotionID)
}

func (s *promotionService) CreateBundlingPromotion(
	requesterID, role, businessID string,
	input models.CreateBundlingPromotionInput,
) (*models.PromotionDetailResponse, error) {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return nil, err
	}

	packageName := strings.TrimSpace(input.PackageName)
	if len(packageName) < 3 || len(packageName) > 100 {
		return nil, errors.New("nama paket bundling wajib diisi antara 3 hingga 100 karakter")
	}

	caption := strings.TrimSpace(input.Caption)
	if caption == "" {
		caption = fmt.Sprintf("Promo Paket Hemat: %s!", packageName)
	}
	if len(caption) > 150 {
		return nil, errors.New("caption promosi maksimal 150 karakter")
	}

	if input.PackagePrice <= 0 {
		return nil, errors.New("harga paket bundling harus lebih besar dari 0")
	}

	if len(input.Products) < 2 {
		return nil, errors.New("minimal 2 produk yang harus dipilih untuk paket bundling")
	}
	if len(input.Products) > 20 {
		return nil, errors.New("maksimal 20 produk yang dapat diikutsertakan dalam paket bundling")
	}

	// Deduplikasi produk ID & validasi kuantitas
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

	now := time.Now()
	if input.StartAt.Before(now.Add(-5 * time.Minute)) {
		input.StartAt = now
	}

	var endAt time.Time
	if input.EndAt != nil && !input.EndAt.IsZero() {
		endAt = *input.EndAt
	} else if input.DurationDays != nil && *input.DurationDays > 0 {
		endAt = input.StartAt.AddDate(0, 0, *input.DurationDays)
	} else {
		endAt = input.StartAt.AddDate(0, 0, 7) // Default 7 hari
	}

	if !endAt.After(input.StartAt) {
		return nil, errors.New("tanggal berakhir promosi harus setelah tanggal mulai")
	}

	if endAt.Sub(input.StartAt).Hours() > 90*24 {
		return nil, errors.New("durasi maksimal promosi adalah 90 hari")
	}

	// Ambil produk dari database
	products, err := s.repo.GetProductsByIDs(businessID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa produk: %w", err)
	}

	prodMap := make(map[string]models.Product)
	for _, p := range products {
		prodMap[p.ProductID] = p
	}

	var originalTotalPrice float64
	promotionID := utils.GenerateUUID("PRM")
	bundleItems := make([]models.PromotionBundleItem, 0, len(input.Products))

	for _, pid := range productIDs {
		prod, exists := prodMap[pid]
		if !exists {
			return nil, fmt.Errorf("produk dengan ID %s tidak ditemukan atau tidak aktif di bisnis ini", pid)
		}
		qty := qtyMap[pid]
		originalTotalPrice += prod.Price * float64(qty)

		bundleItems = append(bundleItems, models.PromotionBundleItem{
			PromotionID:     promotionID,
			ProductID:       pid,
			Quantity:        qty,
			ItemNormalPrice: prod.Price,
		})
	}

	// Validasi: harga paket harus lebih murah dari total harga normal produk
	if input.PackagePrice >= originalTotalPrice {
		return nil, fmt.Errorf("harga paket (Rp %.0f) harus lebih murah dari total harga normal produk (Rp %.0f)", input.PackagePrice, originalTotalPrice)
	}

	savingsAmount := originalTotalPrice - input.PackagePrice
	savingsPercentage := math.Round((savingsAmount/originalTotalPrice)*10000) / 100

	// Penentuan foto paket
	var packageImage *string
	if input.ImageURL != nil && strings.TrimSpace(*input.ImageURL) != "" {
		cleaned := strings.TrimSpace(*input.ImageURL)
		packageImage = &cleaned
	} else if firstProd, ok := prodMap[productIDs[0]]; ok && len(firstProd.ProductPreview) > 0 {
		img := firstProd.ProductPreview[0]
		packageImage = &img
	}

	isActiveNow := !input.StartAt.After(now)
	status := models.PromotionStatusScheduled
	var notifiedAt *time.Time
	if isActiveNow {
		status = models.PromotionStatusActive
		notifiedAt = &now
	}

	mainProductID := productIDs[0]
	promo := &models.Promotion{
		PromotionID:        promotionID,
		BusinessID:         businessID,
		Caption:            caption,
		MainProductID:      mainProductID,
		StartAt:            input.StartAt,
		EndAt:              endAt,
		Status:             status,
		NotifiedAt:         notifiedAt,
		CreatedBy:          requesterID,
		PromotionType:      models.PromotionTypeBundling,
		PackageName:        &packageName,
		PackagePrice:       &input.PackagePrice,
		PackageImage:       packageImage,
		OriginalTotalPrice: &originalTotalPrice,
		SavingsAmount:      &savingsAmount,
		SavingsPercentage:  &savingsPercentage,
	}

	// Buat auto-post di feed jika promosi langsung aktif
	var autoPost *models.BusinessPost
	if isActiveNow {
		postImages := make([]string, 0)
		if packageImage != nil && *packageImage != "" {
			postImages = append(postImages, *packageImage)
		}
		autoPost = &models.BusinessPost{
			PostID:         utils.GenerateUUID("PST"),
			BusinessID:     businessID,
			Content:        caption,
			Images:         postImages,
			IsPromotion:    true,
			PromotionID:    &promotionID,
			PromotionEndAt: &endAt,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	if err := s.repo.CreateBundlingPromotion(promo, bundleItems, autoPost); err != nil {
		return nil, fmt.Errorf("gagal membuat promosi bundling: %w", err)
	}

	// Jika langsung aktif, kirim notifikasi dan sync search
	if isActiveNow && s.notifService != nil {
		var mainProdModel *models.Product
		if firstProd, ok := prodMap[productIDs[0]]; ok {
			bundlePreview := []string{}
			if packageImage != nil && *packageImage != "" {
				bundlePreview = []string{*packageImage}
			} else {
				bundlePreview = firstProd.ProductPreview
			}
			mainProdModel = &models.Product{
				ProductID:      firstProd.ProductID,
				ProductName:    packageName,
				ProductPreview: bundlePreview,
			}
		}
		notification.NotifyPromotionCreated(database.DB, s.notifService, businessID, promo, mainProdModel, productIDs, requesterID)
		go meilisearch.SyncBusiness(database.DB, businessID)
	}
	if isActiveNow {
		go meilisearch.SyncPromotion(database.DB, promotionID)
	}

	return s.GetPromotionDetail(requesterID, role, businessID, promotionID)
}

func (s *promotionService) GetPromotions(
	requesterID, role, businessID, tab, cursor string,
	limit int,
) (models.GetPromotionsResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return models.GetPromotionsResponse{}, errors.New("anda tidak memiliki akses ke bisnis ini")
	}

	if limit <= 0 {
		limit = 10
	}

	summary, err := s.repo.GetPromotionSummary(businessID)
	if err != nil {
		return models.GetPromotionsResponse{}, err
	}

	// 1. Ambil katalog promosi
	promos, err := s.repo.GetPromotions(businessID, tab, "", 100)
	if err != nil {
		return models.GetPromotionsResponse{}, err
	}

	// 2. Ambil postingan promosi
	posts, _ := s.repo.GetPromotionalPosts(businessID, tab)

	now := time.Now()
	allItems := make([]models.PromotionListItem, 0, len(promos)+len(posts))

	// Format katalog promosi
	for _, p := range promos {
		remainingDays := 0
		if p.EndAt.After(now) {
			remainingDays = int(math.Ceil(p.EndAt.Sub(now).Hours() / 24.0))
		}

		var mainCard *models.PromotionMainProductCard
		pType := models.PromotionTypeCatalog
		prodCount := len(p.Products)

		if p.PromotionType == models.PromotionTypeBundling {
			pType = models.PromotionTypeBundling
			prodCount = len(p.BundleItems)
			bundleImg := ""
			if p.PackageImage != nil && *p.PackageImage != "" {
				bundleImg = *p.PackageImage
			} else if p.MainProduct != nil && len(p.MainProduct.ProductPreview) > 0 {
				bundleImg = p.MainProduct.ProductPreview[0]
			}
			var pkgPrice, origTotal, savAmt, savPct float64
			if p.PackagePrice != nil {
				pkgPrice = *p.PackagePrice
			}
			if p.OriginalTotalPrice != nil {
				origTotal = *p.OriginalTotalPrice
			}
			if p.SavingsAmount != nil {
				savAmt = *p.SavingsAmount
			}
			if p.SavingsPercentage != nil {
				savPct = *p.SavingsPercentage
			}
			pkgName := ""
			if p.PackageName != nil {
				pkgName = *p.PackageName
			}
			mainCard = &models.PromotionMainProductCard{
				ProductID:      p.MainProductID,
				ProductName:    pkgName,
				ImageURL:       bundleImg,
				OriginalPrice:  origTotal,
				FinalPrice:     pkgPrice,
				DiscountType:   "nominal",
				Discount:       savAmt,
				DiscountAmount: savAmt,
				DiscountPct:    savPct,
			}
		} else if p.MainProduct != nil {
			img := ""
			if len(p.MainProduct.ProductPreview) > 0 {
				img = p.MainProduct.ProductPreview[0]
			}
			cut, pct := utils.CalculateProductDiscount(p.MainProduct.Price, p.MainProduct.DiscountType, p.MainProduct.Discount, p.MainProduct.MaxDiscount)
			mainCard = &models.PromotionMainProductCard{
				ProductID:      p.MainProduct.ProductID,
				ProductName:    p.MainProduct.ProductName,
				ImageURL:       img,
				OriginalPrice:  p.MainProduct.Price,
				FinalPrice:     p.MainProduct.Price - cut,
				DiscountType:   p.MainProduct.DiscountType,
				Discount:       p.MainProduct.Discount,
				DiscountAmount: cut,
				DiscountPct:    pct,
			}
		}

		var bundleCount *int
		if p.PromotionType == models.PromotionTypeBundling {
			cnt := len(p.BundleItems)
			bundleCount = &cnt
		}

		notifSent := p.NotifiedAt != nil
		canPush := (p.Status == models.PromotionStatusActive && !notifSent)
		pTitle := p.Caption
		if p.PromotionType == models.PromotionTypeBundling && p.PackageName != nil && *p.PackageName != "" {
			pTitle = fmt.Sprintf("Promo Paket: %s", *p.PackageName)
		} else if p.MainProduct != nil && p.MainProduct.ProductName != "" {
			pTitle = fmt.Sprintf("Promo %s", p.MainProduct.ProductName)
		}

		allItems = append(allItems, models.PromotionListItem{
			ID:                 p.PromotionID,
			PromotionID:        p.PromotionID,
			Title:              pTitle,
			Description:        p.Caption,
			PromotionType:      pType,
			PostID:             p.PostID,
			Caption:            p.Caption,
			MainProduct:        mainCard,
			Post:               nil,
			PackageName:        p.PackageName,
			PackagePrice:       p.PackagePrice,
			PackageImage:       p.PackageImage,
			OriginalTotalPrice: p.OriginalTotalPrice,
			SavingsAmount:      p.SavingsAmount,
			SavingsPercentage:  p.SavingsPercentage,
			BundleItemCount:    bundleCount,
			ProductCount:       prodCount,
			RemainingDays:      remainingDays,
			StartAt:            p.StartAt,
			EndAt:              p.EndAt,
			Status:             p.Status,
			NotificationSent:   notifSent,
			CanPushNotification: canPush,
			ImpressionCount:    p.ImpressionCount,
			ClickCount:         p.ClickCount,
			CreatedAt:          p.CreatedAt,
		})
	}

	// Format postingan promosi
	for _, post := range posts {
		remainingDays := 0
		endAt := post.CreatedAt.AddDate(0, 0, 7)
		if post.PromotionEndAt != nil {
			endAt = *post.PromotionEndAt
		}
		if endAt.After(now) {
			remainingDays = int(math.Ceil(endAt.Sub(now).Hours() / 24.0))
		}

		status := models.PromotionStatusActive
		if !post.IsPromotion || endAt.Before(now) {
			status = models.PromotionStatusEnded
		}

		postCard := &models.PromotionPostCard{
			PostID:       post.PostID,
			Content:      post.Content,
			Images:       post.Images,
			VideoURL:     post.VideoURL,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
		}

		postIDVal := post.PostID
		allItems = append(allItems, models.PromotionListItem{
			ID:                  post.PostID,
			PromotionID:         post.PostID,
			Title:               post.Content,
			Description:         post.Content,
			PromotionType:       models.PromotionTypePost,
			PostID:              &postIDVal,
			Caption:             post.Content,
			MainProduct:         nil,
			Post:                postCard,
			ProductCount:        0,
			RemainingDays:       remainingDays,
			StartAt:             post.CreatedAt,
			EndAt:               endAt,
			Status:              status,
			NotificationSent:    true,
			CanPushNotification: false,
			ImpressionCount:     0,
			ClickCount:          post.LikeCount,
			CreatedAt:           post.CreatedAt,
		})
	}

	// Urutkan daftar gabungan
	switch tab {
	case "popular":
		sort.Slice(allItems, func(i, j int) bool {
			if allItems[i].ClickCount != allItems[j].ClickCount {
				return allItems[i].ClickCount > allItems[j].ClickCount
			}
			return allItems[i].CreatedAt.After(allItems[j].CreatedAt)
		})
	case "ended":
		sort.Slice(allItems, func(i, j int) bool {
			if !allItems[i].EndAt.Equal(allItems[j].EndAt) {
				return allItems[i].EndAt.After(allItems[j].EndAt)
			}
			return allItems[i].CreatedAt.After(allItems[j].CreatedAt)
		})
	default: // "active"
		sort.Slice(allItems, func(i, j int) bool {
			if !allItems[i].EndAt.Equal(allItems[j].EndAt) {
				return allItems[i].EndAt.Before(allItems[j].EndAt)
			}
			return allItems[i].CreatedAt.After(allItems[j].CreatedAt)
		})
	}

	// Cursor pagination
	startIndex := 0
	if cursor != "" {
		for idx, item := range allItems {
			if item.PromotionID == cursor {
				startIndex = idx + 1
				break
			}
		}
	}

	hasMore := false
	var pagedItems []models.PromotionListItem
	if startIndex < len(allItems) {
		endIndex := startIndex + limit
		if endIndex < len(allItems) {
			hasMore = true
			pagedItems = allItems[startIndex:endIndex]
		} else {
			pagedItems = allItems[startIndex:]
		}
	} else {
		pagedItems = []models.PromotionListItem{}
	}

	nextCursor := ""
	if hasMore && len(pagedItems) > 0 {
		nextCursor = pagedItems[len(pagedItems)-1].PromotionID
	}

	return models.GetPromotionsResponse{
		Summary:    summary,
		Promotions: pagedItems,
		Pagination: models.PaginationMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func formatRupiah(amount float64) string {
	str := fmt.Sprintf("%.0f", amount)
	n := len(str)
	if n <= 3 {
		return str
	}
	var res []byte
	rem := n % 3
	if rem > 0 {
		res = append(res, []byte(str[:rem])...)
	}
	for i := rem; i < n; i += 3 {
		if len(res) > 0 {
			res = append(res, '.')
		}
		res = append(res, []byte(str[i:i+3])...)
	}
	return string(res)
}

func mapBusinessToPromotionProfile(bus *models.Business) *models.PromotionBusinessProfile {
	if bus == nil {
		return nil
	}
	bizName := "Toko"
	if bus.BusinessName != nil && strings.TrimSpace(*bus.BusinessName) != "" {
		bizName = strings.TrimSpace(*bus.BusinessName)
	}
	logo := ""
	if bus.LogoURL != nil {
		logo = *bus.LogoURL
	}
	address := ""
	if bus.AddressDetail != nil {
		address = *bus.AddressDetail
	}
	isVerified := bus.IsVerified
	if !isVerified && bus.NIB != nil && *bus.NIB != "" {
		isVerified = true
	}
	return &models.PromotionBusinessProfile{
		ID:         bus.BusinessID,
		Name:       bizName,
		LogoURL:    logo,
		Address:    address,
		Rating:     bus.AverageRating,
		IsVerified: isVerified,
	}
}

func (s *promotionService) buildPromotionDetailResponse(promo *models.Promotion) *models.PromotionDetailResponse {
	now := time.Now()
	remainingDays := 0
	if promo.EndAt.After(now) {
		remainingDays = int(math.Ceil(promo.EndAt.Sub(now).Hours() / 24.0))
	}

	var postCard *models.PromotionPostCard
	if promo.Post != nil {
		postCard = &models.PromotionPostCard{
			PostID:       promo.Post.PostID,
			Content:      promo.Post.Content,
			Images:       promo.Post.Images,
			VideoURL:     promo.Post.VideoURL,
			LikeCount:    promo.Post.LikeCount,
			CommentCount: promo.Post.CommentCount,
		}
	}

	mainProdID := promo.MainProductID

	// Profil bisnis untuk tombol kunjungi toko
	var bizProfile *models.PromotionBusinessProfile
	bizName := "Toko"
	if promo.Business != nil {
		bizProfile = mapBusinessToPromotionProfile(promo.Business)
		if bizProfile != nil {
			bizName = bizProfile.Name
		}
	} else if promo.BusinessID != "" {
		biz, err := s.repo.GetBusinessByID(promo.BusinessID)
		if err == nil && biz != nil {
			bizProfile = mapBusinessToPromotionProfile(biz)
			if bizProfile != nil {
				bizName = bizProfile.Name
			}
		}
	}

	// Status notifikasi push
	notifSent := promo.NotifiedAt != nil
	notifStatus := "not_sent"
	if notifSent {
		notifStatus = "sent"
	}
	canPush := (promo.Status == models.PromotionStatusActive && !notifSent)
	notifInfo := &models.PromotionNotificationInfo{
		Status:  notifStatus,
		SentAt:  promo.NotifiedAt,
		CanPush: canPush,
	}

	if promo.PromotionType == models.PromotionTypeBundling {
		bundleItemDetails := make([]models.PromotionBundleItemDetail, 0, len(promo.BundleItems))
		for _, bi := range promo.BundleItems {
			name := ""
			img := ""
			normPrice := bi.ItemNormalPrice
			if bi.Product != nil {
				name = bi.Product.ProductName
				if len(bi.Product.ProductPreview) > 0 {
					img = bi.Product.ProductPreview[0]
				}
				if normPrice == 0 {
					normPrice = bi.Product.Price
				}
			}
			qty := bi.Quantity
			if qty <= 0 {
				qty = 1
			}
			bundleItemDetails = append(bundleItemDetails, models.PromotionBundleItemDetail{
				ProductID:           bi.ProductID,
				ProductName:         name,
				ImageURL:            img,
				Quantity:            qty,
				NormalPrice:         normPrice,
				SubtotalNormalPrice: normPrice * float64(qty),
			})
		}

		var discInfo *models.PromotionDiscountInfo
		if promo.SavingsAmount != nil {
			discInfo = &models.PromotionDiscountInfo{
				Type:  "nominal",
				Value: *promo.SavingsAmount,
			}
		}

		pkgName := ""
		if promo.PackageName != nil {
			pkgName = *promo.PackageName
		}
		title := fmt.Sprintf("%s: Promo Paket Hemat %s", bizName, pkgName)

		return &models.PromotionDetailResponse{
			PromotionID:        promo.PromotionID,
			ID:                 promo.PromotionID,
			Title:              title,
			Description:        promo.Caption,
			PromotionType:      models.PromotionTypeBundling,
			BusinessID:         promo.BusinessID,
			Caption:            promo.Caption,
			MainProductID:      &mainProdID,
			Discount:           discInfo,
			Notification:       notifInfo,
			Business:           bizProfile,
			ProductCount:       len(bundleItemDetails),
			PostID:             promo.PostID,
			Post:               postCard,
			PackageName:        promo.PackageName,
			PackagePrice:       promo.PackagePrice,
			PackageImage:       promo.PackageImage,
			OriginalTotalPrice: promo.OriginalTotalPrice,
			SavingsAmount:      promo.SavingsAmount,
			SavingsPercentage:  promo.SavingsPercentage,
			StartAt:            promo.StartAt,
			EndAt:              promo.EndAt,
			Status:             promo.Status,
			RemainingDays:      remainingDays,
			ImpressionCount:    promo.ImpressionCount,
			ClickCount:         promo.ClickCount,
			CreatedAt:          promo.CreatedAt,
			Products:           []models.PromotionProductDetailItem{},
			BundleItems:        bundleItemDetails,
		}
	}

	productDetails := make([]models.PromotionProductDetailItem, 0, len(promo.Products))
	var mainProdName string

	for _, pp := range promo.Products {
		name := ""
		img := ""
		origPrice := pp.SnapshotOriginalPrice
		if pp.Product != nil {
			name = pp.Product.ProductName
			if len(pp.Product.ProductPreview) > 0 {
				img = pp.Product.ProductPreview[0]
			}
			if origPrice == 0 {
				origPrice = pp.Product.Price
			}
		}

		cut, pct := utils.CalculateProductDiscount(origPrice, pp.DiscountType, pp.Discount, pp.MaxDiscount)
		finalPrice := origPrice - cut
		if finalPrice < 0 {
			finalPrice = 0
		}

		isMain := pp.ProductID == promo.MainProductID
		if isMain && mainProdName == "" {
			mainProdName = name
		}

		productDetails = append(productDetails, models.PromotionProductDetailItem{
			ProductID:      pp.ProductID,
			ID:             pp.ProductID,
			ProductName:    name,
			Name:           name,
			ImageURL:       img,
			OriginalPrice:  origPrice,
			Price:          finalPrice,
			FinalPrice:     finalPrice,
			DiscountSource: pp.DiscountSource,
			DiscountType:   pp.DiscountType,
			Discount:       pp.Discount,
			DiscountAmount: cut,
			DiscountPct:    pct,
			MaxDiscount:    pp.MaxDiscount,
			IsMainProduct:  isMain,
		})
	}

	if mainProdName == "" && len(productDetails) > 0 {
		mainProdName = productDetails[0].ProductName
	}

	var discInfo *models.PromotionDiscountInfo
	var discStr string
	if len(promo.Products) > 0 {
		dType := promo.Products[0].DiscountType
		dVal := promo.Products[0].Discount
		discInfo = &models.PromotionDiscountInfo{
			Type:  dType,
			Value: dVal,
		}
		if dType == "nominal" {
			discStr = fmt.Sprintf("Rp %s", formatRupiah(dVal))
		} else {
			discStr = fmt.Sprintf("%.0f%%", dVal)
		}
	}

	nOthers := len(productDetails) - 1
	var title string
	if mainProdName != "" {
		if nOthers > 0 {
			title = fmt.Sprintf("%s diskon %s untuk %s dan %d produk lainnya.", bizName, discStr, mainProdName, nOthers)
		} else {
			title = fmt.Sprintf("%s diskon %s untuk %s.", bizName, discStr, mainProdName)
		}
	} else if discStr != "" {
		title = fmt.Sprintf("%s memberikan diskon %s.", bizName, discStr)
	} else {
		title = fmt.Sprintf("Promosi dari %s", bizName)
	}

	return &models.PromotionDetailResponse{
		PromotionID:     promo.PromotionID,
		ID:              promo.PromotionID,
		Title:           title,
		Description:     promo.Caption,
		PromotionType:   models.PromotionTypeCatalog,
		BusinessID:      promo.BusinessID,
		Caption:         promo.Caption,
		MainProductID:   &mainProdID,
		Discount:        discInfo,
		Notification:    notifInfo,
		Business:        bizProfile,
		ProductCount:    len(productDetails),
		PostID:          promo.PostID,
		Post:            postCard,
		StartAt:         promo.StartAt,
		EndAt:           promo.EndAt,
		Status:          promo.Status,
		RemainingDays:   remainingDays,
		ImpressionCount: promo.ImpressionCount,
		ClickCount:      promo.ClickCount,
		CreatedAt:       promo.CreatedAt,
		Products:        productDetails,
		BundleItems:     []models.PromotionBundleItemDetail{},
	}
}

func (s *promotionService) buildPostPromotionDetailResponse(post *models.BusinessPost) *models.PromotionDetailResponse {
	now := time.Now()
	remainingDays := 0
	endAt := post.CreatedAt.AddDate(0, 0, 7)
	if post.PromotionEndAt != nil {
		endAt = *post.PromotionEndAt
	}
	if endAt.After(now) {
		remainingDays = int(math.Ceil(endAt.Sub(now).Hours() / 24.0))
	}

	status := models.PromotionStatusActive
	if !post.IsPromotion || endAt.Before(now) {
		status = models.PromotionStatusEnded
	}

	var bizProfile *models.PromotionBusinessProfile
	if post.BusinessID != "" {
		biz, err := s.repo.GetBusinessByID(post.BusinessID)
		if err == nil && biz != nil {
			bizProfile = mapBusinessToPromotionProfile(biz)
		}
	}

	postCard := &models.PromotionPostCard{
		PostID:       post.PostID,
		Content:      post.Content,
		Images:       post.Images,
		VideoURL:     post.VideoURL,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
	}

	postIDVal := post.PostID
	return &models.PromotionDetailResponse{
		PromotionID:     post.PostID,
		ID:              post.PostID,
		Title:           post.Content,
		Description:     post.Content,
		PromotionType:   models.PromotionTypePost,
		BusinessID:      post.BusinessID,
		Caption:         post.Content,
		MainProductID:   nil,
		PostID:          &postIDVal,
		Post:            postCard,
		Business:        bizProfile,
		ProductCount:    0,
		StartAt:         post.CreatedAt,
		EndAt:           endAt,
		Status:          status,
		RemainingDays:   remainingDays,
		ImpressionCount: 0,
		ClickCount:      post.LikeCount,
		CreatedAt:       post.CreatedAt,
		Products:        []models.PromotionProductDetailItem{},
	}
}

func (s *promotionService) PushPromotionNotification(
	requesterID, role, businessID, promotionID string,
) (*models.PushPromotionResponse, error) {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return nil, err
	}

	promo, err := s.repo.GetPromotionByID(businessID, promotionID)
	if err != nil || promo == nil {
		return nil, errors.New("promosi tidak ditemukan")
	}

	if promo.Status != models.PromotionStatusActive {
		return nil, fmt.Errorf("hanya promosi aktif yang dapat dikirimkan notifikasinya (status saat ini: %s)", promo.Status)
	}

	if promo.NotifiedAt != nil {
		return nil, errors.New("notifikasi untuk promosi ini sudah pernah dikirimkan")
	}

	now := time.Now()
	if err := s.repo.UpdatePromotionNotifiedAt(businessID, promotionID, now); err != nil {
		return nil, fmt.Errorf("gagal memperbarui status pengiriman notifikasi: %w", err)
	}

	promo.NotifiedAt = &now

	if s.notifService != nil {
		var mainProdModel *models.Product
		productIDsList := make([]string, 0, len(promo.Products))
		for _, p := range promo.Products {
			productIDsList = append(productIDsList, p.ProductID)
			if p.ProductID == promo.MainProductID && p.Product != nil {
				mainProdModel = p.Product
			}
		}
		if mainProdModel == nil && len(promo.Products) > 0 && promo.Products[0].Product != nil {
			mainProdModel = promo.Products[0].Product
		}
		notification.NotifyPromotionCreated(database.DB, s.notifService, businessID, promo, mainProdModel, productIDsList, requesterID)
	}

	return &models.PushPromotionResponse{
		Success:     true,
		Message:     "Notifikasi promosi berhasil dikirimkan ke pelanggan",
		PromotionID: promotionID,
		NotifiedAt:  now,
		SentAt:      now,
	}, nil
}

func (s *promotionService) GetPromotionDetail(requesterID, role, businessID, promotionID string) (*models.PromotionDetailResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("anda tidak memiliki akses ke bisnis ini")
	}

	if strings.HasPrefix(promotionID, "PST-") {
		post, err := s.repo.GetPostByID(businessID, promotionID)
		if err != nil || post == nil {
			return nil, errors.New("postingan promosi tidak ditemukan")
		}
		return s.buildPostPromotionDetailResponse(post), nil
	}

	promo, err := s.repo.GetPromotionByID(businessID, promotionID)
	if err != nil {
		return nil, errors.New("promosi tidak ditemukan")
	}

	return s.buildPromotionDetailResponse(promo), nil
}

func (s *promotionService) GetCustomerPromotionDetail(promotionID string) (*models.PromotionDetailResponse, error) {
	if strings.HasPrefix(promotionID, "PST-") {
		post, err := s.repo.GetPostByID("", promotionID)
		if err != nil || post == nil {
			return nil, errors.New("postingan promosi tidak ditemukan")
		}
		return s.buildPostPromotionDetailResponse(post), nil
	}

	promo, err := s.repo.GetPromotionByID("", promotionID)
	if err != nil {
		return nil, errors.New("promosi tidak ditemukan")
	}

	return s.buildPromotionDetailResponse(promo), nil
}

func (s *promotionService) ExtendPromotion(
	requesterID, role, businessID, promotionID string,
	input models.ExtendPromotionInput,
) error {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return err
	}

	if strings.HasPrefix(promotionID, "PST-") {
		post, err := s.repo.GetPostByID(businessID, promotionID)
		if err != nil || post == nil {
			return errors.New("postingan promosi tidak ditemukan")
		}

		now := time.Now()
		var newEndAt time.Time
		if input.AddDays != nil && *input.AddDays > 0 {
			baseTime := now
			if post.PromotionEndAt != nil && post.PromotionEndAt.After(now) {
				baseTime = *post.PromotionEndAt
			}
			newEndAt = baseTime.AddDate(0, 0, *input.AddDays)
		} else if input.EndAt != nil {
			newEndAt = *input.EndAt
		} else {
			return errors.New("wajib mengisi jumlah hari perpanjangan (add_days) atau tanggal berakhir baru (end_at)")
		}

		if post.PromotionEndAt != nil && !newEndAt.After(*post.PromotionEndAt) {
			return errors.New("tanggal berakhir baru harus lebih lama dari tanggal berakhir saat ini")
		}

		post.IsPromotion = true
		post.PromotionEndAt = &newEndAt
		post.UpdatedAt = now

		return s.repo.UpdatePostPromotion(post)
	}

	promo, err := s.repo.GetPromotionByID(businessID, promotionID)
	if err != nil {
		return errors.New("promosi tidak ditemukan")
	}

	if promo.Status != models.PromotionStatusActive && promo.Status != models.PromotionStatusScheduled {
		return fmt.Errorf("hanya promosi aktif atau terjadwal yang dapat diperpanjang (status saat ini: %s)", promo.Status)
	}

	var newEndAt time.Time
	if input.AddDays != nil && *input.AddDays > 0 {
		newEndAt = promo.EndAt.AddDate(0, 0, *input.AddDays)
	} else if input.EndAt != nil {
		newEndAt = *input.EndAt
	} else {
		return errors.New("wajib mengisi jumlah hari perpanjangan (add_days) atau tanggal berakhir baru (end_at)")
	}

	if !newEndAt.After(promo.EndAt) {
		return errors.New("tanggal berakhir baru harus lebih lama dari tanggal berakhir saat ini")
	}

	totalDurationHours := newEndAt.Sub(promo.StartAt).Hours()
	if totalDurationHours > 90*24 {
		return errors.New("total durasi promosi setelah diperpanjang tidak boleh melebihi 90 hari dari tanggal mulai")
	}

	productIDs := make([]string, 0, len(promo.Products))
	for _, p := range promo.Products {
		productIDs = append(productIDs, p.ProductID)
	}

	if err := s.repo.ExtendPromotion(businessID, promotionID, newEndAt, productIDs); err != nil {
		return fmt.Errorf("gagal memperpanjang promosi: %w", err)
	}

	go meilisearch.SyncBusiness(database.DB, businessID)
	go meilisearch.SyncPromotion(database.DB, promotionID)
	return nil
}

func (s *promotionService) StopPromotion(requesterID, role, businessID, promotionID string) error {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return err
	}

	if strings.HasPrefix(promotionID, "PST-") {
		post, err := s.repo.GetPostByID(businessID, promotionID)
		if err != nil || post == nil {
			return errors.New("postingan promosi tidak ditemukan")
		}

		if !post.IsPromotion {
			return errors.New("promosi postingan ini sudah tidak aktif")
		}

		now := time.Now()
		post.IsPromotion = false
		post.PromotionEndAt = &now
		post.UpdatedAt = now

		return s.repo.UpdatePostPromotion(post)
	}

	promo, err := s.repo.GetPromotionByID(businessID, promotionID)
	if err != nil {
		return errors.New("promosi tidak ditemukan")
	}

	if promo.Status != models.PromotionStatusActive && promo.Status != models.PromotionStatusScheduled {
		return fmt.Errorf("promosi ini sudah tidak aktif (status saat ini: %s)", promo.Status)
	}

	if err := s.repo.StopPromotion(businessID, promotionID, promo.Products, promo.PostID); err != nil {
		return fmt.Errorf("gagal menghentikan promosi: %w", err)
	}

	go meilisearch.SyncBusiness(database.DB, businessID)
	go meilisearch.SyncPromotion(database.DB, promotionID)
	return nil
}

func (s *promotionService) RecordPromotionEvents(events []models.PromotionEvent) error {
	return s.repo.RecordPromotionEvents(events)
}
