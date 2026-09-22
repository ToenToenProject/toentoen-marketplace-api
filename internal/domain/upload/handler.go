package upload

import (
	"fmt"
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/storage"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// Standar Pembatasan Ukuran File
const (
	Size2MB  int64 = 2 * 1024 * 1024  // 2,097,152 bytes (Foto Profil & Logo)
	Size10MB int64 = 10 * 1024 * 1024 // 10,485,760 bytes (Foto, Banner, Ulasan, & Dokumen KYC/Tiket)
	Size50MB int64 = 50 * 1024 * 1024 // 52,428,800 bytes (Video Produk, Postingan Feed, & Ulasan)
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// MediaLimit mendefinisikan batasan upload per folder dan tipe file
type MediaLimit struct {
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	MaxSizeBytes int64    `json:"max_size_bytes"`
	MaxSizeHuman string   `json:"max_size_human"`
	AllowedTypes []string `json:"allowed_content_types"`
}

// getFolderRule mengembalikan aturan validasi batas ukuran dan MIME types yang diizinkan per folder
func getFolderRule(folder, contentType string) (maxBytes int64, maxHuman string, category string, allowedTypes []string, isValidType bool, isFolderFound bool) {
	folder = strings.Trim(strings.ToLower(folder), "/")
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp"}
	imageAndGifTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"}
	videoTypes := []string{"video/mp4", "video/webm", "video/quicktime"}
	docTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "application/pdf"}
	allMediaTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif", "application/pdf", "video/mp4", "video/webm", "video/quicktime"}

	switch folder {
	// 1. Avatar & Logo (Maksimal 2 MB - Hanya Gambar)
	case "accounts/avatars":
		return Size2MB, "2 MB", "Foto Profil / Avatar", imageTypes, contains(imageTypes, contentType), true
	case "businesses/logos":
		return Size2MB, "2 MB", "Logo Toko / Bisnis", imageTypes, contains(imageTypes, contentType), true

	// 2. Dokumen KYC (Maksimal 10 MB - Gambar & PDF)
	case "accounts/kyc", "kyc/documents":
		return Size10MB, "10 MB", "Dokumen Verifikasi KYC", docTypes, contains(docTypes, contentType), true

	// 3. Foto Produk, Banner, & Postingan (Maksimal 10 MB - Hanya Gambar)
	case "businesses/images":
		return Size10MB, "10 MB", "Foto & Banner Toko", imageAndGifTypes, contains(imageAndGifTypes, contentType), true
	case "products/images", "products/previews":
		return Size10MB, "10 MB", "Foto Produk", imageAndGifTypes, contains(imageAndGifTypes, contentType), true
	case "posts/images":
		return Size10MB, "10 MB", "Foto Postingan Feed Toko", imageAndGifTypes, contains(imageAndGifTypes, contentType), true
	case "reviews/images":
		return Size10MB, "10 MB", "Foto Ulasan Produk", imageTypes, contains(imageTypes, contentType), true

	// 4. Video Produk, Postingan, & Ulasan (Maksimal 50 MB - Hanya Video)
	case "products/videos":
		return Size50MB, "50 MB", "Video Produk", videoTypes, contains(videoTypes, contentType), true
	case "posts/videos":
		return Size50MB, "50 MB", "Video Postingan Feed Toko", videoTypes, contains(videoTypes, contentType), true
	case "reviews/videos":
		return Size50MB, "50 MB", "Video Ulasan Produk", videoTypes, contains(videoTypes, contentType), true

	// 5. Lampiran Chat, Demand Signal, & Tiket Bantuan (Gambar/PDF 10 MB, Video 50 MB)
	case "chat/attachments", "demands/attachments", "tickets/attachments":
		cat := "Lampiran Chat / Permintaan"
		if folder == "tickets/attachments" {
			cat = "Lampiran Tiket Bantuan"
		}
		if contains(videoTypes, contentType) {
			return Size50MB, "50 MB", cat + " (Video)", allMediaTypes, true, true
		}
		return Size10MB, "10 MB", cat + " (Gambar/Dokumen)", allMediaTypes, contains(allMediaTypes, contentType), true

	default:
		return 0, "", "", nil, false, false
	}
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, val) {
			return true
		}
	}
	return false
}

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// GeneratePresignURL membuat URL sementara untuk upload file langsung ke S3 dengan batasan ukuran & format ketat
// POST /api/v1/upload/presign
func (h *Handler) GeneratePresignURL(c *fiber.Ctx) error {
	var req struct {
		Folder      string `json:"folder"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		FileSize    int64  `json:"file_size"` // Ukuran file dalam bytes (wajib diisi)
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	// 1. Validasi folder
	folder := strings.Trim(strings.TrimSpace(req.Folder), "/")
	if folder == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'folder' wajib diisi", nil)
	}

	// 2. Validasi filename
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'filename' wajib diisi", nil)
	}

	// 3. Validasi content type
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'content_type' wajib diisi", nil)
	}

	maxBytes, maxHuman, category, allowedTypes, isValidType, isFolderFound := getFolderRule(folder, contentType)
	if !isFolderFound {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			"Folder upload tidak valid. Pilihan folder resmi: businesses/logos, businesses/images, accounts/avatars, accounts/kyc, kyc/documents, chat/attachments, products/images, products/videos, posts/images, posts/videos, reviews/images, reviews/videos, demands/attachments, tickets/attachments", nil)
	}

	if !isValidType {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			fmt.Sprintf("Format file '%s' tidak diizinkan untuk folder '%s'. Format yang didukung: %s",
				contentType, folder, strings.Join(allowedTypes, ", ")), nil)
	}

	// 4. Validasi ukuran file (file_size)
	if req.FileSize <= 0 {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			"Field 'file_size' wajib diisi (dalam bytes) dan harus lebih besar dari 0", nil)
	}

	if req.FileSize > maxBytes {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			fmt.Sprintf("Ukuran file (%s) melebihi batas maksimum untuk %s (maksimal: %s atau %d bytes)",
				formatBytes(req.FileSize), category, maxHuman, maxBytes), fiber.Map{
				"file_size":        req.FileSize,
				"file_size_human":  formatBytes(req.FileSize),
				"max_size_bytes":   maxBytes,
				"max_size_human":   maxHuman,
				"folder":           folder,
				"category":         category,
			})
	}

	// 5. Generate presigned URL dari S3 dengan mengunci ContentLength
	result, err := storage.GeneratePresignedUploadURL(folder, filename, contentType, &req.FileSize)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr,
			"Gagal membuat URL upload. Silakan coba lagi", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "URL upload berhasil dibuat", fiber.Map{
		"presigned_url":    result.PresignedURL,
		"public_url":       result.PublicURL,
		"key":              result.Key,
		"file_size":        req.FileSize,
		"file_size_human":  formatBytes(req.FileSize),
		"max_size_human":   maxHuman,
		"category":         category,
		"expires_in":       "15 menit",
	})
}

// GetUploadLimits mengembalikan katalog informasi batasan ukuran dan format file upload ToenToen
// GET /api/v1/upload/limits
func (h *Handler) GetUploadLimits(c *fiber.Ctx) error {
	limits := []fiber.Map{
		{
			"folder":                 "accounts/avatars",
			"category":               "Foto Profil / Avatar Pengguna",
			"max_size_bytes":         Size2MB,
			"max_size_human":         "2 MB",
			"allowed_content_types":  []string{"image/jpeg", "image/jpg", "image/png", "image/webp"},
		},
		{
			"folder":                 "businesses/logos",
			"category":               "Logo Toko / Bisnis",
			"max_size_bytes":         Size2MB,
			"max_size_human":         "2 MB",
			"allowed_content_types":  []string{"image/jpeg", "image/jpg", "image/png", "image/webp"},
		},
		{
			"folder":                 "accounts/kyc, kyc/documents",
			"category":               "Dokumen Verifikasi KYC",
			"max_size_bytes":         Size10MB,
			"max_size_human":         "10 MB",
			"allowed_content_types":  []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "application/pdf"},
		},
		{
			"folder":                 "products/images, products/previews, businesses/images, posts/images",
			"category":               "Foto Produk, Banner Toko, & Postingan Feed",
			"max_size_bytes":         Size10MB,
			"max_size_human":         "10 MB",
			"allowed_content_types":  []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"},
		},
		{
			"folder":                 "reviews/images",
			"category":               "Foto Ulasan Produk",
			"max_size_bytes":         Size10MB,
			"max_size_human":         "10 MB",
			"allowed_content_types":  []string{"image/jpeg", "image/jpg", "image/png", "image/webp"},
		},
		{
			"folder":                 "products/videos, posts/videos, reviews/videos",
			"category":               "Video Produk, Postingan Feed, & Ulasan",
			"max_size_bytes":         Size50MB,
			"max_size_human":         "50 MB",
			"allowed_content_types":  []string{"video/mp4", "video/webm", "video/quicktime"},
		},
		{
			"folder":                 "chat/attachments, demands/attachments, tickets/attachments",
			"category":               "Lampiran Chat, Demand Signal, & Tiket Bantuan",
			"max_size_bytes_media":   "10 MB untuk Gambar/Dokumen (PDF), 50 MB untuk Video",
			"allowed_content_types":  []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif", "application/pdf", "video/mp4", "video/webm", "video/quicktime"},
		},
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Daftar batasan upload media ToenToen", fiber.Map{
		"limits": limits,
	})
}
