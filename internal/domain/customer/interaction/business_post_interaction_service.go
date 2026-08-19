package interaction

import (
	"errors"
	"fmt"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type PostInteractionService interface {
	TogglePostLike(postID, accountID string) (bool, error)
	AddPostComment(postID, accountID string, input models.CreateCommentInput) error
	EditPostComment(postID, commentID, accountID string, input models.UpdateCommentInput) error
	DeletePostComment(postID, commentID, requesterID string) error
	GetPostComments(postID, currentAccountID string) ([]models.CommentResponse, error)
	TogglePostCommentLike(commentID, accountID string) (bool, error)
	ReportPost(postID, accountID string, input models.CreatePostReportInput) error
}

type postInteractionService struct {
	repo         PostInteractionRepository
	businessRepo business.BusinessRepository
	notifService notification.NotificationService
}

func NewPostInteractionService(repo PostInteractionRepository, businessRepo business.BusinessRepository, notifService notification.NotificationService) PostInteractionService {
	return &postInteractionService{repo, businessRepo, notifService}
}

// 1. TOGGLE LIKE POSTINGAN
func (s *postInteractionService) TogglePostLike(postID, accountID string) (bool, error) {
	isLiked, err := s.repo.TogglePostLike(postID, accountID)
	if err == nil && isLiked {
		go func() {
			businessID, errB := s.repo.GetBusinessIDByPostID(postID)
			if errB == nil {
				staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
				data := map[string]interface{}{
					"post_id":     postID,
					"business_id": businessID,
				}
				s.notifService.SendNotification(
					staffIDs,
					accountID,
					"post_like",
					postID,
					"Postingan Anda disukai!",
					"Seseorang menyukai postingan Anda.",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

// 2. CREATE KOMENTAR / BALASAN POSTINGAN
func (s *postInteractionService) AddPostComment(postID, accountID string, input models.CreateCommentInput) error {
	businessID, err := s.repo.GetBusinessIDByPostID(postID)
	if err != nil {
		return errors.New("gagal menemukan toko untuk postingan ini")
	}

	comment := models.BusinessPostComment{
		CommentID:  utils.GenerateUUID("CMT"),
		PostID:     postID,
		BusinessID: businessID,
		AccountID:  accountID,
		Content:    input.Content,
	}

	var parentComment *models.BusinessPostComment
	if input.ReplyToCommentID != nil && *input.ReplyToCommentID != "" {
		parent, errP := s.repo.GetPostCommentByID(*input.ReplyToCommentID)
		if errP != nil {
			return errors.New("komentar yang ingin dibalas tidak ditemukan")
		}
		parentComment = parent

		if parent.ReplyToCommentID == nil {
			comment.ReplyToCommentID = input.ReplyToCommentID
		} else {
			comment.ReplyToCommentID = parent.ReplyToCommentID
			comment.RepliedToAccountID = &parent.AccountID
			if parent.Account != nil {
				name := parent.Account.FirstName + " " + parent.Account.LastName
				comment.RepliedToAccountName = &name
			}
		}
	}

	if err := s.repo.CreatePostComment(&comment); err != nil {
		return errors.New("gagal menambahkan komentar")
	}

	// Kirim Notifikasi
	go func() {
		if input.ReplyToCommentID != nil && parentComment != nil {
			recipientID := parentComment.AccountID
			if comment.RepliedToAccountID != nil && *comment.RepliedToAccountID != "" {
				recipientID = *comment.RepliedToAccountID
			}

			data := map[string]interface{}{
				"post_id":     postID,
				"business_id": businessID,
				"comment_id":  comment.CommentID,
			}
			s.notifService.SendNotification(
				[]string{recipientID},
				accountID,
				"post_reply",
				postID,
				"Balasan Komentar Baru",
				fmt.Sprintf("Ada yang membalas komentar Anda: %s", utils.TruncateString(input.Content, 50)),
				data,
				nil,
			)
		} else {
			staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
			data := map[string]interface{}{
				"post_id":     postID,
				"business_id": businessID,
				"comment_id":  comment.CommentID,
			}
			s.notifService.SendNotification(
				staffIDs,
				accountID,
				"post_comment",
				postID,
				"Komentar Baru!",
				fmt.Sprintf("Ada pelanggan yang mengomentari postingan Anda: %s", utils.TruncateString(input.Content, 50)),
				data,
				nil,
			)
		}
	}()

	return nil
}

// 3. EDIT KOMENTAR POSTINGAN
func (s *postInteractionService) EditPostComment(postID, commentID, accountID string, input models.UpdateCommentInput) error {
	comment, err := s.repo.GetPostCommentByID(commentID)
	if err != nil {
		return errors.New("komentar tidak ditemukan")
	}

	if comment.AccountID != accountID {
		return errors.New("Anda tidak memiliki izin untuk mengedit komentar ini")
	}

	comment.Content = input.Content
	return s.repo.UpdatePostComment(comment)
}

// 4. DELETE KOMENTAR POSTINGAN
func (s *postInteractionService) DeletePostComment(postID, commentID, requesterID string) error {
	comment, err := s.repo.GetPostCommentByID(commentID)
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

	return s.repo.DeletePostComment(commentID)
}

// 5. GET KOMENTAR POSTINGAN (TREE + BADGES)
func (s *postInteractionService) GetPostComments(postID, currentAccountID string) ([]models.CommentResponse, error) {
	rawComments, err := s.repo.GetCommentsByPostID(postID)
	if err != nil {
		return nil, errors.New("gagal mengambil data komentar")
	}

	if len(rawComments) == 0 {
		return []models.CommentResponse{}, nil
	}

	businessID, _ := s.repo.GetBusinessIDByPostID(postID)
	staffIDs, _ := s.businessRepo.GetBusinessStaffAccountIDs(businessID)
	staffMap := make(map[string]bool)
	for _, sid := range staffIDs {
		staffMap[sid] = true
	}

	likedCommentIDs := make(map[string]bool)
	ownerLikedCommentIDs := make(map[string]bool)
	for _, rc := range rawComments {
		likes, _ := s.repo.GetPostCommentLikes(rc.CommentID)
		for _, l := range likes {
			if currentAccountID != "" && l.AccountID == currentAccountID {
				likedCommentIDs[rc.CommentID] = true
			}
			if staffMap[l.AccountID] {
				ownerLikedCommentIDs[rc.CommentID] = true
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
			accAvatar = rc.Account.ProfilePictureUrl
		}

		isByOwner := staffMap[rc.AccountID]
		isOwnerLiked := ownerLikedCommentIDs[rc.CommentID]

		item := &models.CommentResponse{
			CommentID:             rc.CommentID,
			AccountID:             rc.AccountID,
			AccountName:           accName,
			AccountAvatar:         accAvatar,
			Content:               rc.Content,
			LikeCount:             rc.LikeCount,
			IsLikedByMe:           likedCommentIDs[rc.CommentID],
			IsByOwner:             isByOwner,
			IsOwnerLiked:          isOwnerLiked,
			OwnerInteractionLabel: getOwnerInteractionLabel(isOwnerLiked, false, isByOwner),
			CreatedAt:             rc.CreatedAt,
			RepliedToAccountID:    rc.RepliedToAccountID,
			RepliedToAccountName:  rc.RepliedToAccountName,
			Replies:               []models.CommentResponse{},
		}
		commentMap[rc.CommentID] = item
	}

	for _, rc := range rawComments {
		currentItem := commentMap[rc.CommentID]

		if rc.ReplyToCommentID != nil {
			if parentItem, exists := commentMap[*rc.ReplyToCommentID]; exists {
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

	for _, rc := range rawComments {
		if rc.ReplyToCommentID == nil {
			finalComments = append(finalComments, *commentMap[rc.CommentID])
		}
	}

	return finalComments, nil
}

// 6. TOGGLE LIKE KOMENTAR POST
func (s *postInteractionService) TogglePostCommentLike(commentID, accountID string) (bool, error) {
	isLiked, err := s.repo.TogglePostCommentLike(commentID, accountID)
	if err == nil && isLiked {
		go func() {
			comment, errC := s.repo.GetPostCommentByID(commentID)
			if errC == nil && comment.AccountID != accountID {
				data := map[string]interface{}{
					"post_id":    comment.PostID,
					"comment_id": commentID,
				}
				s.notifService.SendNotification(
					[]string{comment.AccountID},
					accountID,
					"post_comment_like",
					comment.PostID,
					"Komentar Anda disukai!",
					"Seseorang menyukai komentar postingan Anda.",
					data,
					nil,
				)
			}
		}()
	}
	return isLiked, err
}

func (s *postInteractionService) ReportPost(postID, accountID string, input models.CreatePostReportInput) error {
	_, err := s.repo.GetBusinessIDByPostID(postID)
	if err != nil {
		return errors.New("postingan tidak ditemukan")
	}

	report := models.BusinessPostReport{
		PostID:      postID,
		AccountID:   accountID,
		Reason:      input.Reason,
		Description: input.Description,
		Status:      "pending",
	}

	for _, ev := range input.Evidence {
		report.Evidence = append(report.Evidence, models.BusinessPostReportEvidence{
			FileURL: ev,
		})
	}

	return s.repo.CreatePostReport(&report)
}