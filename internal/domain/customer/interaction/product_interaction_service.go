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
	// 1. General & Like Produk
	ToggleProductLike(productID, accountID string) (bool, error)

	// 2. Review Produk (Ulasan Berbintang)
	AddProductReview(productID, accountID string, input models.CreateProductReviewInput) error
	UpdateProductReview(reviewID, accountID string, input models.UpdateProductReviewInput) error
	DeleteProductReview(reviewID, requesterID string) error
	GetProductReviews(productID, currentAccountID string, rating *int) ([]models.ProductReviewResponse, error)
	GetProductReviewDetail(reviewID, currentAccountID string) (*models.ProductReviewResponse, error)
	ToggleReviewLike(reviewID, accountID string) (bool, error)
	AddReviewReply(reviewID, accountID string, input models.CreateProductReviewReplyInput) error
	DeleteReviewReply(commentID, requesterID string) error
	ToggleReviewReplyLike(commentID, accountID string) (bool, error)
	ReportReview(reviewID, accountID string, input models.CreateReportInput) error
	ReportReviewReply(commentID, accountID string, input models.CreateReportInput) error

	// 3. Komentar Diskusi / Q&A Produk
	AddProductComment(productID, accountID string, input models.CreateProductCommentInput) error
	EditProductComment(commentID, accountID string, input models.UpdateProductCommentInput) error
	DeleteProductComment(commentID, requesterID string) error
	GetProductComments(productID, currentAccountID string) ([]models.CommentResponse, error)
	ToggleProductCommentLike(commentID, accountID string) (bool, error)
	ReportProductComment(commentID, accountID string, input models.CreateReportInput) error

	// 4. Unified Likes
	ToggleUnifiedCommentLike(commentID, accountID string) (bool, error)
	ToggleUnifiedReviewLike(reviewID, accountID string) (bool, error)
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

// ----------------------------------------------------
// 1. GENERAL & LIKE PRODUK
// ----------------------------------------------------

func (s *productInteractionService) ToggleProductLike(productID, accountID string) (bool, error) {
	isLiked, err := s.repo.ToggleProductLike(productID, accountID)
	if err == nil && isLiked && s.notifService != nil {
		go func() {
			businessID, errBus := s.repo.GetBusinessIDByProductID(productID)
			if errBus == nil {
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
					"Produk Disukai",
					"Seseorang menyukai produk Anda",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

// ----------------------------------------------------
// 2. REVIEW PRODUK (ULASAN BERBINTANG)
// ----------------------------------------------------

func (s *productInteractionService) AddProductReview(productID, accountID string, input models.CreateProductReviewInput) error {
	businessID, err := s.repo.GetBusinessIDByProductID(productID)
	if err != nil {
		return errors.New("Produk tidak ditemukan atau tidak memiliki toko yang valid")
	}

	if countWords(input.Comment) < 10 {
		return errors.New("Ulasan produk wajib memiliki minimal 10 kata")
	}

	hasReviewed, err := s.repo.HasUserReviewedProduct(productID, accountID)
	if err == nil && hasReviewed {
		return errors.New("Anda sudah memberikan ulasan untuk produk ini. Silakan gunakan fitur edit jika ingin mengubah ulasan.")
	}

	reviewID := utils.GenerateUUID("REV")
	review := &models.ProductReview{
		ReviewID:     reviewID,
		ProductID:    productID,
		BusinessID:   businessID,
		AccountID:    accountID,
		Rating:       input.Rating,
		Comment:      input.Comment,
		ReviewImages: input.ReviewImages,
		ReviewVideo:  input.ReviewVideo,
	}

	if err := s.repo.CreateProductReview(review); err != nil {
		return err
	}

	if s.notifService != nil {
		go func() {
			staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
			data := map[string]interface{}{
				"product_id":  productID,
				"review_id":   reviewID,
				"business_id": businessID,
				"rating":      input.Rating,
			}
			s.notifService.SendNotification(
				staffIDs,
				accountID,
				"product_review",
				reviewID,
				"Ulasan Baru untuk Produk",
				fmt.Sprintf("Produk Anda mendapat ulasan bintang %.0f: \"%s\"", input.Rating, utils.TruncateString(input.Comment, 60)),
				data,
				nil,
			)
		}()
	}

	return nil
}

func (s *productInteractionService) UpdateProductReview(reviewID, accountID string, input models.UpdateProductReviewInput) error {
	review, err := s.repo.GetProductReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan produk tidak ditemukan")
	}

	if review.AccountID != accountID {
		return errors.New("Anda tidak memiliki izin untuk mengedit ulasan ini")
	}

	if countWords(input.Comment) < 10 {
		return errors.New("Ulasan produk wajib memiliki minimal 10 kata")
	}

	review.Rating = input.Rating
	review.Comment = input.Comment
	if input.ReviewImages != nil {
		review.ReviewImages = input.ReviewImages
	}
	review.ReviewVideo = input.ReviewVideo

	return s.repo.UpdateProductReview(review)
}

func (s *productInteractionService) DeleteProductReview(reviewID, requesterID string) error {
	review, err := s.repo.GetProductReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan produk tidak ditemukan")
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
		return errors.New("Anda tidak memiliki izin untuk menghapus ulasan ini")
	}

	return s.repo.DeleteProductReview(reviewID)
}

func (s *productInteractionService) GetProductReviews(productID, currentAccountID string, rating *int) ([]models.ProductReviewResponse, error) {
	rawReviews, err := s.repo.GetProductReviewsByProductID(productID, rating)
	if err != nil {
		return nil, errors.New("gagal mengambil data review produk")
	}

	businessID, _ := s.repo.GetBusinessIDByProductID(productID)
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	result := make([]models.ProductReviewResponse, 0, len(rawReviews))
	for _, r := range rawReviews {
		accName := "Pengguna"
		var accAvatar *string
		if r.Account != nil {
			accName = r.Account.FirstName + " " + r.Account.LastName
			accAvatar = r.Account.ProfilePictureUrl
		}

		likes, _ := s.repo.GetReviewLikes(r.ReviewID)
		isLikedByMe := false
		isOwnerLiked := false
		for _, l := range likes {
			if currentAccountID != "" && l.AccountID == currentAccountID {
				isLikedByMe = true
			}
			if staffMap[l.AccountID] {
				isOwnerLiked = true
			}
		}

		// Process replies
		isOwnerReplied := false
		replyResponses := make([]models.ProductReviewCommentResponse, 0, len(r.Comments))
		for _, c := range r.Comments {
			cName := "Pengguna"
			var cAvatar *string
			if c.Account != nil {
				cName = c.Account.FirstName + " " + c.Account.LastName
				cAvatar = c.Account.ProfilePictureUrl
			}

			isReplyByOwner := staffMap[c.AccountID]
			if isReplyByOwner {
				isOwnerReplied = true
			}

			cLikes, _ := s.repo.GetReviewReplyLikes(c.CommentID)
			cIsLikedByMe := false
			cIsOwnerLiked := false
			for _, cl := range cLikes {
				if currentAccountID != "" && cl.AccountID == currentAccountID {
					cIsLikedByMe = true
				}
				if staffMap[cl.AccountID] {
					cIsOwnerLiked = true
				}
			}

			replyResponses = append(replyResponses, models.ProductReviewCommentResponse{
				CommentID:             c.CommentID,
				ReviewID:              c.ReviewID,
				AccountID:             c.AccountID,
				AccountName:           cName,
				AccountAvatar:         cAvatar,
				Content:               c.Content,
				LikeCount:             c.LikeCount,
				IsLikedByMe:           cIsLikedByMe,
				IsByOwner:             isReplyByOwner,
				IsOwnerLiked:          cIsOwnerLiked,
				OwnerInteractionLabel: getOwnerInteractionLabel(cIsOwnerLiked, false, isReplyByOwner),
				ReplyToReviewID:       c.ReplyToCommentID,
				ReplyToCommentID:      c.ReplyToCommentID,
				RepliedToAccountID:    c.RepliedToAccountID,
				RepliedToAccountName:  c.RepliedToAccountName,
				CreatedAt:             c.CreatedAt,
			})
		}

		result = append(result, models.ProductReviewResponse{
			ReviewID:              r.ReviewID,
			ProductID:             r.ProductID,
			BusinessID:            r.BusinessID,
			AccountID:             r.AccountID,
			AccountName:           accName,
			AccountAvatar:         accAvatar,
			Rating:                r.Rating,
			Comment:               r.Comment,
			ReviewImages:          r.ReviewImages,
			ReviewVideo:           r.ReviewVideo,
			LikeCount:             r.LikeCount,
			CommentCount:          r.CommentCount,
			IsLikedByMe:           isLikedByMe,
			IsOwnerLiked:          isOwnerLiked,
			IsOwnerReplied:        isOwnerReplied,
			OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, false),
			Replies:               replyResponses,
			CreatedAt:             r.CreatedAt,
			UpdatedAt:             r.UpdatedAt,
		})
	}

	return result, nil
}

func (s *productInteractionService) GetProductReviewDetail(reviewID, currentAccountID string) (*models.ProductReviewResponse, error) {
	r, err := s.repo.GetProductReviewByID(reviewID)
	if err != nil {
		return nil, errors.New("review produk tidak ditemukan")
	}

	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(r.BusinessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	accName := "Pengguna"
	var accAvatar *string
	if r.Account != nil {
		accName = r.Account.FirstName + " " + r.Account.LastName
		accAvatar = r.Account.ProfilePictureUrl
	}

	likes, _ := s.repo.GetReviewLikes(r.ReviewID)
	isLikedByMe := false
	isOwnerLiked := false
	for _, l := range likes {
		if currentAccountID != "" && l.AccountID == currentAccountID {
			isLikedByMe = true
		}
		if staffMap[l.AccountID] {
			isOwnerLiked = true
		}
	}

	isOwnerReplied := false
	replyResponses := make([]models.ProductReviewCommentResponse, 0, len(r.Comments))
	for _, c := range r.Comments {
		cName := "Pengguna"
		var cAvatar *string
		if c.Account != nil {
			cName = c.Account.FirstName + " " + c.Account.LastName
			cAvatar = c.Account.ProfilePictureUrl
		}

		isReplyByOwner := staffMap[c.AccountID]
		if isReplyByOwner {
			isOwnerReplied = true
		}

		cLikes, _ := s.repo.GetReviewReplyLikes(c.CommentID)
		cIsLikedByMe := false
		cIsOwnerLiked := false
		for _, cl := range cLikes {
			if currentAccountID != "" && cl.AccountID == currentAccountID {
				cIsLikedByMe = true
			}
			if staffMap[cl.AccountID] {
				cIsOwnerLiked = true
			}
		}

		replyResponses = append(replyResponses, models.ProductReviewCommentResponse{
			CommentID:             c.CommentID,
			ReviewID:              c.ReviewID,
			AccountID:             c.AccountID,
			AccountName:           cName,
			AccountAvatar:         cAvatar,
			Content:               c.Content,
			LikeCount:             c.LikeCount,
			IsLikedByMe:           cIsLikedByMe,
			IsByOwner:             isReplyByOwner,
			IsOwnerLiked:          cIsOwnerLiked,
			OwnerInteractionLabel: getOwnerInteractionLabel(cIsOwnerLiked, false, isReplyByOwner),
			ReplyToReviewID:       c.ReplyToCommentID,
			ReplyToCommentID:      c.ReplyToCommentID,
			RepliedToAccountID:    c.RepliedToAccountID,
			RepliedToAccountName:  c.RepliedToAccountName,
			CreatedAt:             c.CreatedAt,
		})
	}

	return &models.ProductReviewResponse{
		ReviewID:              r.ReviewID,
		ProductID:             r.ProductID,
		BusinessID:            r.BusinessID,
		AccountID:             r.AccountID,
		AccountName:           accName,
		AccountAvatar:         accAvatar,
		Rating:                r.Rating,
		Comment:               r.Comment,
		ReviewImages:          r.ReviewImages,
		ReviewVideo:           r.ReviewVideo,
		LikeCount:             r.LikeCount,
		CommentCount:          r.CommentCount,
		IsLikedByMe:           isLikedByMe,
		IsOwnerLiked:          isOwnerLiked,
		IsOwnerReplied:        isOwnerReplied,
		OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, false),
		Replies:               replyResponses,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}, nil
}

func (s *productInteractionService) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	isLiked, err := s.repo.ToggleReviewLike(reviewID, accountID)
	if err == nil && isLiked && s.notifService != nil {
		go func() {
			review, errRev := s.repo.GetProductReviewByID(reviewID)
			if errRev == nil && review.AccountID != accountID {
				data := map[string]interface{}{
					"review_id":   reviewID,
					"product_id":  review.ProductID,
					"business_id": review.BusinessID,
				}
				s.notifService.SendNotification(
					[]string{review.AccountID},
					accountID,
					"product_review_like",
					reviewID,
					"Ulasan Anda Disukai",
					"Seseorang menyukai ulasan produk Anda",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

func (s *productInteractionService) AddReviewReply(reviewID, accountID string, input models.CreateProductReviewReplyInput) error {
	review, err := s.repo.GetProductReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan produk tidak ditemukan")
	}

	replyTarget := input.ReplyToReviewID
	if replyTarget == nil {
		replyTarget = input.ReplyToCommentID
	}

	var replyToCommentID *string
	if replyTarget != nil && strings.TrimSpace(*replyTarget) != "" && *replyTarget != reviewID {
		cleanID := strings.TrimSpace(*replyTarget)
		parentComment, errParent := s.repo.GetReviewReplyByID(cleanID)
		if errParent != nil || parentComment.ReviewID != reviewID {
			return errors.New("komentar yang ingin dibalas tidak ditemukan di ulasan ini")
		}
		replyToCommentID = &cleanID
	}

	commentID := utils.GenerateUUID("CMT")
	comment := &models.ProductReviewComment{
		CommentID:            commentID,
		ReviewID:             reviewID,
		AccountID:            accountID,
		Content:              input.Content,
		ReplyToCommentID:     replyToCommentID,
		RepliedToAccountID:   input.RepliedToAccountID,
		RepliedToAccountName: input.RepliedToAccountName,
	}

	if err := s.repo.CreateReviewReply(comment); err != nil {
		return err
	}

	if s.notifService != nil {
		go func() {
			data := map[string]interface{}{
				"review_id":   reviewID,
				"comment_id":  commentID,
				"product_id":  review.ProductID,
				"business_id": review.BusinessID,
			}
			if review.AccountID != accountID {
				s.notifService.SendNotification(
					[]string{review.AccountID},
					accountID,
					"product_review_reply",
					reviewID,
					"Balasan Baru di Ulasan Produk",
					fmt.Sprintf("Seseorang membalas ulasan Anda: \"%s\"", utils.TruncateString(input.Content, 60)),
					data,
					nil,
				)
			}
			if input.RepliedToAccountID != nil && *input.RepliedToAccountID != accountID && *input.RepliedToAccountID != review.AccountID {
				s.notifService.SendNotification(
					[]string{*input.RepliedToAccountID},
					accountID,
					"product_review_reply",
					reviewID,
					"Balasan Baru di Komentar Ulasan",
					fmt.Sprintf("Seseorang membalas komentar Anda: \"%s\"", utils.TruncateString(input.Content, 60)),
					data,
					nil,
				)
			}
		}()
	}

	return nil
}

func (s *productInteractionService) DeleteReviewReply(commentID, requesterID string) error {
	comment, err := s.repo.GetReviewReplyByID(commentID)
	if err != nil {
		return errors.New("balasan tidak ditemukan")
	}

	review, errRev := s.repo.GetProductReviewByID(comment.ReviewID)
	if errRev != nil {
		return errors.New("ulasan induk tidak ditemukan")
	}

	isAuthorized := comment.AccountID == requesterID
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
		return errors.New("Anda tidak memiliki izin untuk menghapus balasan ini")
	}

	return s.repo.DeleteReviewReply(commentID)
}

func (s *productInteractionService) ToggleReviewReplyLike(commentID, accountID string) (bool, error) {
	isLiked, err := s.repo.ToggleReviewReplyLike(commentID, accountID)
	if err == nil && isLiked && s.notifService != nil {
		go func() {
			comment, errCom := s.repo.GetReviewReplyByID(commentID)
			if errCom == nil && comment.AccountID != accountID {
				data := map[string]interface{}{
					"comment_id": commentID,
					"review_id":  comment.ReviewID,
				}
				s.notifService.SendNotification(
					[]string{comment.AccountID},
					accountID,
					"product_review_reply_like",
					commentID,
					"Balasan Ulasan Disukai",
					"Seseorang menyukai balasan ulasan Anda",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

func (s *productInteractionService) ReportReview(reviewID, accountID string, input models.CreateReportInput) error {
	report := &models.ProductReviewReport{
		ReportID:    utils.GenerateUUID("PRR"),
		ReviewID:    reviewID,
		AccountID:   accountID,
		Reason:      input.Reason,
		Description: input.Description,
		Evidence:    input.Evidence,
		Status:      "pending",
	}
	return s.repo.CreateReviewReport(report)
}

func (s *productInteractionService) ReportReviewReply(commentID, accountID string, input models.CreateReportInput) error {
	report := &models.ProductReviewCommentReport{
		ReportID:    utils.GenerateUUID("PCR"),
		CommentID:   commentID,
		AccountID:   accountID,
		Reason:      input.Reason,
		Description: input.Description,
		Evidence:    input.Evidence,
		Status:      "pending",
	}
	return s.repo.CreateReviewReplyReport(report)
}

// ----------------------------------------------------
// 3. KOMENTAR DISKUSI / Q&A PRODUK
// ----------------------------------------------------

func (s *productInteractionService) AddProductComment(productID, accountID string, input models.CreateProductCommentInput) error {
	businessID, err := s.repo.GetBusinessIDByProductID(productID)
	if err != nil {
		return errors.New("Produk tidak ditemukan atau tidak memiliki toko yang valid")
	}

	var replyToCommentID *string
	if input.ReplyToCommentID != nil && strings.TrimSpace(*input.ReplyToCommentID) != "" && *input.ReplyToCommentID != productID {
		cleanID := strings.TrimSpace(*input.ReplyToCommentID)
		parentComment, errParent := s.repo.GetProductCommentByID(cleanID)
		if errParent != nil || parentComment.ProductID != productID {
			return errors.New("komentar yang ingin dibalas tidak ditemukan di produk ini")
		}
		replyToCommentID = &cleanID
	}

	commentID := utils.GenerateUUID("CMT")
	comment := &models.ProductComment{
		CommentID:            commentID,
		ProductID:            productID,
		BusinessID:           businessID,
		AccountID:            accountID,
		Content:              input.Content,
		ReplyToCommentID:     replyToCommentID,
		RepliedToAccountID:   input.RepliedToAccountID,
		RepliedToAccountName: input.RepliedToAccountName,
	}

	if err := s.repo.CreateProductComment(comment); err != nil {
		return err
	}

	if s.notifService != nil {
		go func() {
			data := map[string]interface{}{
				"product_id":  productID,
				"comment_id":  commentID,
				"business_id": businessID,
			}
			staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
			s.notifService.SendNotification(
				staffIDs,
				accountID,
				"product_comment",
				commentID,
				"Pertanyaan / Komentar Baru Produk",
				fmt.Sprintf("Seseorang bertanya/berkomentar di produk Anda: \"%s\"", utils.TruncateString(input.Content, 60)),
				data,
				nil,
			)

			if input.RepliedToAccountID != nil && *input.RepliedToAccountID != accountID {
				s.notifService.SendNotification(
					[]string{*input.RepliedToAccountID},
					accountID,
					"product_reply",
					commentID,
					"Balasan Baru di Diskusi Produk",
					fmt.Sprintf("Seseorang membalas pertanyaan Anda: \"%s\"", utils.TruncateString(input.Content, 60)),
					data,
					nil,
				)
			}
		}()
	}

	return nil
}

func (s *productInteractionService) EditProductComment(commentID, accountID string, input models.UpdateProductCommentInput) error {
	comment, err := s.repo.GetProductCommentByID(commentID)
	if err != nil {
		return errors.New("komentar diskusi tidak ditemukan")
	}

	if comment.AccountID != accountID {
		return errors.New("Anda tidak memiliki izin untuk mengedit komentar ini")
	}

	comment.Content = input.Content
	return s.repo.UpdateProductComment(comment)
}

func (s *productInteractionService) DeleteProductComment(commentID, requesterID string) error {
	comment, err := s.repo.GetProductCommentByID(commentID)
	if err != nil {
		return errors.New("komentar diskusi tidak ditemukan")
	}

	isAuthorized := comment.AccountID == requesterID
	if !isAuthorized {
		staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(comment.BusinessID)
		for _, sid := range staffIDs {
			if sid == requesterID {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return errors.New("Anda tidak memiliki izin untuk menghapus komentar ini")
	}

	return s.repo.DeleteProductComment(commentID)
}

func (s *productInteractionService) GetProductComments(productID, currentAccountID string) ([]models.CommentResponse, error) {
	rawComments, err := s.repo.GetProductCommentsByProductID(productID)
	if err != nil {
		return nil, errors.New("gagal mengambil data komentar produk")
	}

	if len(rawComments) == 0 {
		return []models.CommentResponse{}, nil
	}

	businessID, _ := s.repo.GetBusinessIDByProductID(productID)
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	likedCommentIDs := make(map[string]bool)
	ownerLikedCommentIDs := make(map[string]bool)
	if currentAccountID != "" || len(staffIDs) > 0 {
		for _, c := range rawComments {
			likes, _ := s.repo.GetProductCommentLikes(c.CommentID)
			for _, l := range likes {
				if currentAccountID != "" && l.AccountID == currentAccountID {
					likedCommentIDs[c.CommentID] = true
				}
				if staffMap[l.AccountID] {
					ownerLikedCommentIDs[c.CommentID] = true
				}
			}
		}
	}

	commentMap := make(map[string]*models.CommentResponse)
	var finalComments []models.CommentResponse

	for _, c := range rawComments {
		accName := "Pengguna"
		var accAvatar *string
		if c.Account != nil {
			accName = c.Account.FirstName + " " + c.Account.LastName
			accAvatar = c.Account.ProfilePictureUrl
		}

		isByOwner := staffMap[c.AccountID]
		isOwnerLiked := ownerLikedCommentIDs[c.CommentID]

		item := &models.CommentResponse{
			CommentID:             c.CommentID,
			AccountID:             c.AccountID,
			AccountName:           accName,
			AccountAvatar:         accAvatar,
			Content:               c.Content,
			LikeCount:             c.LikeCount,
			IsLikedByMe:           likedCommentIDs[c.CommentID],
			IsByOwner:             isByOwner,
			IsOwnerLiked:          isOwnerLiked,
			OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, false, isByOwner),
			CreatedAt:             c.CreatedAt,
			RepliedToAccountID:    c.RepliedToAccountID,
			RepliedToAccountName:  c.RepliedToAccountName,
			Replies:               []models.CommentResponse{},
		}
		commentMap[c.CommentID] = item
	}

	for _, c := range rawComments {
		currentItem := commentMap[c.CommentID]

		if c.ReplyToCommentID != nil {
			if parentItem, exists := commentMap[*c.ReplyToCommentID]; exists {
				parentItem.Replies = append(parentItem.Replies, *currentItem)
				parentItem.RepliesCount = len(parentItem.Replies)
				parentItem.CommentCount = len(parentItem.Replies)

				if currentItem.IsByOwner {
					parentItem.IsOwnerReplied = true
					parentItem.OwnerInteractionLabel = getOwnerInteractionLabel(parentItem.IsOwnerLiked, true, parentItem.IsByOwner)
				}
			}
		}
	}

	for _, c := range rawComments {
		if c.ReplyToCommentID == nil {
			finalComments = append(finalComments, *commentMap[c.CommentID])
		}
	}

	return finalComments, nil
}

func (s *productInteractionService) ToggleProductCommentLike(commentID, accountID string) (bool, error) {
	isLiked, err := s.repo.ToggleProductCommentLike(commentID, accountID)
	if err == nil && isLiked && s.notifService != nil {
		go func() {
			comment, errCom := s.repo.GetProductCommentByID(commentID)
			if errCom == nil && comment.AccountID != accountID {
				data := map[string]interface{}{
					"comment_id":  commentID,
					"product_id":  comment.ProductID,
					"business_id": comment.BusinessID,
				}
				s.notifService.SendNotification(
					[]string{comment.AccountID},
					accountID,
					"product_comment_like",
					commentID,
					"Komentar Diskusi Disukai",
					"Seseorang menyukai komentar diskusi Anda",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

func (s *productInteractionService) ReportProductComment(commentID, accountID string, input models.CreateReportInput) error {
	report := &models.ProductCommentReport{
		ReportID:    utils.GenerateUUID("PCR"),
		CommentID:   commentID,
		AccountID:   accountID,
		Reason:      input.Reason,
		Description: input.Description,
		Evidence:    input.Evidence,
		Status:      "pending",
	}
	return s.repo.CreateProductCommentReport(report)
}

func (s *productInteractionService) ToggleUnifiedCommentLike(commentID, accountID string) (bool, error) {
	isLiked, _, err := s.repo.ToggleUnifiedCommentLike(commentID, accountID)
	return isLiked, err
}

func (s *productInteractionService) ToggleUnifiedReviewLike(reviewID, accountID string) (bool, error) {
	isLiked, _, err := s.repo.ToggleUnifiedReviewLike(reviewID, accountID)
	return isLiked, err
}