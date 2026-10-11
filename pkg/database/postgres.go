package database

import (
	"fmt"
	"log"
	"time"

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

	// Konfigurasi Connection Pool Skala Industri (Optimal untuk AWS RDS & Auto-scaling)
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to retrieve generic database object from GORM: %v", err)
	}

	maxOpen := config.Envs.DBMaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 50
	}
	maxIdle := config.Envs.DBMaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 15
	}
	connMaxLifetime := time.Duration(config.Envs.DBConnMaxLifetimeMinutes) * time.Minute
	if connMaxLifetime <= 0 {
		connMaxLifetime = 30 * time.Minute
	}
	connMaxIdleTime := time.Duration(config.Envs.DBConnMaxIdleTimeMinutes) * time.Minute
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 5 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	log.Printf("🔌 Database connection pool initialized: max_open=%d, max_idle=%d, lifetime=%v, idle_time=%v",
		maxOpen, maxIdle, connMaxLifetime, connMaxIdleTime)


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
		&models.SignalCategory{},
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
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS certificates jsonb DEFAULT '[]'::jsonb`,
		`UPDATE businesses SET certificates = '[]'::jsonb WHERE certificates IS NULL`,
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
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS confirmed_open_date DATE`,
		`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS confirmed_overtime_date DATE`,
		`ALTER TABLE reviews ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ`,
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
		`DO $$
		DECLARE
			r RECORD;
		BEGIN
			-- Cascade delete untuk bukti laporan bisnis (business_report_evidences -> business_reports)
			FOR r IN (
				SELECT tc.constraint_name 
				FROM information_schema.table_constraints tc 
				JOIN information_schema.key_column_usage kcu 
				  ON tc.constraint_name = kcu.constraint_name 
				 AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'FOREIGN KEY' 
				  AND tc.table_name = 'business_report_evidences' 
				  AND kcu.column_name = 'report_id'
			) LOOP
				EXECUTE 'ALTER TABLE business_report_evidences DROP CONSTRAINT ' || quote_ident(r.constraint_name);
			END LOOP;

			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'business_report_evidences') THEN
				ALTER TABLE business_report_evidences 
					ADD CONSTRAINT fk_business_reports_evidence 
					FOREIGN KEY (report_id) REFERENCES business_reports(report_id) ON DELETE CASCADE;
			END IF;

			-- Cascade delete untuk bukti laporan post (business_post_report_evidences -> business_post_reports)
			FOR r IN (
				SELECT tc.constraint_name 
				FROM information_schema.table_constraints tc 
				JOIN information_schema.key_column_usage kcu 
				  ON tc.constraint_name = kcu.constraint_name 
				 AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'FOREIGN KEY' 
				  AND tc.table_name = 'business_post_report_evidences' 
				  AND kcu.column_name = 'report_id'
			) LOOP
				EXECUTE 'ALTER TABLE business_post_report_evidences DROP CONSTRAINT ' || quote_ident(r.constraint_name);
			END LOOP;

			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'business_post_report_evidences') THEN
				ALTER TABLE business_post_report_evidences 
					ADD CONSTRAINT fk_business_post_reports_evidence 
					FOREIGN KEY (report_id) REFERENCES business_post_reports(report_id) ON DELETE CASCADE;
			END IF;
		END $$;`,
		`DO $$
		DECLARE
			r RECORD;
		BEGIN
			-- Hapus seluruh constraint UNIQUE global lama pada kolom email dan phone_number di tabel accounts
			FOR r IN (
				SELECT tc.constraint_name 
				FROM information_schema.table_constraints tc 
				JOIN information_schema.key_column_usage kcu 
				  ON tc.constraint_name = kcu.constraint_name 
				 AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'UNIQUE' 
				  AND tc.table_name = 'accounts' 
				  AND kcu.column_name IN ('email', 'phone_number')
			) LOOP
				EXECUTE 'ALTER TABLE accounts DROP CONSTRAINT IF EXISTS ' || quote_ident(r.constraint_name);
			END LOOP;

			-- Hapus indeks unik lama jika dibuat sebagai standalone index
			DROP INDEX IF EXISTS idx_accounts_email;
			DROP INDEX IF EXISTS idx_accounts_phone_number;
			DROP INDEX IF EXISTS accounts_email_key;
			DROP INDEX IF EXISTS accounts_phone_number_key;
			DROP INDEX IF EXISTS uq_accounts_email;
			DROP INDEX IF EXISTS uq_accounts_phone;

			-- Buat Partial Unique Index per kelompok role:
			-- 1) Kelompok Customer: 1 email unik hanya untuk akun customer
			CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_email 
				ON accounts (LOWER(email)) 
				WHERE role = 'customer' AND email IS NOT NULL AND email != '';

			CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_phone 
				ON accounts (phone_number) 
				WHERE role = 'customer' AND phone_number IS NOT NULL AND phone_number != '';

			-- 2) Kelompok Merchant: 1 email unik untuk owner & business_admin (owner & admin tetap tidak boleh berbagi email)
			CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_merchant_email 
				ON accounts (LOWER(email)) 
				WHERE role IN ('owner', 'business_admin') AND email IS NOT NULL AND email != '';

			CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_merchant_phone 
				ON accounts (phone_number) 
				WHERE role IN ('owner', 'business_admin') AND phone_number IS NOT NULL AND phone_number != '';
		END $$;`,
		`CREATE TABLE IF NOT EXISTS signal_categories (
			signal_id VARCHAR(50) NOT NULL REFERENCES signals(signal_id) ON DELETE CASCADE,
			category_id BIGINT NOT NULL REFERENCES business_categories(category_id) ON DELETE CASCADE,
			PRIMARY KEY (signal_id, category_id)
		);
		CREATE INDEX IF NOT EXISTS idx_signal_categories_category_id ON signal_categories(category_id);
		INSERT INTO signal_categories (signal_id, category_id)
		SELECT signal_id, category_id FROM signals
		WHERE category_id IS NOT NULL
		ON CONFLICT DO NOTHING;`,
	}

	for _, q := range alterQueries {
		if err := DB.Exec(q).Error; err != nil {
			log.Printf("⚠️ DB Migration Exec: %s -> %v", q, err)
		}
	}
}
