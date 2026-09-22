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

// S3Client adalah client global yang dipakai seluruh aplikasi
var S3Client *s3.Client
var S3PresignClient *s3.PresignClient

// InitS3Client menginisialisasi koneksi ke AWS S3 menggunakan kredensial dari config
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

// PresignUploadResult adalah hasil dari GeneratePresignedUploadURL
type PresignUploadResult struct {
	PresignedURL string `json:"presigned_url"` // URL untuk di-PUT oleh client
	PublicURL    string `json:"public_url"`    // URL publik permanen setelah upload selesai
	Key          string `json:"key"`           // Path/key di dalam bucket S3
}

// GeneratePresignedUploadURL membuat URL sementara agar client bisa upload langsung ke S3
// folder: folder di dalam bucket, misal "products/images"
// filename: nama file asli, misal "foto-produk.jpg"
// contentType: MIME type, misal "image/jpeg"
// contentLength: ukuran file dalam bytes (opsional, jika diisi akan mengunci batas ukuran di S3)
func GeneratePresignedUploadURL(folder, filename, contentType string, contentLength *int64) (*PresignUploadResult, error) {
	// Bersihkan folder dari slash berlebih
	folder = strings.Trim(folder, "/")

	// Buat key unik menggunakan timestamp agar tidak bertabrakan
	timestamp := time.Now().UnixNano()
	// Bersihkan nama file dari karakter berbahaya
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

	// Buat presigned URL yang berlaku selama 15 menit
	presignResult, err := S3PresignClient.PresignPutObject(context.TODO(),
		putInput,
		s3.WithPresignExpires(15*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat presigned URL: %w", err)
	}

	// URL publik permanen setelah file berhasil diupload
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, bucket, key)

	return &PresignUploadResult{
		PresignedURL: presignResult.URL,
		PublicURL:    publicURL,
		Key:          key,
	}, nil
}
