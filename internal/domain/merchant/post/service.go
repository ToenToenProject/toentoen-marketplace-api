package post

import (
	"errors"
	"strings"
	"time"
	"toentoen-backend/internal/domain/merchant/catalog"
	"toentoen-backend/internal/domain/merchant/promotion"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"
)

type BusinessPostService interface {
	CreatePost(requesterID, role, businessID string, input models.CreatePostInput) (*models.BusinessPost, error)
	GetPosts(businessID string, page, limit int) ([]models.BusinessPost, int64, error)
	UpdatePost(requesterID, role, businessID, postID string, input models.UpdatePostInput) (*models.BusinessPost, error)
	DeletePost(requesterID, role, businessID, postID string) error
	ToggleLike(accountID, postID string) (string, error)
	GetPostByID(postID string) (*models.BusinessPost, error)
	StopPostPromotion(requesterID, role, businessID, postID string) error
	ExtendPostPromotion(requesterID, role, businessID, postID string, input models.ExtendPostPromotionInput) (*models.BusinessPost, error)
}

type businessPostService struct {
	postRepo     BusinessPostRepository
	productRepo  catalog.ProductRepository // Pinjam untuk cek otoritas bisnis
	promoRepo    promotion.PromotionRepository
	notifService notification.NotificationService
}

func NewBusinessPostService(
	postRepo BusinessPostRepository,
	productRepo catalog.ProductRepository,
	promoRepo promotion.PromotionRepository,
	notifService notification.NotificationService,
) BusinessPostService {
	return &businessPostService{postRepo, productRepo, promoRepo, notifService}
}

func (s *businessPostService) CreatePost(requesterID, role, businessID string, input models.CreatePostInput) (*models.BusinessPost, error) {
	// Cek Otoritas
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}

	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermManageBusinessPosts) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk membuat postingan")
	}

	// 🛡️ VALIDASI ATURAN BISNIS
	if len(input.Images) > 5 {
		return nil, errors.New("maksimal 5 gambar diperbolehkan per postingan")
	}
	if strings.TrimSpace(input.Content) == "" && len(input.Images) == 0 && input.VideoURL == "" {
		return nil, errors.New("postingan tidak boleh kosong")
	}

	var videoPtr *string
	if input.VideoURL != "" {
		videoPtr = &input.VideoURL
	}

	isPromo := false
	if input.IsPromotion != nil && *input.IsPromotion {
		isPromo = true
	}

	var promoEndAt *time.Time
	if isPromo {
		now := time.Now()
		days := 7
		if input.DurationDays != nil && *input.DurationDays > 0 {
			days = *input.DurationDays
		}
		if input.PromotionEndAt != nil && input.PromotionEndAt.After(now) {
			promoEndAt = input.PromotionEndAt
		} else {
			end := now.AddDate(0, 0, days)
			promoEndAt = &end
		}
	}

	post := models.BusinessPost{
		PostID:         utils.GenerateUUID("PST"),
		BusinessID:     businessID,
		Content:        input.Content,
		Images:         input.Images,
		VideoURL:       videoPtr,
		IsPromotion:    isPromo,
		PromotionEndAt: promoEndAt,
	}

	if err := s.postRepo.CreatePost(&post); err != nil {
		return nil, errors.New("gagal membuat postingan")
	}

	// 📢 Notifikasi ke seluruh customer yang mem-bookmark toko ini
	if s.notifService != nil {
		notification.NotifyBusinessPostCreated(database.DB, s.notifService, businessID, post.PostID, post.Content, requesterID)
	}

	return &post, nil
}

func (s *businessPostService) GetPosts(businessID string, page, limit int) ([]models.BusinessPost, int64, error) {
	posts, totalRows, err := s.postRepo.GetPostsByBusinessID(businessID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	now := time.Now()
	for i := range posts {
		if posts[i].IsPromotion && posts[i].PromotionEndAt != nil && posts[i].PromotionEndAt.Before(now) {
			posts[i].IsPromotion = false
		}
	}

	return posts, totalRows, nil
}

func (s *businessPostService) UpdatePost(requesterID, role, businessID, postID string, input models.UpdatePostInput) (*models.BusinessPost, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}

	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermManageBusinessPosts) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengedit postingan")
	}

	post, err := s.postRepo.GetPostByID(postID)
	if err != nil || post == nil || post.BusinessID != businessID {
		return nil, errors.New("postingan tidak ditemukan di bisnis ini")
	}

	if input.Images != nil && len(*input.Images) > 5 {
		return nil, errors.New("maksimal 5 gambar diperbolehkan per postingan")
	}

	if input.Content != nil {
		post.Content = *input.Content
	}
	if input.Images != nil {
		post.Images = *input.Images
	}
	if input.VideoURL != nil {
		if strings.TrimSpace(*input.VideoURL) == "" {
			post.VideoURL = nil
		} else {
			post.VideoURL = input.VideoURL
		}
	}

	now := time.Now()
	if input.IsPromotion != nil {
		if *input.IsPromotion {
			post.IsPromotion = true
			if input.PromotionEndAt != nil && input.PromotionEndAt.After(now) {
				post.PromotionEndAt = input.PromotionEndAt
			} else if input.DurationDays != nil && *input.DurationDays > 0 {
				end := now.AddDate(0, 0, *input.DurationDays)
				post.PromotionEndAt = &end
			} else if post.PromotionEndAt == nil || post.PromotionEndAt.Before(now) {
				end := now.AddDate(0, 0, 7)
				post.PromotionEndAt = &end
			}
		} else {
			post.IsPromotion = false
			post.PromotionEndAt = nil
			// Jika terhubung ke promosi katalog, hentikan promosi katalognya juga
			if post.PromotionID != nil && *post.PromotionID != "" && s.promoRepo != nil {
				if promo, err := s.promoRepo.GetPromotionByID(businessID, *post.PromotionID); err == nil && promo != nil {
					_ = s.promoRepo.StopPromotion(businessID, promo.PromotionID, promo.Products, &post.PostID)
				}
			}
		}
	} else if post.IsPromotion {
		if input.PromotionEndAt != nil && input.PromotionEndAt.After(now) {
			post.PromotionEndAt = input.PromotionEndAt
		} else if input.DurationDays != nil && *input.DurationDays > 0 {
			end := now.AddDate(0, 0, *input.DurationDays)
			post.PromotionEndAt = &end
		}
	}

	if err := s.postRepo.UpdatePost(post); err != nil {
		return nil, errors.New("gagal memperbarui postingan")
	}
	return post, nil
}

func (s *businessPostService) StopPostPromotion(requesterID, role, businessID, postID string) error {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}
	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermManageBusinessPosts) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengelola postingan")
	}

	post, err := s.postRepo.GetPostByID(postID)
	if err != nil || post == nil || post.BusinessID != businessID {
		return errors.New("postingan tidak ditemukan di bisnis ini")
	}

	now := time.Now()
	// Jika terhubung dengan promosi katalog, hentikan promosi dan rollback diskon produk
	if post.PromotionID != nil && *post.PromotionID != "" && s.promoRepo != nil {
		promo, err := s.promoRepo.GetPromotionByID(businessID, *post.PromotionID)
		if err == nil && promo != nil && promo.Status == models.PromotionStatusActive {
			_ = s.promoRepo.StopPromotion(businessID, promo.PromotionID, promo.Products, &post.PostID)
		}
	}

	post.IsPromotion = false
	post.PromotionEndAt = &now
	post.UpdatedAt = now

	if err := s.postRepo.UpdatePost(post); err != nil {
		return errors.New("gagal menghentikan promosi postingan")
	}

	return nil
}

func (s *businessPostService) ExtendPostPromotion(requesterID, role, businessID, postID string, input models.ExtendPostPromotionInput) (*models.BusinessPost, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}
	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermManageBusinessPosts) {
		return nil, errors.New("forbidden: Anda tidak memiliki izin untuk mengelola postingan")
	}

	post, err := s.postRepo.GetPostByID(postID)
	if err != nil || post == nil || post.BusinessID != businessID {
		return nil, errors.New("postingan tidak ditemukan di bisnis ini")
	}

	now := time.Now()
	var newEndAt time.Time

	if input.EndAt != nil && input.EndAt.After(now) {
		newEndAt = *input.EndAt
	} else {
		addDays := 7
		if input.AddDays != nil && *input.AddDays > 0 {
			addDays = *input.AddDays
		}

		baseTime := now
		if post.PromotionEndAt != nil && post.PromotionEndAt.After(now) {
			baseTime = *post.PromotionEndAt
		}
		newEndAt = baseTime.AddDate(0, 0, addDays)
	}

	// Jika terhubung dengan promosi katalog, perpanjang promosi katalognya juga
	if post.PromotionID != nil && *post.PromotionID != "" && s.promoRepo != nil {
		promo, err := s.promoRepo.GetPromotionByID(businessID, *post.PromotionID)
		if err == nil && promo != nil {
			productIDs := make([]string, 0, len(promo.Products))
			for _, p := range promo.Products {
				productIDs = append(productIDs, p.ProductID)
			}
			_ = s.promoRepo.ExtendPromotion(businessID, promo.PromotionID, newEndAt, productIDs)
		}
	}

	post.IsPromotion = true
	post.PromotionEndAt = &newEndAt
	post.UpdatedAt = now

	if err := s.postRepo.UpdatePost(post); err != nil {
		return nil, errors.New("gagal memperpanjang promosi postingan")
	}

	return post, nil
}

func (s *businessPostService) DeletePost(requesterID, role, businessID, postID string) error {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}
	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermManageBusinessPosts) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menghapus postingan")
	}
	return s.postRepo.DeletePost(postID, businessID)
}

func (s *businessPostService) ToggleLike(accountID, postID string) (string, error) {
	status, err := s.postRepo.ToggleLike(postID, accountID)
	if err != nil {
		return "", errors.New("gagal memproses aksi like")
	}
	return status, nil
}

func (s *businessPostService) GetPostByID(postID string) (*models.BusinessPost, error) {
	post, err := s.postRepo.GetPostByID(postID)
	if err != nil || post == nil {
		return nil, err
	}

	if post.IsPromotion && post.PromotionEndAt != nil && post.PromotionEndAt.Before(time.Now()) {
		post.IsPromotion = false
	}
	return post, nil
}