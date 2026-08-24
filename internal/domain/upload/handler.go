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
	"businesses/logos":    true,
	"businesses/images":   true,
	"accounts/avatars":    true,
	"products/images":     true,
	"products/videos":     true,
	"posts/images":        true,
	"posts/videos":        true,
	"reviews/images":      true,
	"reviews/videos":      true,
	"demands/attachments": true,
	"tickets/attachments": true,
}

// allowedContentTypes — mencegah upload file berbahaya
var allowedContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/jpg":       true,
	"image/png":       true,
	"image/webp":      true,
	"image/gif":       true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/quicktime": true, // Untuk perekaman dari iOS (.mov)
	"application/pdf": true, // Untuk dokumen tiket bantuan
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

	if errors := utils.ValidateStruct(&req); errors != nil {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, utils.FormatValidationErrors(errors), errors)
	}

	// Validasi folder
	folder := strings.TrimSpace(req.Folder)
	if folder == "" {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "Field 'folder' wajib diisi", nil)
	}
	if !allowedFolders[folder] {
		return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation,
			"Folder tidak valid. Pilihan: businesses/logos, businesses/images, accounts/avatars, products/images, products/videos, posts/images, posts/videos, reviews/images, reviews/videos, demands/attachments", nil)
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
			"Tipe file tidak diizinkan. Hanya JPEG, PNG, WebP, GIF, MP4, WebM, dan MOV yang diperbolehkan", nil)
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
