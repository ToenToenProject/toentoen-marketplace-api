package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"toentoen-backend/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Client is the global AWS S3 client instance.
var S3Client *s3.Client
var S3PresignClient *s3.PresignClient

// InitS3Client initializes the AWS S3 client session.
func InitS3Client() {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(config.Envs.AWSRegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.Envs.AWSAccessKey,
			config.Envs.AWSSecretKey,
			"",
		)),
	)
	if err != nil {
		panic(fmt.Sprintf("Gagal menginisialisasi AWS S3: %v", err))
	}

	S3Client = s3.NewFromConfig(cfg)
	S3PresignClient = s3.NewPresignClient(S3Client)
}

// PresignUploadResult contains the pre-signed S3 upload URL and public retrieval URL.
type PresignUploadResult struct {
	PresignedURL string `json:"presigned_url"` // URL untuk di-PUT oleh client
	PublicURL    string `json:"public_url"`    // URL publik permanen setelah upload selesai
	Key          string `json:"key"`           // Path/key di dalam bucket S3
}

func extensionFromContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	default:
		return ""
	}
}

// GeneratePresignedUploadURL generates a temporary presigned PUT URL for direct client-to-S3 uploads.
func GeneratePresignedUploadURL(folder, filename, contentType string, contentLength *int64) (*PresignUploadResult, error) {
	folder = strings.Trim(folder, "/")

	// Generate server-side UUID filename with extension derived strictly from validated MIME type.
	ext := extensionFromContentType(contentType)
	key := fmt.Sprintf("%s/%d-%s%s", folder, time.Now().UnixNano(), uuid.NewString(), ext)

	bucket := config.Envs.AWSRegion
	bucketName := config.Envs.AWSS3Bucket

	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}
	if contentLength != nil && *contentLength > 0 {
		putInput.ContentLength = contentLength
	}

	// 15-minute presigned PUT URL.
	presignResult, err := S3PresignClient.PresignPutObject(context.TODO(),
		putInput,
		s3.WithPresignExpires(15*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat presigned URL: %w", err)
	}

	// Permanent public URL.
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, bucket, key)

	return &PresignUploadResult{
		PresignedURL: presignResult.URL,
		PublicURL:    publicURL,
		Key:          key,
	}, nil
}
