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
	sslMode := config.Envs.DBSSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		config.Envs.DBHost,
		config.Envs.DBUser,
		config.Envs.DBPass,
		config.Envs.DBName,
		config.Envs.DBPort,
		sslMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	log.Println("🚀 Connected to Database successfully")

	if !config.Envs.EnableAutoMigrate {
		log.Println("ℹ️ AutoMigrate dinonaktifkan (ENABLE_AUTO_MIGRATE=false) — menjalankan aplikasi dengan izin least privilege")
		return
	}

	// Auto-migrate models individually so one model issue does not block others
	modelsToMigrate := []interface{}{
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
		&models.Product{},
	}
	for _, m := range modelsToMigrate {
		if err := DB.AutoMigrate(m); err != nil {
			log.Printf("⚠️ AutoMigrate model %T: %v", m, err)
		}
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
		`ALTER TABLE business_admin_assignments ADD COLUMN IF NOT EXISTS is_notify_owner_enabled BOOLEAN DEFAULT true`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.table_constraints 
				WHERE constraint_name = 'fk_product_comments_account' AND table_name = 'product_comments'
			) THEN
				ALTER TABLE product_comments DROP CONSTRAINT fk_product_comments_account;
			END IF;
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_comments') THEN
				ALTER TABLE product_comments 
					ADD CONSTRAINT fk_product_comments_account 
					FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE;
			END IF;
			IF EXISTS (
				SELECT 1 FROM information_schema.table_constraints 
				WHERE constraint_name = 'fk_product_comments_replied_account' AND table_name = 'product_comments'
			) THEN
				ALTER TABLE product_comments DROP CONSTRAINT fk_product_comments_replied_account;
			END IF;
			IF EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'product_comments' AND column_name = 'replied_to_account_id'
			) THEN
				ALTER TABLE product_comments 
					ADD CONSTRAINT fk_product_comments_replied_account 
					FOREIGN KEY (replied_to_account_id) REFERENCES accounts(account_id) ON DELETE SET NULL;
			END IF;
		END $$;`,
	}

	for _, q := range alterQueries {
		if err := DB.Exec(q).Error; err != nil {
			log.Printf("⚠️ DB Migration Exec: %s -> %v", q, err)
		}
	}
}
