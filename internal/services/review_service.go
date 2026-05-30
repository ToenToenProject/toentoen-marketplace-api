package services

import (
	"errors"
	"time"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
	"toentoen-backend/pkg/utils"
)

type ReviewService interface {
	CreateReview(customerID string, input models.CreateReviewInput) error
	ReplyReview(ownerID, reviewID string, input models.ReplyReviewInput) error
	GetAllReviews(businessID string) ([]models.Review, error)
	GetReviewDetail(reviewID string) (*models.Review, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)
}

type reviewService struct {
	reviewRepo   repositories.ReviewRepository
	businessRepo repositories.BusinessRepository // Butuh ini untuk cek kepemilikan toko
}

func NewReviewService(rr repositories.ReviewRepository, br repositories.BusinessRepository) ReviewService {
	return &reviewService{rr, br}
}

// 1. Customer Create Review
func (s *reviewService) CreateReview(customerID string, input models.CreateReviewInput) error {
	review := models.Review{
		ReviewID:     utils.GenerateUUID("REV"),
		BusinessID:   input.BusinessID,
		AccountID:    customerID,
		Rating:       input.Rating,
		Comment:      input.Comment,
		
		// 👇 MAPPING DATA MEDIA
		ReviewImages: input.ReviewImages,
		ReviewVideo:  input.ReviewVideo,
	}
	
	// 1. Simpan ke database
	if err := s.reviewRepo.CreateReview(&review); err != nil {
		return err
	}

	// 2. MAGIC: Hitung ulang statistik
	summary, err := s.reviewRepo.GetReviewSummary(input.BusinessID)
	if err == nil && summary != nil {
		_ = s.businessRepo.UpdateBusinessRating(input.BusinessID, summary.AverageRating, int(summary.TotalReviews))
	}

	return nil
}

// 2. Owner Membalas Review
func (s *reviewService) ReplyReview(ownerID, reviewID string, input models.ReplyReviewInput) error {
	// Ambil reviewnya dulu
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return errors.New("review not found")
	}

	// KEAMANAN: Pastikan owner yang sedang login adalah pemilik toko dari review ini
	_, errBiz := s.businessRepo.FindBusinessByIDAndOwner(review.BusinessID, ownerID)
	if errBiz != nil {
		return errors.New("forbidden: you do not own this business")
	}

	// Update balasan
	now := time.Now()
	review.ReplyComment = &input.ReplyComment
	review.RepliedAt = &now

	return s.reviewRepo.UpdateReview(review)
}

// 3. Get All Reviews
func (s *reviewService) GetAllReviews(businessID string) ([]models.Review, error) {
	return s.reviewRepo.GetAllReviews(businessID)
}

// 4. Get Detail Review
func (s *reviewService) GetReviewDetail(reviewID string) (*models.Review, error) {
	return s.reviewRepo.GetReviewByID(reviewID)
}

// 👇 Tambahkan blok fungsi ini di paling bawah
func (s *reviewService) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	return s.reviewRepo.GetReviewSummary(businessID)
}