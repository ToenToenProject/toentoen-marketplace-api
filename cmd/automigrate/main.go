package main

import (
	"log"

	"github.com/joho/godotenv"
	"toentoen-backend/internal/config"
	"toentoen-backend/pkg/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system variables")
	}

	config.InitConfig()
	database.ConnectDB()

	log.Println(" Membuat tabel business_bookmarks...")
	
	err := database.DB.Exec(`CREATE TABLE IF NOT EXISTS "business_bookmarks" (
		"account_id" varchar NOT NULL,
		"business_id" varchar NOT NULL,
		"created_at" timestamptz DEFAULT (now()),
		PRIMARY KEY ("account_id", "business_id"),
		FOREIGN KEY ("account_id") REFERENCES "accounts" ("account_id") ON DELETE CASCADE,
		FOREIGN KEY ("business_id") REFERENCES "businesses" ("business_id") ON DELETE CASCADE
	);`).Error
	if err != nil {
		log.Fatalf("Gagal membuat tabel business_bookmarks: %v", err)
	}

	log.Println(" Tabel business_bookmarks berhasil ditambahkan!")

	log.Println(" Menambahkan kolom confirmed_open_date dan confirmed_overtime_date ke tabel businesses...")
	if err := database.DB.Exec(`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS confirmed_open_date DATE;`).Error; err != nil {
		log.Fatalf("Gagal menambahkan confirmed_open_date: %v", err)
	}
	if err := database.DB.Exec(`ALTER TABLE businesses ADD COLUMN IF NOT EXISTS confirmed_overtime_date DATE;`).Error; err != nil {
		log.Fatalf("Gagal menambahkan confirmed_overtime_date: %v", err)
	}
	log.Println(" Kolom konfirmasi jadwal berhasil ditambahkan ke businesses!")
}
