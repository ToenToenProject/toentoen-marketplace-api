package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductInteractionService interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	AddProductReview(productID, accountID string, input models.CreateCommentInput) error
	GetProductReviews(productID, currentAccountID string) ([]models.CommentResponse, error)
	GetAllProductReviewsByBusinessID(businessID string) ([]models.CommentResponse, error)
	ToggleReviewLike(reviewID, accountID string) (bool, error)
	ReportReview(reviewID, accountID, reason string) error
}

type productInteractionService struct {
	repo         ProductInteractionRepository
	businessRepo business.BusinessRepository
	notifService notification.NotificationService
}

func NewProductInteractionService(repo ProductInteractionRepository, businessRepo business.BusinessRepository, notifService notification.NotificationService) ProductInteractionService {
	return &productInteractionService{repo, businessRepo, notifService}
}

// 1. TOGGLE LIKE PRODUK
func (s *productInteractionService) ToggleProductLike(productID, accountID string) (bool, error) {
	isLiked, err := s.repo.ToggleProductLike(productID, accountID)
	if err == nil && isLiked {
		go func() {
			businessID, errB := s.repo.GetBusinessIDByProductID(productID)
			if errB == nil {
				staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
				
				data := map[string]interface{}{
					"product_id":   productID,
					"business_id":  businessID,
					"trigger_name": "Seseorang",
				}
				
				s.notifService.SendNotification(
					staffIDs,
					accountID,
					"product_like",
					productID,
					"Produk Anda disukai!",
					"Seseorang menyukai produk Anda.",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

// 2. CREATE REVIEW / BALASAN PRODUK (Logika 1-Level Nesting)
func (s *productInteractionService) AddProductReview(productID, accountID string, input models.CreateCommentInput) error {
	
	businessID, err := s.repo.GetBusinessIDByProductID(productID)
	if err != nil {
		return errors.New("gagal menemukan toko untuk produk ini")
	}

	review := models.ProductReview{
		ReviewID:   utils.GenerateUUID("RVW"),
		ProductID:  productID,
		BusinessID: businessID,
		AccountID:  accountID,
		Content:    input.Content,
		Rating:     input.Rating,
		MediaURLs:  input.MediaURLs,
	}

	if input.ReplyToCommentID != nil && *input.ReplyToCommentID != "" {
		parentReview, err := s.repo.GetProductReviewByID(*input.ReplyToCommentID)
		if err != nil {
			return errors.New("review yang ingin dibalas tidak ditemukan")
		}

		if parentReview.ReplyToReviewID == nil {
			review.ReplyToReviewID = input.ReplyToCommentID
		} else {
			review.ReplyToReviewID = parentReview.ReplyToReviewID
			review.RepliedToAccountID = &parentReview.AccountID
			if parentReview.Account != nil {
				name := parentReview.Account.FirstName + " " + parentReview.Account.LastName 
				review.RepliedToAccountName = &name
			}
		}
	}

	if err := s.repo.CreateProductReview(&review); err != nil {
		return errors.New("gagal menambahkan review produk")
	}

	go func() {
		staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
		data := map[string]interface{}{
			"product_id":   productID,
			"business_id":  businessID,
			"review_id":    review.ReviewID,
			"trigger_name": "Seseorang",
		}
		
		s.notifService.SendNotification(
			staffIDs,
			accountID,
			"product_review",
			review.ReviewID,
			"Review Baru!",
			"Ada pelanggan yang memberikan review pada produk Anda: "+truncateString(input.Content, 50),
			data,
			nil,
		)
	}()

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// 3. GET REVIEW PRODUK (Mapping Array Datar ke JSON Bersarang)
func (s *productInteractionService) GetProductReviews(productID, currentAccountID string) ([]models.CommentResponse, error) {
	rawReviews, err := s.repo.GetReviewsByProductID(productID)
	if err != nil {
		return nil, errors.New("gagal mengambil data review produk")
	}

	if len(rawReviews) == 0 {
		return []models.CommentResponse{}, nil
	}

	likedReviewIDs := make(map[string]bool)
	if currentAccountID != "" {
		for _, rv := range rawReviews {
			likes, _ := s.repo.GetReviewLikes(rv.ReviewID)
			for _, l := range likes {
				if l.AccountID == currentAccountID {
					likedReviewIDs[rv.ReviewID] = true
					break
				}
			}
		}
	}

	reviewMap := make(map[string]*models.CommentResponse)
	var finalReviews []models.CommentResponse

	for _, rv := range rawReviews {
		accName := "Pengguna"
		var accAvatar *string
		if rv.Account != nil {
			accName = rv.Account.FirstName + " " + rv.Account.LastName
			if rv.Account.ProfilePictureUrl != nil {
				accAvatar = rv.Account.ProfilePictureUrl
			}
		}

		item := &models.CommentResponse{
			CommentID:            rv.ReviewID, // Dipetakan ke CommentID di DTO
			AccountID:            rv.AccountID,
			AccountName:          accName,
			AccountAvatar:        accAvatar,
			Content:              rv.Content,
			LikeCount:            rv.LikeCount,
			IsLikedByMe:          likedReviewIDs[rv.ReviewID],
			Rating:               rv.Rating,
			MediaURLs:            rv.MediaURLs,
			CreatedAt:            rv.CreatedAt,
			RepliedToAccountID:   rv.RepliedToAccountID,
			RepliedToAccountName: rv.RepliedToAccountName,
			Replies:              []models.CommentResponse{},
		}
		reviewMap[rv.ReviewID] = item
	}

	for _, rv := range rawReviews {
		currentItem := reviewMap[rv.ReviewID]

		if rv.ReplyToReviewID != nil {
			if parentItem, exists := reviewMap[*rv.ReplyToReviewID]; exists {
				parentItem.Replies = append(parentItem.Replies, *currentItem)
				parentItem.RepliesCount = len(parentItem.Replies)
			}
		}
	}

	for _, rv := range rawReviews {
		if rv.ReplyToReviewID == nil {
			finalReviews = append(finalReviews, *reviewMap[rv.ReviewID])
		}
	}

	return finalReviews, nil
}

func (s *productInteractionService) GetAllProductReviewsByBusinessID(businessID string) ([]models.CommentResponse, error) {
	rawReviews, err := s.repo.GetAllProductReviewsByBusinessID(businessID)
	if err != nil {
		return nil, errors.New("gagal mengambil review produk")
	}

	var finalReviews []models.CommentResponse
	for _, rv := range rawReviews {
		accName := "Pengguna"
		var accAvatar *string
		if rv.Account != nil {
			accName = rv.Account.FirstName + " " + rv.Account.LastName
			if rv.Account.ProfilePictureUrl != nil {
				accAvatar = rv.Account.ProfilePictureUrl
			}
		}

		item := models.CommentResponse{
			CommentID:            rv.ReviewID,
			AccountID:            rv.AccountID,
			AccountName:          accName,
			AccountAvatar:        accAvatar,
			Content:              rv.Content,
			LikeCount:            rv.LikeCount,
			IsLikedByMe:          false, 
			Rating:               rv.Rating,
			MediaURLs:            rv.MediaURLs,
			CreatedAt:            rv.CreatedAt,
			RepliedToAccountID:   rv.RepliedToAccountID,
			RepliedToAccountName: rv.RepliedToAccountName,
			RepliesCount:         len(rv.Replies),
		}

		if rv.Product != nil {
			item.ProductID = &rv.Product.ProductID
			item.ProductName = &rv.Product.ProductName
			item.ProductPrice = &rv.Product.Price
			if rv.Product.Discount > 0 {
				orig := rv.Product.Price + rv.Product.Discount
				item.ProductOriginalPrice = &orig
			}
			if len(rv.Product.ProductImages) > 0 {
				item.ProductImage = &rv.Product.ProductImages[0]
			}
		}
		finalReviews = append(finalReviews, item)
	}

	return finalReviews, nil
}

func (s *productInteractionService) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	return s.repo.ToggleReviewLike(reviewID, accountID)
}

func (s *productInteractionService) ReportReview(reviewID, accountID, reason string) error {
	report := models.ProductReviewReport{
		AccountID: accountID,
		ReviewID:  reviewID,
		Reason:    reason,
	}
	return s.repo.CreateReviewReport(&report)
}