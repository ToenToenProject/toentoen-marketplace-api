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

// GeneratePresignedUploadURL generates a temporary presigned PUT URL for direct client-to-S3 uploads.
func GeneratePresignedUploadURL(folder, filename, contentType string, contentLength *int64) (*PresignUploadResult, error) {
	folder = strings.Trim(folder, "/")

	// Build unique S3 key with timestamp prefix to prevent object collisions.
	timestamp := time.Now().UnixNano()
	safeFilename := strings.ReplaceAll(filename, " ", "-")
	key := fmt.Sprintf("%s/%d-%s", folder, timestamp, safeFilename)

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
