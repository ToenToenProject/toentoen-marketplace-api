package promotion

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

type PromotionService interface {
	GetEligibleProducts(requesterID, role, businessID string) ([]models.EligibleProductItem, error)
	CreatePromotion(requesterID, role, businessID string, input models.CreatePromotionInput) (*models.PromotionDetailResponse, error)
	GetPromotions(requesterID, role, businessID, tab, cursor string, limit int) (models.GetPromotionsResponse, error)
	GetPromotionDetail(requesterID, role, businessID, promotionID string) (*models.PromotionDetailResponse, error)
	GetCustomerPromotionDetail(promotionID string) (*models.PromotionDetailResponse, error)
	ExtendPromotion(requesterID, role, businessID, promotionID string, input models.ExtendPromotionInput) error
	StopPromotion(requesterID, role, businessID, promotionID string) error

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

func (s *promotionService) GetEligibleProducts(requesterID, role, businessID string) ([]models.EligibleProductItem, error) {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return nil, err
	}
	return s.repo.GetEligibleProducts(businessID)
}

func (s *promotionService) CreatePromotion(
	requesterID, role, businessID string,
	input models.CreatePromotionInput,
) (*models.PromotionDetailResponse, error) {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return nil, err
	}

	caption := strings.TrimSpace(input.Caption)
	if len(caption) < 3 || len(caption) > 150 {
		return nil, errors.New("caption promosi wajib diisi antara 3 hingga 150 karakter")
	}

	if len(input.Products) == 0 {
		return nil, errors.New("minimal 1 produk harus dipilih untuk promosi")
	}
	if len(input.Products) > 20 {
		return nil, errors.New("maksimal 20 produk yang dapat diikutsertakan dalam satu promosi")
	}

	now := time.Now()
	// Berikan toleransi mundur 5 menit untuk start_at agar tidak gagal karena sedikit latency jam
	if input.StartAt.Before(now.Add(-5 * time.Minute)) {
		input.StartAt = now
	}
	if !input.EndAt.After(input.StartAt) {
		return nil, errors.New("tanggal berakhir promosi harus setelah tanggal mulai")
	}

	totalDurationHours := input.EndAt.Sub(input.StartAt).Hours()
	if totalDurationHours > 90*24 {
		return nil, errors.New("durasi maksimal promosi adalah 90 hari")
	}

	// Cek apakah main_product_id ada di list produk
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
	if !mainProductFound {
		return nil, errors.New("produk utama (main_product_id) harus merupakan salah satu produk yang dipilih")
	}

	// Cek konflik: pastikan produk tidak sedang dalam promosi aktif/terjadwal lain
	activePromoMap, err := s.repo.GetActivePromotionProductIDs(businessID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa status promosi produk: %w", err)
	}
	for pID := range productIDMap {
		if existingPromoID, conflict := activePromoMap[pID]; conflict {
			return nil, fmt.Errorf("produk %s sudah terdaftar dalam promosi aktif/terjadwal (%s)", pID, existingPromoID)
		}
	}

	// Ambil data produk dari DB untuk validasi dan snapshot harga
	eligibleProducts, err := s.repo.GetEligibleProducts(businessID)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat data produk: %w", err)
	}
	eligibleMap := make(map[string]models.EligibleProductItem)
	for _, ep := range eligibleProducts {
		eligibleMap[ep.ProductID] = ep
	}

	var mainProductItem *models.EligibleProductItem
	promotionID := utils.GenerateUUID("PRM")
	isActiveNow := !input.StartAt.After(now)

	status := models.PromotionStatusScheduled
	var notifiedAt *time.Time
	if isActiveNow {
		status = models.PromotionStatusActive
		notifiedAt = &now
	}

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
			if item.DiscountType == nil || *item.DiscountType == "" {
				return nil, fmt.Errorf("tipe diskon (discount_type) wajib diisi untuk produk %s", item.ProductID)
			}
			finalDiscType = strings.ToLower(strings.TrimSpace(*item.DiscountType))
			if finalDiscType != "percentage" && finalDiscType != "nominal" {
				return nil, fmt.Errorf("tipe diskon produk %s harus 'percentage' atau 'nominal'", item.ProductID)
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
			PostID:       utils.GenerateUUID("POST"),
			BusinessID:   businessID,
			Content:      caption,
			Images:       images,
			IsPromotion:  true,
			PromotionID:  &promotionID,
			LikeCount:    0,
			CommentCount: 0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	if err := s.repo.CreatePromotion(promo, promoProducts, productsToUpdate, autoPost); err != nil {
		return nil, fmt.Errorf("gagal membuat promosi: %w", err)
	}

	// Jika langsung aktif, kirim notifikasi dan sync search
	if isActiveNow && s.notifService != nil {
		var mainProdModel *models.Product
		if mainProductItem != nil {
			mainProdModel = &models.Product{
				ProductID:      mainProductItem.ProductID,
				ProductName:    mainProductItem.ProductName,
				ProductPreview: []string{mainProductItem.ImageURL},
			}
		}
		notification.NotifyPromotionCreated(database.DB, s.notifService, businessID, promo, mainProdModel, productIDsList, requesterID)
		go meilisearch.SyncBusiness(database.DB, businessID)
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

	summary, err := s.repo.GetPromotionSummary(businessID)
	if err != nil {
		return models.GetPromotionsResponse{}, err
	}

	promos, err := s.repo.GetPromotions(businessID, tab, cursor, limit)
	if err != nil {
		return models.GetPromotionsResponse{}, err
	}

	hasMore := false
	if len(promos) > limit {
		hasMore = true
		promos = promos[:limit]
	}

	nextCursor := ""
	if hasMore && len(promos) > 0 {
		nextCursor = promos[len(promos)-1].PromotionID
	}

	now := time.Now()
	items := make([]models.PromotionListItem, 0, len(promos))
	for _, p := range promos {
		remainingDays := 0
		if p.EndAt.After(now) {
			remainingDays = int(math.Ceil(p.EndAt.Sub(now).Hours() / 24.0))
		}

		mainCard := models.PromotionMainProductCard{}
		if p.MainProduct != nil {
			img := ""
			if len(p.MainProduct.ProductPreview) > 0 {
				img = p.MainProduct.ProductPreview[0]
			}
			cut, pct := utils.CalculateProductDiscount(p.MainProduct.Price, p.MainProduct.DiscountType, p.MainProduct.Discount, p.MainProduct.MaxDiscount)
			mainCard = models.PromotionMainProductCard{
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

		items = append(items, models.PromotionListItem{
			PromotionID:     p.PromotionID,
			Caption:         p.Caption,
			MainProduct:     mainCard,
			ProductCount:    len(p.Products),
			RemainingDays:   remainingDays,
			StartAt:         p.StartAt,
			EndAt:           p.EndAt,
			Status:          p.Status,
			ImpressionCount: p.ImpressionCount,
			ClickCount:      p.ClickCount,
			CreatedAt:       p.CreatedAt,
		})
	}

	return models.GetPromotionsResponse{
		Summary:    summary,
		Promotions: items,
		Pagination: models.PaginationMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *promotionService) buildPromotionDetailResponse(promo *models.Promotion) *models.PromotionDetailResponse {
	now := time.Now()
	remainingDays := 0
	if promo.EndAt.After(now) {
		remainingDays = int(math.Ceil(promo.EndAt.Sub(now).Hours() / 24.0))
	}

	productDetails := make([]models.PromotionProductDetailItem, 0, len(promo.Products))
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

		productDetails = append(productDetails, models.PromotionProductDetailItem{
			ProductID:      pp.ProductID,
			ProductName:    name,
			ImageURL:       img,
			OriginalPrice:  origPrice,
			FinalPrice:     finalPrice,
			DiscountSource: pp.DiscountSource,
			DiscountType:   pp.DiscountType,
			Discount:       pp.Discount,
			DiscountAmount: cut,
			DiscountPct:    pct,
			MaxDiscount:    pp.MaxDiscount,
			IsMainProduct:  pp.ProductID == promo.MainProductID,
		})
	}

	return &models.PromotionDetailResponse{
		PromotionID:     promo.PromotionID,
		BusinessID:      promo.BusinessID,
		Caption:         promo.Caption,
		MainProductID:   promo.MainProductID,
		PostID:          promo.PostID,
		StartAt:         promo.StartAt,
		EndAt:           promo.EndAt,
		Status:          promo.Status,
		RemainingDays:   remainingDays,
		ImpressionCount: promo.ImpressionCount,
		ClickCount:      promo.ClickCount,
		CreatedAt:       promo.CreatedAt,
		Products:        productDetails,
	}
}

func (s *promotionService) GetPromotionDetail(requesterID, role, businessID, promotionID string) (*models.PromotionDetailResponse, error) {
	if !s.repo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("anda tidak memiliki akses ke bisnis ini")
	}

	promo, err := s.repo.GetPromotionByID(businessID, promotionID)
	if err != nil {
		return nil, errors.New("promosi tidak ditemukan")
	}

	return s.buildPromotionDetailResponse(promo), nil
}

func (s *promotionService) GetCustomerPromotionDetail(promotionID string) (*models.PromotionDetailResponse, error) {
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
	return nil
}

func (s *promotionService) StopPromotion(requesterID, role, businessID, promotionID string) error {
	if err := s.checkManageAccess(requesterID, role, businessID); err != nil {
		return err
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
	return nil
}

func (s *promotionService) RecordPromotionEvents(events []models.PromotionEvent) error {
	return s.repo.RecordPromotionEvents(events)
}
