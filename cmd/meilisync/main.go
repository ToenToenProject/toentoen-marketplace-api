package main

import (
	"log"

	"github.com/joho/godotenv"
	"toentoen-backend/internal/config"
	"toentoen-backend/pkg/database"
	myMeili "toentoen-backend/pkg/meilisearch"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system variables")
	}

	config.InitConfig()
	database.ConnectDB()
	myMeili.ConnectMeilisearch()
	db := database.DB

	log.Println("🚀 Memulai proses sinkronisasi PostgreSQL ke Meilisearch...")

	myMeili.InitMeilisearchIndices()

	count, err := myMeili.SyncAllBusinesses(db)
	if err != nil {
		log.Fatalf("Gagal sinkronisasi Meilisearch: %v", err)
	}

	log.Printf("✅ Selesai! %d bisnis telah disinkronkan ke Meilisearch.", count)
}
