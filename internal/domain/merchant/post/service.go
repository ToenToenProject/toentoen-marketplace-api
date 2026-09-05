package post

import (
	"toentoen-backend/internal/domain/merchant/catalog"
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
	"strings"
)

type BusinessPostService interface {
	CreatePost(requesterID, role, businessID string, input models.CreatePostInput) (*models.BusinessPost, error)
	GetPosts(businessID string, page, limit int) ([]models.BusinessPost, int64, error)
	UpdatePost(requesterID, role, businessID, postID string, input models.UpdatePostInput) (*models.BusinessPost, error)
	DeletePost(requesterID, role, businessID, postID string) error
	ToggleLike(accountID, postID string) (string, error)
	GetPostByID(postID string) (*models.BusinessPost, error)
}

type businessPostService struct {
	postRepo    BusinessPostRepository
	productRepo catalog.ProductRepository // Pinjam untuk cek otoritas bisnis
}

func NewBusinessPostService(postRepo BusinessPostRepository, productRepo catalog.ProductRepository) BusinessPostService {
	return &businessPostService{postRepo, productRepo}
}

func (s *businessPostService) CreatePost(requesterID, role, businessID string, input models.CreatePostInput) (*models.BusinessPost, error) {
	// Cek Otoritas
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}

	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermCreatePost) {
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

	post := models.BusinessPost{
		PostID:     utils.GenerateUUID("PST"),
		BusinessID: businessID,
		Content:    input.Content,
		Images:     input.Images,
		VideoURL:   videoPtr,
	}

	if err := s.postRepo.CreatePost(&post); err != nil {
		return nil, errors.New("gagal membuat postingan")
	}
	return &post, nil
}

func (s *businessPostService) GetPosts(businessID string, page, limit int) ([]models.BusinessPost, int64, error) {
	// Tidak perlu cek otoritas, karena Postigan (Feed) biasanya bersifat publik untuk dilihat pembeli
	return s.postRepo.GetPostsByBusinessID(businessID, page, limit)
}

func (s *businessPostService) UpdatePost(requesterID, role, businessID, postID string, input models.UpdatePostInput) (*models.BusinessPost, error) {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return nil, errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
	}

	if role == "business_admin" && !s.productRepo.HasPermission(requesterID, businessID, models.PermCreatePost) {
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

	if err := s.postRepo.UpdatePost(post); err != nil {
		return nil, errors.New("gagal memperbarui postingan")
	}
	return post, nil
}

func (s *businessPostService) DeletePost(requesterID, role, businessID, postID string) error {
	if !s.productRepo.HasAccessToBusiness(requesterID, role, businessID) {
		return errors.New("forbidden: Anda tidak memiliki akses ke bisnis ini")
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
	return s.postRepo.GetPostByID(postID)
}