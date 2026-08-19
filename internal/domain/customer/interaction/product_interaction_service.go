package interaction

import (
	"errors"
	"fmt"
	"strings"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductInteractionService interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	AddProductReview(productID, accountID string, input models.CreateCommentInput) error
	EditProductComment(productID, reviewID, accountID string, input models.UpdateCommentInput) error
	DeleteProductComment(productID, reviewID, requesterID string) error
	GetProductReviews(productID, currentAccountID, filterType string, rating *int) ([]models.CommentResponse, error)
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

func countWords(s string) int {
	return len(strings.Fields(strings.TrimSpace(s)))
}

func getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, isByOwner bool) string {
	if isOwnerLiked && isOwnerReplied {
		return "Pemilik toko menyukai dan membalas"
	}
	if isOwnerLiked {
		return "Disukai Pemilik"
	}
	if isByOwner {
		return "Pemilik Toko"
	}
	return ""
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
					"product_id":  productID,
					"business_id": businessID,
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

// 2. CREATE REVIEW / COMMENT PRODUK
func (s *productInteractionService) AddProductReview(productID, accountID string, input models.CreateCommentInput) error {
	businessID, err := s.repo.GetBusinessIDByProductID(productID)
	if err != nil {
		return errors.New("Produk tidak ditemukan atau tidak memiliki toko yang valid")
	}

	isReview := input.Rating != nil && *input.Rating > 0
	isReply := input.ReplyToCommentID != nil && *input.ReplyToCommentID != ""

	// Validasi jika ini adalah Review (bukan sekadar komentar/balasan)
	if isReview && !isReply {
		if countWords(input.Content) < 10 {
			return errors.New("Ulasan produk wajib memiliki minimal 10 kata")
		}

		hasReviewed, err := s.repo.HasUserReviewedProduct(productID, accountID)
		if err == nil && hasReviewed {
			return errors.New("Anda sudah memberikan ulasan untuk produk ini. Silakan gunakan fitur edit jika ingin mengubah ulasan.")
		}
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

	var parentReview *models.ProductReview
	if isReply {
		parent, errP := s.repo.GetProductReviewByID(*input.ReplyToCommentID)
		if errP != nil {
			return errors.New("komentar/ulasan yang ingin dibalas tidak ditemukan")
		}
		parentReview = parent

		if parent.ReplyToReviewID == nil {
			review.ReplyToReviewID = input.ReplyToCommentID
		} else {
			review.ReplyToReviewID = parent.ReplyToReviewID
			review.RepliedToAccountID = &parent.AccountID
			if parent.Account != nil {
				name := parent.Account.FirstName + " " + parent.Account.LastName
				review.RepliedToAccountName = &name
			}
		}
	}

	if err := s.repo.CreateProductReview(&review); err != nil {
		return errors.New("gagal menambahkan ulasan/komentar produk")
	}

	// Kirim Notifikasi
	go func() {
		if isReply && parentReview != nil {
			recipientID := parentReview.AccountID
			if review.RepliedToAccountID != nil && *review.RepliedToAccountID != "" {
				recipientID = *review.RepliedToAccountID
			}

			data := map[string]interface{}{
				"product_id":  productID,
				"business_id": businessID,
				"review_id":   review.ReviewID,
			}
			s.notifService.SendNotification(
				[]string{recipientID},
				accountID,
				"product_reply",
				productID,
				"Balasan Komentar Baru",
				fmt.Sprintf("Ada yang membalas komentar Anda pada produk: %s", utils.TruncateString(input.Content, 50)),
				data,
				nil,
			)
		} else if isReview {
			staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
			data := map[string]interface{}{
				"product_id":  productID,
				"business_id": businessID,
				"review_id":   review.ReviewID,
				"rating":      *input.Rating,
			}
			s.notifService.SendNotification(
				staffIDs,
				accountID,
				"product_review",
				review.ReviewID,
				"Ulasan Produk Baru!",
				fmt.Sprintf("Pelanggan memberikan rating %.1f★: %s", *input.Rating, utils.TruncateString(input.Content, 50)),
				data,
				nil,
			)
		} else {
			staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
			data := map[string]interface{}{
				"product_id":  productID,
				"business_id": businessID,
				"comment_id":  review.ReviewID,
			}
			s.notifService.SendNotification(
				staffIDs,
				accountID,
				"product_comment",
				productID,
				"Komentar Produk Baru",
				fmt.Sprintf("Ada komentar baru pada produk Anda: %s", utils.TruncateString(input.Content, 50)),
				data,
				nil,
			)
		}
	}()

	return nil
}

// 3. EDIT REVIEW / COMMENT PRODUK
func (s *productInteractionService) EditProductComment(productID, reviewID, accountID string, input models.UpdateCommentInput) error {
	review, err := s.repo.GetProductReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan/komentar tidak ditemukan")
	}

	if review.AccountID != accountID {
		return errors.New("Anda tidak memiliki izin untuk mengedit ini")
	}

	if input.Rating != nil && *input.Rating > 0 && review.ReplyToReviewID == nil {
		if countWords(input.Content) < 10 {
			return errors.New("Ulasan produk wajib memiliki minimal 10 kata")
		}
		review.Rating = input.Rating
	}

	review.Content = input.Content
	if input.MediaURLs != nil {
		review.MediaURLs = input.MediaURLs
	}

	return s.repo.UpdateProductReview(review)
}

// 4. DELETE REVIEW / COMMENT PRODUK
func (s *productInteractionService) DeleteProductComment(productID, reviewID, requesterID string) error {
	review, err := s.repo.GetProductReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan/komentar tidak ditemukan")
	}

	isAuthorized := review.AccountID == requesterID
	if !isAuthorized {
		staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(review.BusinessID)
		for _, sid := range staffIDs {
			if sid == requesterID {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return errors.New("Anda tidak memiliki izin untuk menghapus ini")
	}

	return s.repo.DeleteProductReview(reviewID)
}

// 5. GET REVIEWS / COMMENTS PRODUK (TREE STRUCTURE + BADGES + FILTER)
func (s *productInteractionService) GetProductReviews(productID, currentAccountID, filterType string, rating *int) ([]models.CommentResponse, error) {
	rawReviews, err := s.repo.GetReviewsByProductID(productID)
	if err != nil {
		return nil, errors.New("gagal mengambil data review produk")
	}

	if len(rawReviews) == 0 {
		return []models.CommentResponse{}, nil
	}

	businessID, _ := s.repo.GetBusinessIDByProductID(productID)
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	likedReviewIDs := make(map[string]bool)
	ownerLikedReviewIDs := make(map[string]bool)
	if currentAccountID != "" || len(staffIDs) > 0 {
		for _, rv := range rawReviews {
			likes, _ := s.repo.GetReviewLikes(rv.ReviewID)
			for _, l := range likes {
				if currentAccountID != "" && l.AccountID == currentAccountID {
					likedReviewIDs[rv.ReviewID] = true
				}
				if staffMap[l.AccountID] {
					ownerLikedReviewIDs[rv.ReviewID] = true
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

		isByOwner := staffMap[rv.AccountID]
		isOwnerLiked := ownerLikedReviewIDs[rv.ReviewID]

		item := &models.CommentResponse{
			CommentID:             rv.ReviewID,
			AccountID:             rv.AccountID,
			AccountName:           accName,
			AccountAvatar:         accAvatar,
			Content:               rv.Content,
			LikeCount:             rv.LikeCount,
			IsLikedByMe:           likedReviewIDs[rv.ReviewID],
			IsByOwner:             isByOwner,
			IsOwnerLiked:          isOwnerLiked,
			OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, false, isByOwner),
			Rating:                rv.Rating,
			MediaURLs:             rv.MediaURLs,
			CreatedAt:             rv.CreatedAt,
			RepliedToAccountID:    rv.RepliedToAccountID,
			RepliedToAccountName:  rv.RepliedToAccountName,
			Replies:               []models.CommentResponse{},
		}
		reviewMap[rv.ReviewID] = item
	}

	for _, rv := range rawReviews {
		currentItem := reviewMap[rv.ReviewID]

		if rv.ReplyToReviewID != nil {
			if parentItem, exists := reviewMap[*rv.ReplyToReviewID]; exists {
				parentItem.Replies = append(parentItem.Replies, *currentItem)
				parentItem.RepliesCount = len(parentItem.Replies)
				parentItem.CommentCount = len(parentItem.Replies)

				// Jika salah satu reply adalah dari owner
				if currentItem.IsByOwner {
					parentItem.IsOwnerReplied = true
					parentItem.OwnerInteractionLabel = getOwnerInteractionLabel(parentItem.IsOwnerLiked, true, parentItem.IsByOwner)
				}
			}
		}
	}

	for _, rv := range rawReviews {
		if rv.ReplyToReviewID == nil {
			isReview := rv.Rating != nil && *rv.Rating > 0

			if filterType == "reviews" && !isReview {
				continue
			}
			if filterType == "comments" && isReview {
				continue
			}
			if rating != nil && *rating > 0 {
				if rv.Rating == nil || int(*rv.Rating) != *rating {
					continue
				}
			}

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
			CommentCount:         len(rv.Replies),
			RepliesCount:         len(rv.Replies),
			IsLikedByMe:          false,
			Rating:               rv.Rating,
			MediaURLs:            rv.MediaURLs,
			CreatedAt:            rv.CreatedAt,
			RepliedToAccountID:   rv.RepliedToAccountID,
			RepliedToAccountName: rv.RepliedToAccountName,
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
	isLiked, err := s.repo.ToggleReviewLike(reviewID, accountID)
	if err == nil && isLiked {
		go func() {
			review, errR := s.repo.GetProductReviewByID(reviewID)
			if errR == nil && review.AccountID != accountID {
				data := map[string]interface{}{
					"product_id": review.ProductID,
					"review_id":  reviewID,
				}
				s.notifService.SendNotification(
					[]string{review.AccountID},
					accountID,
					"product_review_like",
					review.ProductID,
					"Ulasan/Komentar Anda disukai!",
					"Seseorang menyukai ulasan/komentar produk Anda.",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

func (s *productInteractionService) ReportReview(reviewID, accountID, reason string) error {
	report := models.ProductReviewReport{
		AccountID: accountID,
		ReviewID:  reviewID,
		Reason:    reason,
	}
	return s.repo.CreateReviewReport(&report)
}