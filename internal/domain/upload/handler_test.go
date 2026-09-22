package upload

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "500 B", formatBytes(500))
	assert.Equal(t, "1.0 MB", formatBytes(1*1024*1024))
	assert.Equal(t, "2.0 MB", formatBytes(2*1024*1024))
	assert.Equal(t, "10.0 MB", formatBytes(10*1024*1024))
	assert.Equal(t, "50.0 MB", formatBytes(50*1024*1024))
}

func TestGetFolderRule(t *testing.T) {
	// 1. Avatar (2 MB, image only)
	maxBytes, maxHuman, _, _, isValidType, isFolderFound := getFolderRule("accounts/avatars", "image/jpeg")
	assert.True(t, isFolderFound)
	assert.True(t, isValidType)
	assert.Equal(t, Size2MB, maxBytes)
	assert.Equal(t, "2 MB", maxHuman)

	// Video to avatar should fail
	_, _, _, _, isValidType, isFolderFound = getFolderRule("accounts/avatars", "video/mp4")
	assert.True(t, isFolderFound)
	assert.False(t, isValidType)

	// 2. KYC (10 MB, image & pdf)
	maxBytes, _, _, _, isValidType, isFolderFound = getFolderRule("accounts/kyc", "application/pdf")
	assert.True(t, isFolderFound)
	assert.True(t, isValidType)
	assert.Equal(t, Size10MB, maxBytes)

	// 3. Products Images (10 MB)
	maxBytes, _, _, _, isValidType, isFolderFound = getFolderRule("products/images", "image/png")
	assert.True(t, isFolderFound)
	assert.True(t, isValidType)
	assert.Equal(t, Size10MB, maxBytes)

	// 4. Products Videos (50 MB)
	maxBytes, _, _, _, isValidType, isFolderFound = getFolderRule("products/videos", "video/mp4")
	assert.True(t, isFolderFound)
	assert.True(t, isValidType)
	assert.Equal(t, Size50MB, maxBytes)

	// Image to product video folder should fail
	_, _, _, _, isValidType, isFolderFound = getFolderRule("products/videos", "image/png")
	assert.True(t, isFolderFound)
	assert.False(t, isValidType)

	// 5. Chat attachments (10 MB image, 50 MB video)
	maxBytesImg, _, _, _, isValidType, _ := getFolderRule("chat/attachments", "image/jpeg")
	assert.True(t, isValidType)
	assert.Equal(t, Size10MB, maxBytesImg)

	maxBytesVid, _, _, _, isValidType, _ := getFolderRule("chat/attachments", "video/mp4")
	assert.True(t, isValidType)
	assert.Equal(t, Size50MB, maxBytesVid)

	// 6. Unknown folder
	_, _, _, _, _, isFolderFound = getFolderRule("random/unknown", "image/jpeg")
	assert.False(t, isFolderFound)
}

func TestUploadValidationEndpoints(t *testing.T) {
	app := fiber.New()
	h := NewHandler()

	app.Post("/upload/presign", h.GeneratePresignURL)
	app.Get("/upload/limits", h.GetUploadLimits)

	// Case 1: Get limits
	reqLimits := httptest.NewRequest(http.MethodGet, "/upload/limits", nil)
	respLimits, err := app.Test(reqLimits)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respLimits.StatusCode)

	// Case 2: Missing file_size or file_size <= 0
	bodyNoSize, _ := json.Marshal(map[string]interface{}{
		"folder":       "accounts/avatars",
		"filename":     "avatar.png",
		"content_type": "image/png",
	})
	reqNoSize := httptest.NewRequest(http.MethodPost, "/upload/presign", bytes.NewReader(bodyNoSize))
	reqNoSize.Header.Set("Content-Type", "application/json")
	respNoSize, err := app.Test(reqNoSize)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, respNoSize.StatusCode)

	// Case 3: Oversized avatar (e.g. 5 MB when max is 2 MB)
	bodyOver, _ := json.Marshal(map[string]interface{}{
		"folder":       "accounts/avatars",
		"filename":     "avatar.png",
		"content_type": "image/png",
		"file_size":    5 * 1024 * 1024,
	})
	reqOver := httptest.NewRequest(http.MethodPost, "/upload/presign", bytes.NewReader(bodyOver))
	reqOver.Header.Set("Content-Type", "application/json")
	respOver, err := app.Test(reqOver)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, respOver.StatusCode)

	// Case 4: Invalid content type for folder (e.g. mp4 in accounts/avatars)
	bodyInvalidType, _ := json.Marshal(map[string]interface{}{
		"folder":       "accounts/avatars",
		"filename":     "video.mp4",
		"content_type": "video/mp4",
		"file_size":    1024,
	})
	reqInvalidType := httptest.NewRequest(http.MethodPost, "/upload/presign", bytes.NewReader(bodyInvalidType))
	reqInvalidType.Header.Set("Content-Type", "application/json")
	respInvalidType, err := app.Test(reqInvalidType)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, respInvalidType.StatusCode)

	// Case 5: Invalid folder
	bodyInvalidFolder, _ := json.Marshal(map[string]interface{}{
		"folder":       "hack/root",
		"filename":     "script.sh",
		"content_type": "application/x-sh",
		"file_size":    1024,
	})
	reqInvalidFolder := httptest.NewRequest(http.MethodPost, "/upload/presign", bytes.NewReader(bodyInvalidFolder))
	reqInvalidFolder.Header.Set("Content-Type", "application/json")
	respInvalidFolder, err := app.Test(reqInvalidFolder)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, respInvalidFolder.StatusCode)
}
