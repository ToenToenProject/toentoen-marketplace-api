package meilisearch

import (
	"log"
	"strings"
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

	// Category fields
	CategoryID    int64    `json:"category_id,omitempty"`
	CategoryIDs   []int64  `json:"category_ids"`
	CategoryNames []string `json:"category_names"`
	Categories    []string `json:"categories,omitempty"` // alias untuk kompatibilitas

	// Tag fields
	TagIDs     []int64  `json:"tag_ids"`
	TagNames   []string `json:"tag_names"`
	Tags       []string `json:"tags,omitempty"` // alias untuk kompatibilitas
	CustomTags []string `json:"custom_tags,omitempty"`
	Hashtags   []string `json:"hashtags,omitempty"`
}

// InitMeilisearchIndices sets filterable, sortable, and searchable attributes on Meilisearch
func InitMeilisearchIndices() {
	if Client == nil {
		return
	}
	index := Client.Index("businesses")

	filterable := &[]interface{}{
		"_geo",
		"rating",
		"is_open",
		"category_id",
		"category_ids",
		"category_names",
		"tag_ids",
		"tag_names",
	}
	sortable := &[]string{"_geo", "rating"}
	searchable := &[]string{
		"name",
		"products",
		"category_names",
		"tag_names",
		"custom_tags",
		"hashtags",
	}

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
	if db != nil {
		db.Where("business_id = ?", b.BusinessID).Select("product_name").Find(&products)
	}

	productNames := make([]string, 0, len(products))
	for _, p := range products {
		if p.ProductName != "" {
			productNames = append(productNames, p.ProductName)
		}
	}

	// Fetch categories if not preloaded
	categories := b.Categories
	if len(categories) == 0 && db != nil {
		db.Table("business_categories").
			Joins("JOIN business_has_categories ON business_has_categories.category_id = business_categories.category_id").
			Where("business_has_categories.business_id = ?", b.BusinessID).
			Find(&categories)
	}

	categoryIDs := make([]int64, 0, len(categories))
	categoryNames := make([]string, 0, len(categories))
	for _, c := range categories {
		categoryIDs = append(categoryIDs, c.CategoryID)
		if c.Name != "" {
			categoryNames = append(categoryNames, c.Name)
		}
	}
	if b.CustomCategory != nil && strings.TrimSpace(*b.CustomCategory) != "" {
		categoryNames = append(categoryNames, strings.TrimSpace(*b.CustomCategory))
	}
	var primaryCatID int64
	if len(categoryIDs) > 0 {
		primaryCatID = categoryIDs[0]
	}

	// Fetch tags if not preloaded
	tags := b.Tags
	if len(tags) == 0 && db != nil {
		db.Table("business_tags").
			Joins("JOIN business_has_tags ON business_has_tags.tag_id = business_tags.tag_id").
			Where("business_has_tags.business_id = ?", b.BusinessID).
			Find(&tags)
	}

	tagIDs := make([]int64, 0, len(tags))
	tagNames := make([]string, 0, len(tags))
	for _, t := range tags {
		tagIDs = append(tagIDs, t.TagID)
		if t.Name != "" {
			tagNames = append(tagNames, t.Name)
		}
	}
	for _, ct := range b.CustomTags {
		if trimmed := strings.TrimSpace(ct); trimmed != "" {
			tagNames = append(tagNames, trimmed)
		}
	}
	for _, ht := range b.Hashtags {
		if trimmed := strings.TrimSpace(ht); trimmed != "" {
			tagNames = append(tagNames, trimmed)
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

	previewImages := b.BusinessImages
	if previewImages == nil {
		previewImages = []string{}
	}

	doc := MeiliBusinessDocument{
		ID:            b.BusinessID,
		Name:          name,
		Products:      productNames,
		Rating:        b.AverageRating,
		ReviewCount:   b.ReviewCount,
		PreviewImages: previewImages,
		IsOpen:        isOpen,
		CloseTime:     closeTime,
		CategoryID:    primaryCatID,
		CategoryIDs:   categoryIDs,
		CategoryNames: categoryNames,
		Categories:    categoryNames,
		TagIDs:        tagIDs,
		TagNames:      tagNames,
		Tags:          tagNames,
		CustomTags:    b.CustomTags,
		Hashtags:      b.Hashtags,
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
	err := db.Preload("Hours").Preload("Categories").Preload("Tags").Where("business_id = ?", businessID).First(&b).Error
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

	log.Printf("🔍 Berhasil sync business %s (%s) ke Meilisearch [cats: %v, tags: %v]", businessID, doc.Name, doc.CategoryNames, doc.TagNames)
	return nil
}

// SyncAllBusinesses synchronizes all active businesses from PostgreSQL to Meilisearch
func SyncAllBusinesses(db *gorm.DB) (int, error) {
	if Client == nil {
		return 0, nil
	}

	var businesses []models.Business
	err := db.Preload("Hours").Preload("Categories").Preload("Tags").Where("is_active = ? AND status != ?", true, "deleted").Find(&businesses).Error
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
