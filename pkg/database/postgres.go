package database

import (
	"fmt"
	"log"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	// Setup DNS
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		config.Envs.DBHost,
		config.Envs.DBUser,
		config.Envs.DBPass,
		config.Envs.DBName,
		config.Envs.DBPort,
	)

	// Connect to database
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	// Handle connection error
	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	err = DB.AutoMigrate(
		&models.Category{},
		&models.Tag{},
		&models.Account{}, // Tambahkan ini di atas agar GORM tahu tabel utama
		&models.BusinessBookmark{},
		&models.BusinessReport{},
		&models.BusinessReportEvidence{},
		&models.BusinessPostReport{},
		&models.BusinessPostReportEvidence{},
		&models.Review{},
		&models.ProductReview{},
		&models.ProductLike{},
		&models.ProductReviewLike{},
		&models.ProductReviewReport{},
		&models.BusinessPostComment{},
		&models.BusinessPostLike{},
		&models.BusinessReviewLike{},
		&models.BusinessReviewComment{},
		&models.BusinessReviewCommentLike{},
		&models.BusinessReviewReport{},
		&models.BusinessReviewCommentReport{},
		&models.ProductWishlist{},

		// Signal V2
		&models.Signal{},
		&models.SignalImage{},
		&models.SignalConversation{},
		&models.SignalMessage{},
		
		// Notifications
		&models.UserNotification{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database. \n", err)
	}

	log.Println("🚀 Connected to Database successfully")
}
