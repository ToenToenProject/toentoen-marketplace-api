package review

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/internal/domain/shared/permissions"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"
)

type ReviewService interface {
	CreateReview(customerID string, input models.CreateReviewInput) error
	UpdateReview(customerID, reviewID string, input models.UpdateReviewInput) error
	DeleteReview(requesterID, role, reviewID string) error
	ReplyReview(accountID, role, reviewID string, input models.CreateReviewReplyInput) error
	DeleteReviewComment(requesterID, role, commentID string) error

	GetAllReviews(businessID, currentAccountID string, page, limit, ratingFilter int, sortStr string) ([]models.BusinessReviewResponse, int64, error)
	GetSmartDisplayReviews(businessID, currentAccountID string) ([]models.BusinessReviewResponse, error)
	GetReviewDetail(reviewID, currentAccountID string) (*models.BusinessReviewResponse, error)
	GetReviewSummary(businessID string) (*models.ReviewSummary, error)

	ToggleReviewLike(reviewID, accountID string) (bool, error)
	ToggleReviewCommentLike(commentID, accountID string) (bool, error)
	ReportReview(reviewID, accountID string, input models.CreatePostReportInput) error
	ReportReviewComment(commentID, accountID string, input models.CreatePostReportInput) error
}

type reviewService struct {
	reviewRepo   ReviewRepository
	businessRepo business.BusinessRepository
	notifService notification.NotificationService
}

func NewReviewService(rr ReviewRepository, br business.BusinessRepository, ns notification.NotificationService) ReviewService {
	return &reviewService{rr, br, ns}
}

func countWords(s string) int {
	fields := strings.Fields(strings.TrimSpace(s))
	return len(fields)
}

func getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, isByOwner bool) string {
	if isOwnerLiked && isOwnerReplied {
		return "Pemilik toko menyukai dan membalas"
	}
	if isOwnerLiked {
		return "Disukai Pemilik"
	}
	if isOwnerReplied {
		return "Dibalas Pemilik"
	}
	if isByOwner {
		return "Pemilik Toko"
	}
	return ""
}

// 1. Customer Create Review Toko
func (s *reviewService) CreateReview(customerID string, input models.CreateReviewInput) error {
	if countWords(input.Comment) < 10 {
		return errors.New("Ulasan wajib memiliki minimal 10 kata")
	}

	// 🛡️ Cegah pemilik dan staf toko mereview toko sendiri
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(input.BusinessID)
	for _, sid := range staffIDs {
		if sid == customerID {
			return errors.New("Pemilik dan staf toko tidak diperkenankan memberikan ulasan pada toko sendiri")
		}
	}

	hasReviewed, err := s.reviewRepo.HasUserReviewedBusiness(input.BusinessID, customerID)
	if err == nil && hasReviewed {
		return errors.New("Anda sudah memberikan ulasan untuk toko ini. Silakan gunakan fitur edit jika ingin mengubah ulasan.")
	}

	review := models.Review{
		ReviewID:     utils.GenerateUUID("REV"),
		BusinessID:   input.BusinessID,
		AccountID:    customerID,
		Rating:       input.Rating,
		Comment:      input.Comment,
		ReviewImages: input.ReviewImages,
		ReviewVideo:  input.ReviewVideo,
	}

	if err := s.reviewRepo.CreateReview(&review); err != nil {
		return err
	}

	// Update statistik rating toko
	s.recalculateBusinessRating(input.BusinessID)

	// Kirim Notifikasi ke Pemilik Toko
	go func() {
		staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(input.BusinessID)
		data := map[string]interface{}{
			"business_id": input.BusinessID,
			"review_id":   review.ReviewID,
			"rating":      review.Rating,
		}
		s.notifService.SendNotification(
			staffIDs,
			customerID,
			"business_review",
			review.ReviewID,
			"Ulasan Baru untuk Toko Anda!",
			fmt.Sprintf("Pelanggan memberikan rating %d★: %s", review.Rating, utils.TruncateString(input.Comment, 60)),
			data,
			nil,
		)
	}()

	return nil
}

// 2. Customer Update Review Toko
func (s *reviewService) UpdateReview(customerID, reviewID string, input models.UpdateReviewInput) error {
	if countWords(input.Comment) < 10 {
		return errors.New("Ulasan wajib memiliki minimal 10 kata")
	}

	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan tidak ditemukan")
	}

	if review.AccountID != customerID {
		return errors.New("Anda tidak memiliki izin untuk mengedit ulasan ini")
	}

	review.Rating = input.Rating
	review.Comment = input.Comment
	review.ReviewImages = input.ReviewImages
	review.ReviewVideo = input.ReviewVideo
	review.UpdatedAt = time.Now()

	if err := s.reviewRepo.UpdateReview(review); err != nil {
		return err
	}

	s.recalculateBusinessRating(review.BusinessID)
	return nil
}

// 3. Delete Review Toko
func (s *reviewService) DeleteReview(requesterID, role, reviewID string) error {
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan tidak ditemukan")
	}

	// 🛡️ Hanya pembuat ulasan yang dapat menghapus ulasan sendiri
	if review.AccountID != requesterID {
		return errors.New("Anda hanya memiliki izin untuk menghapus ulasan yang Anda buat sendiri")
	}

	businessID := review.BusinessID
	if err := s.reviewRepo.DeleteReview(reviewID); err != nil {
		return err
	}

	s.recalculateBusinessRating(businessID)
	return nil
}

// 4. Balas Review Toko (Customer & Merchant)
func (s *reviewService) ReplyReview(accountID, role, reviewID string, input models.CreateReviewReplyInput) error {
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return errors.New("ulasan yang ingin dibalas tidak ditemukan")
	}

	if role == "business_admin" && !permissions.HasPermission(database.DB, accountID, review.BusinessID, models.PermReplyBusinessReviews) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk membalas ulasan toko")
	}

	replyTarget := input.ReplyToReviewID
	if replyTarget == nil {
		replyTarget = input.ReplyToCommentID
	}

	var replyToCommentID *string
	if replyTarget != nil && strings.TrimSpace(*replyTarget) != "" && *replyTarget != reviewID {
		cleanID := strings.TrimSpace(*replyTarget)
		parentComment, errParent := s.reviewRepo.GetReviewCommentByID(cleanID)
		if errParent != nil || parentComment.ReviewID != reviewID {
			return errors.New("komentar yang ingin dibalas tidak ditemukan di ulasan ini")
		}
		replyToCommentID = &cleanID
	}

	comment := models.BusinessReviewComment{
		CommentID:            utils.GenerateUUID("CMT"),
		ReviewID:             reviewID,
		BusinessID:           review.BusinessID,
		AccountID:            accountID,
		Content:              input.Content,
		ReplyToCommentID:     replyToCommentID,
		RepliedToAccountID:   input.RepliedToAccountID,
		RepliedToAccountName: input.RepliedToAccountName,
	}

	if err := s.reviewRepo.CreateReviewComment(&comment); err != nil {
		return errors.New("gagal mengirimkan balasan")
	}

	// Kirim Notifikasi
	go func() {
		recipientID := review.AccountID
		if input.RepliedToAccountID != nil && *input.RepliedToAccountID != "" {
			recipientID = *input.RepliedToAccountID
		}

		data := map[string]interface{}{
			"business_id": review.BusinessID,
			"review_id":   reviewID,
			"comment_id":  comment.CommentID,
		}

		s.notifService.SendNotification(
			[]string{recipientID},
			accountID,
			"business_review_reply",
			reviewID,
			"Balasan Ulasan Baru",
			fmt.Sprintf("Ada yang membalas ulasan Anda: %s", utils.TruncateString(input.Content, 50)),
			data,
			nil,
		)
	}()

	return nil
}

// 5. Delete Review Comment
func (s *reviewService) DeleteReviewComment(requesterID, role, commentID string) error {
	comment, err := s.reviewRepo.GetReviewCommentByID(commentID)
	if err != nil {
		return errors.New("komentar tidak ditemukan")
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

	return s.reviewRepo.DeleteReviewComment(commentID)
}

func (s *reviewService) recalculateBusinessRating(businessID string) {
	summary, err := s.reviewRepo.GetReviewSummary(businessID)
	if err == nil && summary != nil {
		_ = s.businessRepo.UpdateBusinessRating(businessID, summary.AverageRating, int(summary.TotalReviews))
	}
}

// 6. Toggle Like Review Toko
func (s *reviewService) ToggleReviewLike(reviewID, accountID string) (bool, error) {
	isLiked, err := s.reviewRepo.ToggleReviewLike(reviewID, accountID)
	if err == nil && isLiked {
		go func() {
			review, errR := s.reviewRepo.GetReviewByID(reviewID)
			if errR == nil && review.AccountID != accountID {
				data := map[string]interface{}{
					"business_id": review.BusinessID,
					"review_id":   reviewID,
				}
				s.notifService.SendNotification(
					[]string{review.AccountID},
					accountID,
					"business_review_like",
					reviewID,
					"Ulasan Anda disukai!",
					"Seseorang menyukai ulasan Anda.",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

// 7. Toggle Like Review Comment
func (s *reviewService) ToggleReviewCommentLike(commentID, accountID string) (bool, error) {
	isLiked, err := s.reviewRepo.ToggleReviewCommentLike(commentID, accountID)
	if err == nil && isLiked {
		go func() {
			comment, errC := s.reviewRepo.GetReviewCommentByID(commentID)
			if errC == nil && comment.AccountID != accountID {
				data := map[string]interface{}{
					"business_id": comment.BusinessID,
					"review_id":   comment.ReviewID,
					"comment_id":  commentID,
				}
				s.notifService.SendNotification(
					[]string{comment.AccountID},
					accountID,
					"business_review_comment_like",
					comment.ReviewID,
					"Komentar Anda disukai!",
					"Seseorang menyukai balasan ulasan Anda.",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

// 8. Report Review & Comment
func (s *reviewService) ReportReview(reviewID, accountID string, input models.CreatePostReportInput) error {
	report := models.BusinessReviewReport{
		AccountID:   accountID,
		ReviewID:    reviewID,
		Reason:      input.Reason,
		Description: input.Description,
		Evidence:    input.Evidence,
		Status:      "pending",
	}
	return s.reviewRepo.CreateReviewReport(&report)
}

func (s *reviewService) ReportReviewComment(commentID, accountID string, input models.CreatePostReportInput) error {
	report := models.BusinessReviewCommentReport{
		AccountID:   accountID,
		CommentID:   commentID,
		Reason:      input.Reason,
		Description: input.Description,
		Evidence:    input.Evidence,
		Status:      "pending",
	}
	return s.reviewRepo.CreateReviewCommentReport(&report)
}

func (s *reviewService) GetReviewSummary(businessID string) (*models.ReviewSummary, error) {
	return s.reviewRepo.GetReviewSummary(businessID)
}

// 9. Get Review Detail with Comments
func (s *reviewService) GetReviewDetail(reviewID, currentAccountID string) (*models.BusinessReviewResponse, error) {
	review, err := s.reviewRepo.GetReviewByID(reviewID)
	if err != nil {
		return nil, errors.New("ulasan tidak ditemukan")
	}

	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(review.BusinessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	likes, _ := s.reviewRepo.GetReviewLikes(reviewID)
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

	comments, _ := s.reviewRepo.GetReviewComments(reviewID)
	isOwnerReplied := false
	var commentResponses []models.BusinessReviewCommentResponse

	for _, c := range comments {
		isByOwner := staffMap[c.AccountID]
		if isByOwner {
			isOwnerReplied = true
		}

		cLikes, _ := s.reviewRepo.GetReviewCommentLikes(c.CommentID)
		isCLikedByMe := false
		isCOwnerLiked := false
		for _, cl := range cLikes {
			if currentAccountID != "" && cl.AccountID == currentAccountID {
				isCLikedByMe = true
			}
			if staffMap[cl.AccountID] {
				isCOwnerLiked = true
			}
		}

		accName := "Pengguna"
		var accAvatar *string
		if c.Account != nil {
			accName = c.Account.FirstName + " " + c.Account.LastName
			accAvatar = c.Account.ProfilePictureUrl
		}

		cResp := models.BusinessReviewCommentResponse{
			CommentID:             c.CommentID,
			ReviewID:              c.ReviewID,
			AccountID:             c.AccountID,
			AccountName:           accName,
			AccountAvatar:         accAvatar,
			Content:               c.Content,
			LikeCount:             c.LikeCount,
			IsLikedByMe:           isCLikedByMe,
			IsByOwner:             isByOwner,
			IsOwnerLiked:          isCOwnerLiked,
			OwnerInteractionLabel: getOwnerInteractionLabel(isCOwnerLiked, false, isByOwner),
			ReplyToCommentID:      c.ReplyToCommentID,
			RepliedToAccountID:    c.RepliedToAccountID,
			RepliedToAccountName:  c.RepliedToAccountName,
			CreatedAt:             c.CreatedAt,
			Replies:               []models.BusinessReviewCommentResponse{},
		}
		commentResponses = append(commentResponses, cResp)
	}

	accName := "Pengguna"
	var accAvatar *string
	if review.Account != nil {
		accName = review.Account.FirstName + " " + review.Account.LastName
		accAvatar = review.Account.ProfilePictureUrl
	}

	resp := &models.BusinessReviewResponse{
		ReviewID:              review.ReviewID,
		BusinessID:            review.BusinessID,
		AccountID:             review.AccountID,
		AccountName:           accName,
		AccountAvatar:         accAvatar,
		Rating:                review.Rating,
		Comment:               review.Comment,
		ReviewImages:          review.ReviewImages,
		ReviewVideo:           review.ReviewVideo,
		LikeCount:             review.LikeCount,
		CommentCount:          len(commentResponses),
		IsLikedByMe:           isLikedByMe,
		IsOwnerLiked:          isOwnerLiked,
		IsOwnerReplied:        isOwnerReplied,
		OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, staffMap[review.AccountID]),
		Replies:               commentResponses,
		CreatedAt:             review.CreatedAt,
		UpdatedAt:             review.UpdatedAt,
	}

	return resp, nil
}

// 10. Get Smart Display 5 Reviews
func (s *reviewService) GetSmartDisplayReviews(businessID, currentAccountID string) ([]models.BusinessReviewResponse, error) {
	rawReviews, err := s.reviewRepo.GetEligibleReviewsForSmartDisplay(businessID)
	if err != nil {
		return nil, err
	}

	// Filter kualitas: Hanya review dengan min. 10 kata
	var eligible []models.Review
	for _, r := range rawReviews {
		if countWords(r.Comment) >= 10 {
			eligible = append(eligible, r)
		}
	}

	if len(eligible) == 0 {
		return []models.BusinessReviewResponse{}, nil
	}

	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	// Hitung rata-rata rating
	summary, _ := s.reviewRepo.GetReviewSummary(businessID)
	avgRating := 5.0
	if summary != nil && summary.TotalReviews > 0 {
		avgRating = summary.AverageRating
	}

	// Helper score calculator
	type ScoredReview struct {
		Review         models.Review
		Score          float64
		IsOwnerLiked   bool
		IsOwnerReplied bool
	}

	var highList, midList, lowList []ScoredReview

	for _, r := range eligible {
		likes, _ := s.reviewRepo.GetReviewLikes(r.ReviewID)
		isOwnerLiked := false
		for _, l := range likes {
			if staffMap[l.AccountID] {
				isOwnerLiked = true
				break
			}
		}

		comments, _ := s.reviewRepo.GetReviewComments(r.ReviewID)
		isOwnerReplied := false
		for _, c := range comments {
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
		score := float64(r.LikeCount*2) + float64(r.CommentCount*3) + ownerLikeBonus + ownerReplyBonus + wordBonus

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
	var responses []models.BusinessReviewResponse
	for _, sr := range selected {
		r := sr.Review
		isLikedByMe := false
		if currentAccountID != "" {
			likes, _ := s.reviewRepo.GetReviewLikes(r.ReviewID)
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

		resp := models.BusinessReviewResponse{
			ReviewID:              r.ReviewID,
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
			IsOwnerLiked:          sr.IsOwnerLiked,
			IsOwnerReplied:        sr.IsOwnerReplied,
			OwnerInteractionLabel: getOwnerInteractionLabel(sr.IsOwnerLiked, sr.IsOwnerReplied, staffMap[r.AccountID]),
			CreatedAt:             r.CreatedAt,
			UpdatedAt:             r.UpdatedAt,
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// 11. Get All Reviews Paginated
func (s *reviewService) GetAllReviews(businessID, currentAccountID string, page, limit, ratingFilter int, sortStr string) ([]models.BusinessReviewResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	reviews, total, err := s.reviewRepo.GetReviewsPaginated(businessID, ratingFilter, sortStr, page, limit)
	if err != nil {
		return nil, 0, err
	}

	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	var responses []models.BusinessReviewResponse
	for _, r := range reviews {
		likes, _ := s.reviewRepo.GetReviewLikes(r.ReviewID)
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

		comments, _ := s.reviewRepo.GetReviewComments(r.ReviewID)
		isOwnerReplied := false
		for _, c := range comments {
			if staffMap[c.AccountID] {
				isOwnerReplied = true
				break
			}
		}

		accName := "Pengguna"
		var accAvatar *string
		if r.Account != nil {
			accName = r.Account.FirstName + " " + r.Account.LastName
			accAvatar = r.Account.ProfilePictureUrl
		}

		resp := models.BusinessReviewResponse{
			ReviewID:              r.ReviewID,
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
			OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, isOwnerReplied, staffMap[r.AccountID]),
			CreatedAt:             r.CreatedAt,
			UpdatedAt:             r.UpdatedAt,
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}