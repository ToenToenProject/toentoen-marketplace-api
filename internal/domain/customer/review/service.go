package review

import (
	"toentoen-backend/internal/domain/merchant/business"
	"errors"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ReviewService interface {
	CreateReview(customerID string, input models.CreateReviewInput) error
	ReplyReview(ownerID, reviewID string, input models.ReplyReviewInput) error
	GetAllReviews(businessID string) ([]models.Review, error)
	GetReviewDetail(reviewID string) (*models.Review, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)

	ToggleReviewLike(reviewID, accountID string) (bool, error)
	AddReviewComment(reviewID, accountID string, input models.CreateCommentInput) error
	GetReviewComments(reviewID, currentAccountID string) ([]models.CommentResponse, error)
	ToggleReviewCommentLike(commentID, accountID string) (bool, error)
	ReportReview(reviewID, accountID string, input models.CreatePostReportInput) error
	ReportReviewComment(commentID, accountID string, input models.CreatePostReportInput) error
}

type reviewService struct {
	reviewRepo   ReviewRepository
	businessRepo business.BusinessRepository // Butuh ini untuk cek kepemilikan toko
}

func NewReviewService(rr ReviewRepository, br business.BusinessRepository) ReviewService {
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

func (s *reviewService) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	return s.reviewRepo.ToggleReviewLike(reviewID, accountID)
}

func (s *reviewService) AddReviewComment(reviewID, accountID string, input models.CreateCommentInput) error {
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return errors.New("review tidak ditemukan")
	}

	comment := models.BusinessReviewComment{
		CommentID:  utils.GenerateUUID("RVC"),
		ReviewID:   reviewID,
		BusinessID: review.BusinessID,
		AccountID:  accountID,
		Content:    input.Content,
	}

	if input.ReplyToCommentID != nil && *input.ReplyToCommentID != "" {
		parentComment, err := s.reviewRepo.GetReviewCommentByID(*input.ReplyToCommentID)
		if err != nil {
			return errors.New("komentar yang ingin dibalas tidak ditemukan")
		}

		if parentComment.ReplyToCommentID == nil {
			comment.ReplyToCommentID = input.ReplyToCommentID
		} else {
			comment.ReplyToCommentID = parentComment.ReplyToCommentID
			comment.RepliedToAccountID = &parentComment.AccountID
			if parentComment.Account != nil {
				name := parentComment.Account.FirstName + " " + parentComment.Account.LastName
				comment.RepliedToAccountName = &name
			}
		}
	}

	if err := s.reviewRepo.CreateReviewComment(&comment); err != nil {
		return errors.New("gagal menambahkan komentar review")
	}
	return nil
}

func (s *reviewService) GetReviewComments(reviewID, currentAccountID string) ([]models.CommentResponse, error) {
	rawComments, err := s.reviewRepo.GetReviewComments(reviewID)
	if err != nil {
		return nil, errors.New("gagal mengambil komentar review")
	}

	if len(rawComments) == 0 {
		return []models.CommentResponse{}, nil
	}

	likedCommentIDs := make(map[string]bool)
	if currentAccountID != "" {
		for _, rc := range rawComments {
			likes, _ := s.reviewRepo.GetReviewCommentLikes(rc.CommentID)
			for _, l := range likes {
				if l.AccountID == currentAccountID {
					likedCommentIDs[rc.CommentID] = true
					break
				}
			}
		}
	}

	commentMap := make(map[string]*models.CommentResponse)
	var finalComments []models.CommentResponse

	for _, rc := range rawComments {
		accName := "Pengguna"
		var accAvatar *string
		if rc.Account != nil {
			accName = rc.Account.FirstName + " " + rc.Account.LastName
			if rc.Account.ProfilePictureUrl != nil {
				accAvatar = rc.Account.ProfilePictureUrl
			}
		}

		item := &models.CommentResponse{
			CommentID:            rc.CommentID,
			AccountID:            rc.AccountID,
			AccountName:          accName,
			AccountAvatar:        accAvatar,
			Content:              rc.Content,
			LikeCount:            rc.LikeCount,
			IsLikedByMe:          likedCommentIDs[rc.CommentID],
			CreatedAt:            rc.CreatedAt,
			RepliedToAccountID:   rc.RepliedToAccountID,
			RepliedToAccountName: rc.RepliedToAccountName,
			Replies:              []models.CommentResponse{},
		}
		commentMap[rc.CommentID] = item
	}

	for _, rc := range rawComments {
		currentItem := commentMap[rc.CommentID]
		if rc.ReplyToCommentID != nil {
			if parentItem, exists := commentMap[*rc.ReplyToCommentID]; exists {
				parentItem.Replies = append(parentItem.Replies, *currentItem)
			}
		}
	}

	for _, rc := range rawComments {
		if rc.ReplyToCommentID == nil {
			finalComments = append(finalComments, *commentMap[rc.CommentID])
		}
	}

	return finalComments, nil
}

func (s *reviewService) ToggleReviewCommentLike(commentID, accountID string) (bool, error) {
	return s.reviewRepo.ToggleReviewCommentLike(commentID, accountID)
}

func (s *reviewService) ReportReview(reviewID, accountID string, input models.CreatePostReportInput) error {
	report := models.BusinessReviewReport{
		AccountID:   accountID,
		ReviewID:    reviewID,
		Reason:      input.Reason,
	}
	return s.reviewRepo.CreateReviewReport(&report)
}

func (s *reviewService) ReportReviewComment(commentID, accountID string, input models.CreatePostReportInput) error {
	report := models.BusinessReviewCommentReport{
		AccountID: accountID,
		CommentID: commentID,
		Reason:    input.Reason,
	}
	return s.reviewRepo.CreateReviewCommentReport(&report)
}