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

	// Bersihkan data orphan pada tabel signal jika ada data bisnis yang sudah terhapus sebelumnya
	_ = DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_conversations') AND
			   EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'businesses') THEN
				IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_messages') THEN
					DELETE FROM signal_messages WHERE conversation_id IN (
						SELECT conversation_id FROM signal_conversations WHERE business_id NOT IN (SELECT business_id FROM businesses)
					);
				END IF;
				DELETE FROM signal_conversations WHERE business_id NOT IN (SELECT business_id FROM businesses);
			END IF;
		END $$;
	`)

	err = DB.AutoMigrate(
		&models.Category{},
		&models.Tag{},
		&models.Account{}, // Tambahkan ini di atas agar GORM tahu tabel utama
		&models.Business{}, // Tambahkan Business agar kolom barunya ter-AutoMigrate
		&models.BusinessHour{},
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
