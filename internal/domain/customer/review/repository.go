package review

import (
	"toentoen-backend/internal/domain/shared/models"
	"gorm.io/gorm"
)

type ReviewRepository interface {
	CreateReview(review *models.Review) error
	GetReviewByID(reviewID string) (*models.Review, error)
	UpdateReview(review *models.Review) error
	GetAllReviews(businessID string) ([]models.Review, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)
}

type reviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &reviewRepository{db}
}

func (r *reviewRepository) CreateReview(review *models.Review) error {
	return r.db.Create(review).Error
}

func (r *reviewRepository) GetReviewByID(reviewID string) (*models.Review, error) {
	var review models.Review
	err := r.db.Where("review_id = ?", reviewID).First(&review).Error
	return &review, err
}

func (r *reviewRepository) UpdateReview(review *models.Review) error {
	return r.db.Save(review).Error
}

func (r *reviewRepository) GetAllReviews(businessID string) ([]models.Review, error) {
	var reviews []models.Review
	query := r.db.Model(&models.Review{})

	if businessID != "" {
		query = query.Where("business_id = ?", businessID)
	}

	err := query.Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

// --- Implementasinya ---
func (r *reviewRepository) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	var summary models.ReviewSummary
	var total float64 // Untuk menghitung rata-rata
	// var count int64   // Total review	

	// Struct sementara untuk menangkap hasil GROUP BY SQL
	type Result struct {
		Rating int
		Count  int64
	}
	var results []Result

	// Query: Kelompokkan berdasarkan rating, dan hitung jumlahnya
	err := r.db.Model(&models.Review{}).
		Select("rating, count(*) as count").
		Where("business_id = ?", businessID).
		Group("rating").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Petakan hasil ke dalam struct Distribution
	for _, res := range results {
		summary.TotalReviews += res.Count
		total += float64(res.Rating) * float64(res.Count) // Hitung total bintang

		switch res.Rating {
		case 5: summary.Distribution.Star5 = res.Count
		case 4: summary.Distribution.Star4 = res.Count
		case 3: summary.Distribution.Star3 = res.Count
		case 2: summary.Distribution.Star2 = res.Count
		case 1: summary.Distribution.Star1 = res.Count
		}
	}

	// Hitung rata-rata rating (hindari pembagian dengan nol)
	if summary.TotalReviews > 0 {
		summary.AverageRating = total / float64(summary.TotalReviews)
		// Opsional: Bulatkan ke 1 angka di belakang koma (misal: 4.56 -> 4.6)
		summary.AverageRating = float64(int(summary.AverageRating*10)) / 10 
	}

	return &summary, nil
}