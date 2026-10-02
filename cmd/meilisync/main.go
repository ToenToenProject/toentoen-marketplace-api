package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/customer/explore"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	myMeili "toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/utils"
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

	// 1. Configure Meilisearch Index Settings
	index := myMeili.Client.Index("businesses")
	
	filterable := &[]interface{}{"_geo", "rating"}
	sortable := &[]string{"_geo", "rating"}

	_, err := index.UpdateFilterableAttributes(filterable)
	if err != nil {
		log.Fatalf("Gagal set filterable: %v", err)
	}

	_, err = index.UpdateSortableAttributes(sortable)
	if err != nil {
		log.Fatalf("Gagal set sortable: %v", err)
	}
	log.Println("✅ Pengaturan Index Meilisearch berhasil diupdate")

	// 2. Ambil data dari PostgreSQL
	var businesses []models.Business
	err = db.Preload("Hours").Preload("Tags").Find(&businesses).Error
	if err != nil {
		log.Fatalf("Gagal mengambil data dari DB: %v", err)
	}

	var docs []explore.MeiliBusinessDocument

	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)
	currentDayInt := int(now.Weekday())

	for _, b := range businesses {
		// Ambil produk dari business ini
		var products []models.Product
		db.Where("business_id = ?", b.BusinessID).Find(&products)
		
		var productNames []string
		for _, p := range products {
			productNames = append(productNames, p.ProductName)
		}

		// Hitung IsOpen & CloseTime secara akurat
		isOpen := utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours)
		closeTime := ""
		for _, h := range b.Hours {
			if h.DayOfWeek == currentDayInt && h.CloseTime != nil {
				closeTime = *h.CloseTime
				break
			}
		}


		name := ""
		if b.BusinessName != nil {
			name = *b.BusinessName
		}

		doc := explore.MeiliBusinessDocument{
			ID:            b.BusinessID,
			Name:          name,
			Products:      productNames,
			Rating:        b.AverageRating,
			ReviewCount:   b.ReviewCount,
			PreviewImages: b.BusinessImages,
			IsOpen:        isOpen,
			CloseTime:     closeTime,
		}

		if b.Latitude != nil && b.Longitude != nil {
			doc.Geo = explore.GeoLocation{
				Lat: *b.Latitude,
				Lng: *b.Longitude,
			}
		}

		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		log.Println("ℹ️ Tidak ada data bisnis berstatus 'published' untuk disinkronkan.")
		return
	}

	// 3. Masukkan ke Meilisearch
	task, err := index.AddDocuments(docs, nil)
	if err != nil {
		log.Fatalf("Gagal upload ke Meilisearch: %v", err)
	}

	log.Printf("✅ Selesai! %d bisnis telah disinkronkan ke Meilisearch. (Task UID: %d)", len(docs), task.TaskUID)
}
