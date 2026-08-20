package interaction

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"

	"gorm.io/gorm"
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
	GetSmartDisplayProductReviews(productID, currentAccountID string) ([]models.ProductReviewResponse, error)
	GetProductReviewSummary(productID string) (*models.ReviewSummary, error)
	GetAllProductReviewsPaginated(productID, currentAccountID string, page, limit, ratingFilter int, sortStr string) ([]models.ProductReviewResponse, int64, error)
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

	// 4. Unified Likes & Comments
	ToggleUnifiedCommentLike(commentID, accountID string) (bool, error)
	ToggleUnifiedReviewLike(reviewID, accountID string) (bool, error)
	DeleteUnifiedComment(commentID, requesterID string) error
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

// Get Smart Display 5 Reviews Produk
func (s *productInteractionService) GetSmartDisplayProductReviews(productID, currentAccountID string) ([]models.ProductReviewResponse, error) {
	rawReviews, err := s.repo.GetEligibleProductReviewsForSmartDisplay(productID)
	if err != nil {
		return nil, err
	}

	// Filter kualitas: Hanya review dengan min. 10 kata
	var eligible []models.ProductReview
	for _, r := range rawReviews {
		if countWords(r.Comment) >= 10 {
			eligible = append(eligible, r)
		}
	}

	if len(eligible) == 0 {
		return []models.ProductReviewResponse{}, nil
	}

	var businessID string
	if len(eligible) > 0 {
		businessID = eligible[0].BusinessID
	}
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	// Hitung rata-rata rating
	summary, _ := s.repo.GetProductReviewSummary(productID)
	avgRating := 5.0
	if summary != nil && summary.TotalReviews > 0 {
		avgRating = summary.AverageRating
	}

	// Helper score calculator
	type ScoredReview struct {
		Review         models.ProductReview
		Score          float64
		IsOwnerLiked   bool
		IsOwnerReplied bool
	}

	var highList, midList, lowList []ScoredReview

	for _, r := range eligible {
		likes, _ := s.repo.GetReviewLikes(r.ReviewID)
		isOwnerLiked := false
		for _, l := range likes {
			if staffMap[l.AccountID] {
				isOwnerLiked = true
				break
			}
		}

		isOwnerReplied := false
		for _, c := range r.Comments {
			if staffMap[c.AccountID] {
				isOwnerReplied = true
				break
			}
		}

		ownerLikeBonus := 0.0
		if isOwnerLiked {
			ownerLikeBonus = 10.0
		}
		ownerReplyBonus := 0.0
		if isOwnerReplied {
			ownerReplyBonus = 15.0
		}

		wordBonus := float64(countWords(r.Comment)) / 10.0
		score := float64(r.LikeCount*2) + float64(len(r.Comments)*3) + ownerLikeBonus + ownerReplyBonus + wordBonus

		sr := ScoredReview{
			Review:         r,
			Score:          score,
			IsOwnerLiked:   isOwnerLiked,
			IsOwnerReplied: isOwnerReplied,
		}

		if r.Rating >= 4 {
			highList = append(highList, sr)
		} else if r.Rating == 3 {
			midList = append(midList, sr)
		} else {
			lowList = append(lowList, sr)
		}
	}

	sortScored := func(list []ScoredReview) {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Score > list[j].Score
		})
	}
	sortScored(highList)
	sortScored(midList)
	sortScored(lowList)

	// Target kuota 5 review berdasarkan rata-rata rating
	var targetHigh, targetMid, targetLow int
	if avgRating >= 4.0 {
		targetHigh, targetMid, targetLow = 3, 1, 1
	} else if avgRating >= 3.0 {
		targetHigh, targetMid, targetLow = 2, 2, 1
	} else {
		targetHigh, targetMid, targetLow = 1, 1, 3
	}

	var selected []ScoredReview
	usedMap := make(map[string]bool)

	takeN := func(list []ScoredReview, n int) {
		count := 0
		for _, sr := range list {
			if count >= n {
				break
			}
			if !usedMap[sr.Review.ReviewID] {
				selected = append(selected, sr)
				usedMap[sr.Review.ReviewID] = true
				count++
			}
		}
	}

	takeN(highList, targetHigh)
	takeN(midList, targetMid)
	takeN(lowList, targetLow)

	// Jika belum mencapai 5, isi dari sisa daftar apa saja berdasarkan skor tertinggi
	if len(selected) < 5 {
		var allRemaining []ScoredReview
		for _, l := range [][]ScoredReview{highList, midList, lowList} {
			for _, sr := range l {
				if !usedMap[sr.Review.ReviewID] {
					allRemaining = append(allRemaining, sr)
				}
			}
		}
		sortScored(allRemaining)
		for _, sr := range allRemaining {
			if len(selected) >= 5 {
				break
			}
			selected = append(selected, sr)
			usedMap[sr.Review.ReviewID] = true
		}
	}

	// Map ke Response DTO
	var responses []models.ProductReviewResponse
	for _, sr := range selected {
		r := sr.Review
		isLikedByMe := false
		if currentAccountID != "" {
			likes, _ := s.repo.GetReviewLikes(r.ReviewID)
			for _, l := range likes {
				if l.AccountID == currentAccountID {
					isLikedByMe = true
					break
				}
			}
		}

		accName := "Pengguna"
		var accAvatar *string
		if r.Account != nil {
			accName = r.Account.FirstName + " " + r.Account.LastName
			accAvatar = r.Account.ProfilePictureUrl
		}

		replyResponses := make([]models.ProductReviewCommentResponse, 0, len(r.Comments))
		for _, c := range r.Comments {
			cName := "Pengguna"
			var cAvatar *string
			if c.Account != nil {
				cName = c.Account.FirstName + " " + c.Account.LastName
				cAvatar = c.Account.ProfilePictureUrl
			}
			isReplyByOwner := staffMap[c.AccountID]
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

		resp := models.ProductReviewResponse{
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
			CommentCount:          len(r.Comments),
			IsLikedByMe:           isLikedByMe,
			IsOwnerLiked:          sr.IsOwnerLiked,
			IsOwnerReplied:        sr.IsOwnerReplied,
			OwnerInteractionLabel: getOwnerInteractionLabel(sr.IsOwnerLiked, sr.IsOwnerReplied, staffMap[r.AccountID]),
			Replies:               replyResponses,
			CreatedAt:             r.CreatedAt,
			UpdatedAt:             r.UpdatedAt,
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *productInteractionService) GetProductReviewSummary(productID string) (*models.ReviewSummary, error) {
	return s.repo.GetProductReviewSummary(productID)
}

func (s *productInteractionService) GetAllProductReviewsPaginated(productID, currentAccountID string, page, limit, ratingFilter int, sortStr string) ([]models.ProductReviewResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	reviews, total, err := s.repo.GetProductReviewsPaginated(productID, ratingFilter, sortStr, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var businessID string
	if len(reviews) > 0 {
		businessID = reviews[0].BusinessID
	} else {
		businessID, _ = s.repo.GetBusinessIDByProductID(productID)
	}
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	var responses []models.ProductReviewResponse
	for _, r := range reviews {
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

		accName := "Pengguna"
		var accAvatar *string
		if r.Account != nil {
			accName = r.Account.FirstName + " " + r.Account.LastName
			accAvatar = r.Account.ProfilePictureUrl
		}

		resp := models.ProductReviewResponse{
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
			CommentCount:          len(r.Comments),
			IsLikedByMe:           isLikedByMe,
			IsOwnerLiked:          isOwnerLiked,
			IsOwnerReplied:        isOwnerReplied,
			OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, staffMap[r.AccountID]),
			Replies:               replyResponses,
			CreatedAt:             r.CreatedAt,
			UpdatedAt:             r.UpdatedAt,
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
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
	reviewID = strings.TrimSpace(reviewID)
	accountID = strings.TrimSpace(accountID)

	review, err := s.repo.GetProductReviewByID(reviewID)
	if err == nil {
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

	// 2. Cek apakah ini ulasan bisnis / toko di reviews
	var bReview models.Review
	if errB := database.DB.Where("review_id = ?", reviewID).First(&bReview).Error; errB == nil {
		replyTarget := input.ReplyToReviewID
		if replyTarget == nil {
			replyTarget = input.ReplyToCommentID
		}

		var replyToCommentID *string
		if replyTarget != nil && strings.TrimSpace(*replyTarget) != "" && *replyTarget != reviewID {
			cleanID := strings.TrimSpace(*replyTarget)
			var count int64
			database.DB.Model(&models.BusinessReviewComment{}).Where("comment_id = ? AND review_id = ?", cleanID, reviewID).Count(&count)
			if count == 0 {
				return errors.New("komentar yang ingin dibalas tidak ditemukan di ulasan ini")
			}
			replyToCommentID = &cleanID
		}

		commentID := utils.GenerateUUID("CMT")
		comment := models.BusinessReviewComment{
			CommentID:            commentID,
			ReviewID:             reviewID,
			BusinessID:           bReview.BusinessID,
			AccountID:            accountID,
			Content:              input.Content,
			ReplyToCommentID:     replyToCommentID,
			RepliedToAccountID:   input.RepliedToAccountID,
			RepliedToAccountName: input.RepliedToAccountName,
		}

		if err := database.DB.Create(&comment).Error; err != nil {
			return errors.New("gagal mengirimkan balasan")
		}

		_ = database.DB.Model(&models.Review{}).Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", reviewID).UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error

		if s.notifService != nil {
			go func() {
				recipientID := bReview.AccountID
				if input.RepliedToAccountID != nil && *input.RepliedToAccountID != "" {
					recipientID = *input.RepliedToAccountID
				}
				if recipientID != accountID {
					data := map[string]interface{}{
						"business_id": bReview.BusinessID,
						"review_id":   reviewID,
						"comment_id":  commentID,
					}
					s.notifService.SendNotification(
						[]string{recipientID},
						accountID,
						"business_review_reply",
						reviewID,
						"Ulasan Anda dibalas!",
						"Seseorang membalas ulasan Anda.",
						data,
						nil,
					)
				}
			}()
		}
		return nil
	}

	return errors.New("ulasan tidak ditemukan")
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

func (s *productInteractionService) DeleteUnifiedComment(commentID, requesterID string) error {
	commentID = strings.TrimSpace(commentID)
	requesterID = strings.TrimSpace(requesterID)

	// 1. Cek di product_comments
	if err := s.DeleteProductComment(commentID, requesterID); err == nil {
		return nil
	}

	// 2. Cek di product_review_comments
	if err := s.DeleteReviewReply(commentID, requesterID); err == nil {
		return nil
	}

	// 3. Cek di business_review_comments
	var bComment models.BusinessReviewComment
	if err := database.DB.Where("comment_id = ?", commentID).First(&bComment).Error; err == nil {
		staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(bComment.BusinessID)
		isStaff := false
		for _, sid := range staffIDs {
			if sid == requesterID {
				isStaff = true
				break
			}
		}
		if bComment.AccountID != requesterID && !isStaff {
			return errors.New("tidak memiliki izin untuk menghapus balasan ini")
		}
		if err := database.DB.Delete(&bComment).Error; err != nil {
			return err
		}
		_ = database.DB.Model(&models.Review{}).Where("TRIM(LOWER(review_id)) = TRIM(LOWER(?))", bComment.ReviewID).UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - 1, 0)")).Error
		return nil
	}

	// 4. Cek di business_post_comments
	var bpComment models.BusinessPostComment
	if err := database.DB.Where("comment_id = ?", commentID).First(&bpComment).Error; err == nil {
		staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(bpComment.BusinessID)
		isStaff := false
		for _, sid := range staffIDs {
			if sid == requesterID {
				isStaff = true
				break
			}
		}
		if bpComment.AccountID != requesterID && !isStaff {
			return errors.New("tidak memiliki izin untuk menghapus komentar ini")
		}
		return database.DB.Delete(&bpComment).Error
	}

	return errors.New("komentar tidak ditemukan")
}