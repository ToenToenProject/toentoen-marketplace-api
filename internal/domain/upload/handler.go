package upload

import (
	"strings"
	"toentoen-backend/internal/constants"
	"toentoen-backend/pkg/storage"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// Folder yang diizinkan untuk upload — mencegah client upload ke sembarang folder
var allowedFolders = map[string]bool{
	"businesses/logos":   true,
	"businesses/images":  true,
	"accounts/avatars":   true,
	"products/images":    true,
	"posts/images":       true,
	"reviews/images":     true,
	"demands/attachments": true,
}

// allowedContentTypes — mencegah upload file berbahaya
var allowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// GeneratePresignURL membuat URL sementara untuk upload file langsung ke S3
// POST /api/v1/upload/presign
func (h *Handler) GeneratePresignURL(c *fiber.Ctx) error {
	var req struct {
		Folder      string `json:"folder"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Format data tidak valid", nil)
	}

	// Validasi folder
	folder := strings.TrimSpace(req.Folder)
	if folder == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'folder' wajib diisi", nil)
	}
	if !allowedFolders[folder] {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			"Folder tidak valid. Pilihan: businesses/logos, businesses/images, accounts/avatars, products/images, posts/images, reviews/images, demands/attachments", nil)
	}

	// Validasi filename
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'filename' wajib diisi", nil)
	}

	// Validasi content type
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'content_type' wajib diisi", nil)
	}
	if !allowedContentTypes[contentType] {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			"Tipe file tidak diizinkan. Hanya JPEG, PNG, WebP, dan GIF yang diperbolehkan", nil)
	}

	// Generate presigned URL dari S3
	result, err := storage.GeneratePresignedUploadURL(folder, filename, contentType)
	if err != nil {
		return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr,
			"Gagal membuat URL upload. Silakan coba lagi", nil)
	}

	return utils.SendResponse(c, fiber.StatusOK, constants.Success, "URL upload berhasil dibuat", fiber.Map{
		"presigned_url": result.PresignedURL,
		"public_url":    result.PublicURL,
		"key":           result.Key,
		"expires_in":    "15 menit",
	})
}
