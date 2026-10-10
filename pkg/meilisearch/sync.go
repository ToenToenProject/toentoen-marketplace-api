package meilisearch

import (
	"log"
	"time"

	"gorm.io/gorm"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type MeiliBusinessDocument struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Products      []string    `json:"products"`
	Rating        float64     `json:"rating"`
	ReviewCount   int64       `json:"review_count"`
	PreviewImages []string    `json:"preview_images"`
	IsOpen        bool        `json:"is_open"`
	CloseTime     string      `json:"close_time"`
	Geo           GeoLocation `json:"_geo"`
	Distance      float64     `json:"_geoDistance,omitempty"`
}

// InitMeilisearchIndices sets filterable, sortable, and searchable attributes on Meilisearch
func InitMeilisearchIndices() {
	if Client == nil {
		return
	}
	index := Client.Index("businesses")

	filterable := &[]interface{}{"_geo", "rating"}
	sortable := &[]string{"_geo", "rating"}
	searchable := &[]string{"name", "products"}

	if _, err := index.UpdateFilterableAttributes(filterable); err != nil {
		log.Printf("⚠️ Gagal set filterable attributes Meilisearch: %v", err)
	}
	if _, err := index.UpdateSortableAttributes(sortable); err != nil {
		log.Printf("⚠️ Gagal set sortable attributes Meilisearch: %v", err)
	}
	if _, err := index.UpdateSearchableAttributes(searchable); err != nil {
		log.Printf("⚠️ Gagal set searchable attributes Meilisearch: %v", err)
	}
}

// BuildBusinessDocument creates a MeiliBusinessDocument from a Business model
func BuildBusinessDocument(db *gorm.DB, b *models.Business) MeiliBusinessDocument {
	loc := utils.GetWIBLocation()
	now := time.Now().In(loc)
	currentDayInt := int(now.Weekday())

	// Fetch product names for this business
	var products []models.Product
	db.Where("business_id = ?", b.BusinessID).Select("product_name").Find(&products)

	var productNames []string
	for _, p := range products {
		if p.ProductName != "" {
			productNames = append(productNames, p.ProductName)
		}
	}

	// Calculate IsOpen and CloseTime
	isOpen := utils.CalculateStoreStatus(b.IsManuallyClosed, b.Hours, b.ConfirmedOpenDate, b.ConfirmedOvertimeDate)
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

	doc := MeiliBusinessDocument{
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
		doc.Geo = GeoLocation{
			Lat: *b.Latitude,
			Lng: *b.Longitude,
		}
	}

	return doc
}

// SyncBusiness synchronizes a single business by ID to Meilisearch
func SyncBusiness(db *gorm.DB, businessID string) error {
	if Client == nil || businessID == "" {
		return nil
	}

	var b models.Business
	err := db.Preload("Hours").Where("business_id = ?", businessID).First(&b).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// If not found in DB, ensure it's deleted from Meilisearch
			return DeleteBusinessDoc(businessID)
		}
		log.Printf("⚠️ Gagal query business %s untuk Meilisearch: %v", businessID, err)
		return err
	}

	// If business is inactive or soft-deleted, remove from index
	if !b.IsActive || b.Status == "deleted" {
		return DeleteBusinessDoc(businessID)
	}

	doc := BuildBusinessDocument(db, &b)
	index := Client.Index("businesses")
	_, err = index.AddDocuments([]MeiliBusinessDocument{doc}, nil)
	if err != nil {
		log.Printf("⚠️ Gagal sync business %s (%s) ke Meilisearch: %v", businessID, doc.Name, err)
		return err
	}

	log.Printf("🔍 Berhasil sync business %s (%s) ke Meilisearch", businessID, doc.Name)
	return nil
}

// SyncAllBusinesses synchronizes all active businesses from PostgreSQL to Meilisearch
func SyncAllBusinesses(db *gorm.DB) (int, error) {
	if Client == nil {
		return 0, nil
	}

	var businesses []models.Business
	err := db.Preload("Hours").Where("is_active = ? AND status != ?", true, "deleted").Find(&businesses).Error
	if err != nil {
		log.Printf("⚠️ Gagal query semua businesses untuk Meilisearch: %v", err)
		return 0, err
	}

	if len(businesses) == 0 {
		log.Println("ℹ️ Tidak ada data bisnis untuk disinkronkan ke Meilisearch.")
		return 0, nil
	}

	docs := make([]MeiliBusinessDocument, 0, len(businesses))
	for i := range businesses {
		doc := BuildBusinessDocument(db, &businesses[i])
		docs = append(docs, doc)
	}

	index := Client.Index("businesses")
	task, err := index.AddDocuments(docs, nil)
	if err != nil {
		log.Printf("⚠️ Gagal upload batch businesses ke Meilisearch: %v", err)
		return 0, err
	}

	log.Printf("✅ Selesai sync! %d bisnis telah disinkronkan ke Meilisearch (Task UID: %d)", len(docs), task.TaskUID)
	return len(docs), nil
}

// DeleteBusinessDoc removes a business document from Meilisearch
func DeleteBusinessDoc(businessID string) error {
	if Client == nil || businessID == "" {
		return nil
	}
	index := Client.Index("businesses")
	_, err := index.DeleteDocument(businessID, nil)
	if err != nil {
		log.Printf("⚠️ Gagal hapus business %s dari Meilisearch: %v", businessID, err)
		return err
	}
	return nil
}
