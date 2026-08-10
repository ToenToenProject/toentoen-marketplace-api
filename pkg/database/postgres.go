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
		&models.Account{}, // Tambahkan ini di atas agar GORM tahu tabel utama
		&models.BusinessBookmark{},
		&models.BusinessReport{},
		&models.BusinessReportEvidence{},
		&models.BusinessPostReport{},
		&models.BusinessPostReportEvidence{},
		&models.Review{},
		&models.ProductComment{},
		&models.ProductLike{},
		&models.ProductCommentLike{},
		&models.ProductCommentReport{},
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
	)
	if err != nil {
		log.Fatal("Failed to migrate database. \n", err)
	}

	// Manual Migration for Products (Aman dari bentrok GORM Relations)
	DB.Exec("ALTER TABLE products ADD COLUMN IF NOT EXISTS product_origin varchar(50) DEFAULT 'domestic'")
	DB.Exec("ALTER TABLE products ADD COLUMN IF NOT EXISTS imported_from varchar(100)")

	// PostGIS: Pastikan ekstensi postgis aktif sebelum membuat kolom location
	DB.Exec("CREATE EXTENSION IF NOT EXISTS postgis")
	DB.Exec("ALTER TABLE signals ADD COLUMN IF NOT EXISTS location geometry(Point, 4326)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_signals_location ON signals USING GIST(location)")

	log.Println("🚀 Connected to Database successfully")
}