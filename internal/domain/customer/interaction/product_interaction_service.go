package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductInteractionService interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	AddProductComment(productID, accountID string, input models.CreateCommentInput) error
	GetProductComments(productID string) ([]models.CommentResponse, error)
}

type productInteractionService struct {
	repo ProductInteractionRepository
}

func NewProductInteractionService(repo ProductInteractionRepository) ProductInteractionService {
	return &productInteractionService{repo}
}

// 1. TOGGLE LIKE PRODUK
func (s *productInteractionService) ToggleProductLike(productID, accountID string) (bool, error) {
	return s.repo.ToggleProductLike(productID, accountID)
}

// 2. CREATE KOMENTAR / BALASAN PRODUK (Logika 1-Level Nesting)
func (s *productInteractionService) AddProductComment(productID, accountID string, input models.CreateCommentInput) error {
	
	businessID, err := s.repo.GetBusinessIDByProductID(productID)
	if err != nil {
		return errors.New("gagal menemukan toko untuk produk ini")
	}

	// 🪄 RAKIT STRUCT KOMENTARNYA
	comment := models.ProductComment{
		CommentID:  utils.GenerateUUID("CMT"),
		ProductID:  productID,
		BusinessID: businessID, // 👈 Hasil pencarian di atas disuntikkan ke sini!
		AccountID:  accountID,
		Content:    input.Content,
	}

	// ==========================================
	// VALIDASI & PEMOTONGAN LEVEL KOMENTAR
	// ==========================================
	if input.ReplyToCommentID != nil && *input.ReplyToCommentID != "" {
		parentComment, err := s.repo.GetProductCommentByID(*input.ReplyToCommentID)
		if err != nil {
			return errors.New("komentar yang ingin dibalas tidak ditemukan")
		}

		if parentComment.ReplyToCommentID == nil {
			// Membalas Komentar Utama
			comment.ReplyToCommentID = input.ReplyToCommentID
		} else {
			// Membalas Balasan (Paksa sejajar)
			comment.ReplyToCommentID = parentComment.ReplyToCommentID
			comment.RepliedToAccountID = &parentComment.AccountID
			if parentComment.Account != nil {
				name := parentComment.Account.Fullname 
				comment.RepliedToAccountName = &name
			}
		}
	}

	// Simpan ke database
	if err := s.repo.CreateProductComment(&comment); err != nil {
		return errors.New("gagal menambahkan komentar produk")
	}

	return nil
}

// 3. GET KOMENTAR PRODUK (Mapping Array Datar ke JSON Bersarang)
func (s *productInteractionService) GetProductComments(productID string) ([]models.CommentResponse, error) {
	// 1. Tarik semua data mentah dari database (sudah diurutkan dari yang terlama)
	rawComments, err := s.repo.GetCommentsByProductID(productID)
	if err != nil {
		return nil, errors.New("gagal mengambil data komentar produk")
	}

	// Jika tidak ada komentar, kembalikan array kosong agar JSON rapi (tidak null)
	if len(rawComments) == 0 {
		return []models.CommentResponse{}, nil
	}

	// Gunakan Map untuk mempercepat pencarian data saat merakit balasan
	commentMap := make(map[string]*models.CommentResponse)
	var finalComments []models.CommentResponse

	// Looping Tahap 1: Pindahkan data mentah ke DTO Response & masukkan ke Map
	for _, rc := range rawComments {
		accName := "Pengguna" // Default nama jika relasi Account tidak ditemukan
		if rc.Account != nil {
			accName = rc.Account.Fullname // Sesuaikan 'Name' dengan field di struct Account Anda
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
			Replies:              []models.CommentResponse{}, // Inisialisasi array kosong
		}
		commentMap[rc.CommentID] = item
	}

	// Looping Tahap 2: Masukkan balasan (Level 2) ke dalam array Replies milik parent masing-masing
	for _, rc := range rawComments {
		currentItem := commentMap[rc.CommentID]

		// Jika ini adalah balasan (punya parent)
		if rc.ReplyToCommentID != nil {
			if parentItem, exists := commentMap[*rc.ReplyToCommentID]; exists {
				parentItem.Replies = append(parentItem.Replies, *currentItem)
			}
		}
	}

	// Looping Tahap 3: Kumpulkan HANYA komentar Utama (Level 1) untuk dikembalikan
	for _, rc := range rawComments {
		if rc.ReplyToCommentID == nil {
			finalComments = append(finalComments, *commentMap[rc.CommentID])
		}
	}

	return finalComments, nil
}