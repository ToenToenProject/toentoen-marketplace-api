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
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		config.Envs.DBHost,
		config.Envs.DBUser,
		config.Envs.DBPass,
		config.Envs.DBName,
		config.Envs.DBPort,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	log.Println("🚀 Connected to Database successfully")

	// Auto-migrate models
	if err := DB.AutoMigrate(
		&models.Owner{},
		&models.BusinessAdmin{},
		&models.CSChatSession{},
		&models.CSChatMessage{},
		&models.SupportTicket{},
		&models.SupportTicketAttachment{},
		&models.DirectBusinessConversation{},
		&models.DirectBusinessMessage{},
		&models.BusinessTransferRequest{},
		&models.UserPostInteraction{},
		&models.UserCategoryAffinity{},
		&models.Signal{},
		&models.SignalTag{},
		&models.SignalImage{},
		&models.SignalConversation{},
		&models.SignalMessage{},
	); err != nil {
		log.Printf("⚠️ AutoMigrate models: %v", err)
	}

	// Ensure columns in owners, business_admins, and businesses can store encrypted ciphertext (TEXT)
	alterQueries := []string{
		`ALTER TABLE owners ALTER COLUMN ktp_number TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_photo_url TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_name TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_birth_place TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_gender TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_address TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_rt_rw TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_village TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_district TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_religion TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_occupation TYPE TEXT`,
		`ALTER TABLE owners ALTER COLUMN ktp_citizenship TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_number TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_photo_url TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_name TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_birth_place TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_gender TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_address TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_rt_rw TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_village TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_district TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_religion TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_occupation TYPE TEXT`,
		`ALTER TABLE business_admins ALTER COLUMN ktp_citizenship TYPE TEXT`,
		`ALTER TABLE businesses ALTER COLUMN nib TYPE TEXT`,
		`ALTER TABLE signals ADD COLUMN IF NOT EXISTS min_budget NUMERIC`,
		`ALTER TABLE signals ADD COLUMN IF NOT EXISTS max_budget NUMERIC`,
		`ALTER TABLE signals ADD COLUMN IF NOT EXISTS max_offers INTEGER DEFAULT 5`,
		`ALTER TABLE signals ADD COLUMN IF NOT EXISTS offer_count INTEGER DEFAULT 0`,
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS is_live BOOLEAN DEFAULT false`,
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS live_platform VARCHAR(50)`,
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS live_started_at TIMESTAMPTZ`,
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS last_live_notified_at TIMESTAMPTZ`,
		`ALTER TABLE accounts ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMP`,
		`ALTER TABLE accounts ADD COLUMN IF NOT EXISTS phone_verified_at TIMESTAMP`,
		`ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_status VARCHAR(20) NOT NULL DEFAULT 'unverified'`,
		`ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_rejection_reason TEXT`,
		`ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_submitted_at TIMESTAMP`,
		`ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_verified_at TIMESTAMP`,
		`ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_status VARCHAR(20) NOT NULL DEFAULT 'unverified'`,
		`ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_rejection_reason TEXT`,
		`ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_submitted_at TIMESTAMP`,
		`ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_verified_at TIMESTAMP`,
	}

	for _, q := range alterQueries {
		if err := DB.Exec(q).Error; err != nil {
			log.Printf("⚠️ DB Migration Exec: %s -> %v", q, err)
		}
	}
}
