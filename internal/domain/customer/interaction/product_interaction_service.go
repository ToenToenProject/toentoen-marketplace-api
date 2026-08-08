package interaction

import (
	"errors"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductInteractionService interface {
	ToggleProductLike(productID, accountID string) (bool, error)
	AddProductComment(productID, accountID string, input models.CreateCommentInput) error
	GetProductComments(productID, currentAccountID string) ([]models.CommentResponse, error)
	GetAllProductCommentsByBusinessID(businessID string) ([]models.CommentResponse, error)
	ToggleCommentLike(commentID, accountID string) (bool, error)
	ReportComment(commentID, accountID, reason string) error
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
		Rating:     input.Rating,
		MediaURLs:  input.MediaURLs,
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
func (s *productInteractionService) GetProductComments(productID, currentAccountID string) ([]models.CommentResponse, error) {
	// 1. Tarik semua data mentah dari database (sudah diurutkan dari yang terlama)
	rawComments, err := s.repo.GetCommentsByProductID(productID)
	if err != nil {
		return nil, errors.New("gagal mengambil data komentar produk")
	}

	if len(rawComments) == 0 {
		return []models.CommentResponse{}, nil
	}

	// Buat himpunan (Set) komentar mana saja yang dilike oleh user yang sedang login
	likedCommentIDs := make(map[string]bool)
	if currentAccountID != "" {
		// Ambil semua like pada seluruh komentar ini oleh user ini
		// Karena repot ambil 1-1, kita loop saja (bisa lebih dioptimasi nanti dengan query IN)
		// atau cukup cek di DB
		// Tapi untuk saat ini, kita ambil likes per comment saja jika butuh, atau kita query manual.
		// Untuk efisiensi, abaikan dulu query batch, lakukan loop cek
		for _, rc := range rawComments {
			likes, _ := s.repo.GetCommentLikes(rc.CommentID)
			for _, l := range likes {
				if l.AccountID == currentAccountID {
					likedCommentIDs[rc.CommentID] = true
					break
				}
			}
		}
	}

	// Gunakan Map untuk mempercepat pencarian data saat merakit balasan
	commentMap := make(map[string]*models.CommentResponse)
	var finalComments []models.CommentResponse

	// Looping Tahap 1: Pindahkan data mentah ke DTO Response & masukkan ke Map
	for _, rc := range rawComments {
		accName := "Pengguna" // Default nama jika relasi Account tidak ditemukan
		var accAvatar *string
		if rc.Account != nil {
			accName = rc.Account.Fullname // Sesuaikan 'Name' dengan field di struct Account Anda
			if rc.Account.Customer != nil {
				accAvatar = rc.Account.Customer.ProfilePictureUrl
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
			Rating:               rc.Rating,
			MediaURLs:            rc.MediaURLs,
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
				parentItem.RepliesCount = len(parentItem.Replies)
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

func (s *productInteractionService) GetAllProductCommentsByBusinessID(businessID string) ([]models.CommentResponse, error) {
	rawComments, err := s.repo.GetAllProductCommentsByBusinessID(businessID)
	if err != nil {
		return nil, errors.New("gagal mengambil review produk")
	}

	var finalComments []models.CommentResponse
	for _, rc := range rawComments {
		accName := "Pengguna"
		var accAvatar *string
		if rc.Account != nil {
			accName = rc.Account.Fullname
			if rc.Account.Customer != nil {
				accAvatar = rc.Account.Customer.ProfilePictureUrl
			}
		}

		item := models.CommentResponse{
			CommentID:            rc.CommentID,
			AccountID:            rc.AccountID,
			AccountName:          accName,
			AccountAvatar:        accAvatar,
			Content:              rc.Content,
			LikeCount:            rc.LikeCount,
			IsLikedByMe:          false, // we might not know unless we pass currentAccountID, for now keep simple or pass it later
			Rating:               rc.Rating,
			MediaURLs:            rc.MediaURLs,
			CreatedAt:            rc.CreatedAt,
			RepliedToAccountID:   rc.RepliedToAccountID,
			RepliedToAccountName: rc.RepliedToAccountName,
			RepliesCount:         len(rc.Replies),
		}

		if rc.Product != nil {
			item.ProductID = &rc.Product.ProductID
			item.ProductName = &rc.Product.ProductName
			item.ProductPrice = &rc.Product.Price
			if rc.Product.Discount > 0 {
				orig := rc.Product.Price + rc.Product.Discount
				item.ProductOriginalPrice = &orig
			}
			if len(rc.Product.ProductImages) > 0 {
				item.ProductImage = &rc.Product.ProductImages[0]
			}
		}
		finalComments = append(finalComments, item)
	}

	return finalComments, nil
}

func (s *productInteractionService) ToggleCommentLike(commentID, accountID string) (bool, error) {
	return s.repo.ToggleCommentLike(commentID, accountID)
}

func (s *productInteractionService) ReportComment(commentID, accountID, reason string) error {
	report := models.ProductCommentReport{
		AccountID: accountID,
		CommentID: commentID,
		Reason:    reason,
	}
	return s.repo.CreateCommentReport(&report)
}