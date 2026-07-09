package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type PostInteractionService interface {
	TogglePostLike(postID, accountID string) (bool, error)
	AddPostComment(postID, accountID string, input models.CreateCommentInput) error
	GetPostComments(postID string) ([]models.CommentResponse, error)
}

type postInteractionService struct {
	repo PostInteractionRepository
}

func NewPostInteractionService(repo PostInteractionRepository) PostInteractionService {
	return &postInteractionService{repo}
}

// 1. TOGGLE LIKE POSTINGAN
func (s *postInteractionService) TogglePostLike(postID, accountID string) (bool, error) {
	// Validasi tambahan bisa ditaruh di sini (misal: cek apakah post-nya masih aktif)
	return s.repo.TogglePostLike(postID, accountID)
}

// 2. CREATE KOMENTAR / BALASAN (Sihir 1-Level Nesting)
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

	// ==========================================
	// LOGIKA VALIDASI 1-LEVEL NESTING
	// ==========================================
	if input.ReplyToCommentID != nil && *input.ReplyToCommentID != "" {
		parentComment, err := s.repo.GetPostCommentByID(*input.ReplyToCommentID)
		if err != nil {
			return errors.New("komentar yang ingin dibalas tidak ditemukan")
		}

		if parentComment.ReplyToCommentID == nil {
			// KASUS A: Membalas Komentar Utama (Root)
			// Biarkan statusnya sebagai anak langsung
			comment.ReplyToCommentID = input.ReplyToCommentID
		} else {
			// KASUS B: Membalas Balasan (Child membalas Child)
			// PAKSA komentar ini menjadi saudara kandung dari parentComment (menginduk ke Root yang sama)
			comment.ReplyToCommentID = parentComment.ReplyToCommentID
			
			// Tambahkan info mention agar UI tahu siapa yang sedang diajak bicara
			comment.RepliedToAccountID = &parentComment.AccountID
			
			// Asumsi model Account Anda memiliki field 'Name' atau 'FullName'
			if parentComment.Account != nil {
				// Sesuaikan .Name dengan field di struct Account Anda
				// Jika struct Account Anda pakai FullName, ganti jadi parentComment.Account.FullName
				name := parentComment.Account.Fullname 
				comment.RepliedToAccountName = &name
			}
		}
	}

	// Simpan ke database
	if err := s.repo.CreatePostComment(&comment); err != nil {
		return errors.New("gagal menambahkan komentar")
	}

	return nil
}

// 3. GET KOMENTAR (Mengubah Array Datar menjadi JSON Nested)
func (s *postInteractionService) GetPostComments(postID string) ([]models.CommentResponse, error) {
	rawComments, err := s.repo.GetCommentsByPostID(postID)
	if err != nil {
		return nil, errors.New("gagal mengambil data komentar")
	}

	// Map pointer untuk menyusun hierarki
	commentMap := make(map[string]*models.CommentResponse)
	var finalComments []models.CommentResponse

	// Looping 1: Pindahkan data mentah ke bentuk DTO (Format Response Frontend)
	for _, rc := range rawComments {
		// Ambil nama user dari relasi (Preload)
		accName := "User"
		if rc.Account != nil {
			accName = rc.Account.Fullname // Sesuaikan dengan field nama di struct Account
		}

		item := &models.CommentResponse{
			CommentID:            rc.CommentID,
			AccountID:            rc.AccountID,
			AccountName:          accName,
			Content:              rc.Content,
			LikeCount:            rc.LikeCount,
			CreatedAt:            rc.CreatedAt,
			RepliedToAccountID:   rc.RepliedToAccountID,
			RepliedToAccountName: rc.RepliedToAccountName,
			Replies:              []models.CommentResponse{},
		}
		commentMap[rc.CommentID] = item
	}

	// Looping 2: Susun relasi Induk dan Anak
	for _, rc := range rawComments {
		currentItem := commentMap[rc.CommentID]

		if rc.ReplyToCommentID == nil {
			// Jika tidak ada parent, ini adalah komentar utama (Level 1)
			// Kita biarkan dulu di map, nanti ditarik di akhir
			continue
		} else {
			// Jika ini balasan, masukkan ke array Replies milik induknya
			if parentItem, exists := commentMap[*rc.ReplyToCommentID]; exists {
				parentItem.Replies = append(parentItem.Replies, *currentItem)
			}
		}
	}

	// Looping 3: Kumpulkan semua komentar Root (beserta balasan yang sudah bersarang di dalamnya)
	for _, rc := range rawComments {
		if rc.ReplyToCommentID == nil {
			finalComments = append(finalComments, *commentMap[rc.CommentID])
		}
	}

	return finalComments, nil
}