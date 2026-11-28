package main

import (
	"log"
	"strings"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/meta"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type tagSeed struct {
	Slug     string
	Name     string
	Group    string
	Children []tagSeed
}

type groupSeed struct {
	Slug    string
	Name    string
	TagType string
}

type categoryRuleSeed struct {
	Slug     string
	Name     string
	TagSlugs []string
}

func upsertCategory(db *gorm.DB, slug, name string, sortOrder int) (int64, error) {
	var cat models.Category
	err := db.Where("slug = ?", slug).First(&cat).Error
	if err != nil {
		// Try matching by case-insensitive name if slug wasn't set yet
		errName := db.Where("LOWER(name) = LOWER(?) AND parent_id IS NULL", name).First(&cat).Error
		if errName != nil {
			cat = models.Category{
				Slug:      slug,
				Name:      name,
				SortOrder: sortOrder,
				IsActive:  true,
			}
			if errCreate := db.Create(&cat).Error; errCreate != nil {
				return 0, errCreate
			}
			return cat.CategoryID, nil
		}
	}
	if err := db.Model(&models.Category{}).Where("category_id = ?", cat.CategoryID).Updates(map[string]interface{}{
		"slug":       slug,
		"name":       name,
		"sort_order": sortOrder,
		"is_active":  true,
	}).Error; err != nil {
		return 0, err
	}
	return cat.CategoryID, nil
}

func upsertTagGroup(db *gorm.DB, g groupSeed, sortOrder int) (int64, error) {
	var grp models.TagGroup
	err := db.Where("slug = ?", g.Slug).First(&grp).Error
	if err != nil {
		grp = models.TagGroup{
			Slug:      g.Slug,
			Name:      g.Name,
			TagType:   g.TagType,
			SortOrder: sortOrder,
			IsActive:  true,
		}
		if errCreate := db.Create(&grp).Error; errCreate != nil {
			return 0, errCreate
		}
		return grp.GroupID, nil
	}
	if err := db.Model(&models.TagGroup{}).Where("group_id = ?", grp.GroupID).Updates(map[string]interface{}{
		"name":       g.Name,
		"tag_type":   g.TagType,
		"sort_order": sortOrder,
		"is_active":  true,
	}).Error; err != nil {
		return 0, err
	}
	return grp.GroupID, nil
}

func upsertTag(db *gorm.DB, t tagSeed, groupID int64, tagType string, parentID *int64, sortOrder int, tagIDMap map[string]int64) error {
	var existing models.Tag
	err := db.Where("slug = ?", t.Slug).First(&existing).Error
	if err != nil {
		errName := db.Where("LOWER(name) = LOWER(?) AND (slug IS NULL OR slug = '')", t.Name).First(&existing).Error
		if errName != nil {
			existing = models.Tag{
				Slug:      t.Slug,
				Name:      t.Name,
				TagType:   tagType,
				GroupID:   &groupID,
				ParentID:  parentID,
				IsGlobal:  true,
				SortOrder: sortOrder,
				IsActive:  true,
			}
			if errCreate := db.Create(&existing).Error; errCreate != nil {
				return errCreate
			}
		} else {
			if errUpd := db.Model(&models.Tag{}).Where("tag_id = ?", existing.TagID).Updates(map[string]interface{}{
				"slug":       t.Slug,
				"name":       t.Name,
				"tag_type":   tagType,
				"group_id":   groupID,
				"parent_id":  parentID,
				"is_global":  true,
				"sort_order": sortOrder,
				"is_active":  true,
			}).Error; errUpd != nil {
				return errUpd
			}
		}
	} else {
		if errUpd := db.Model(&models.Tag{}).Where("tag_id = ?", existing.TagID).Updates(map[string]interface{}{
			"name":       t.Name,
			"tag_type":   tagType,
			"group_id":   groupID,
			"parent_id":  parentID,
			"is_global":  true,
			"sort_order": sortOrder,
			"is_active":  true,
		}).Error; errUpd != nil {
			return errUpd
		}
	}

	tagIDMap[t.Slug] = existing.TagID
	for idx, child := range t.Children {
		if err := upsertTag(db, child, groupID, tagType, &existing.TagID, idx+1, tagIDMap); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	config.InitConfig()
	database.ConnectDB()
	db := database.DB

	log.Println("Starting idempotent slug-based Category & Tag seeding (conflict-safe)...")

	groups := []groupSeed{
		{Slug: "accommodation-type", Name: "Accommodation Type", TagType: "characteristic"},
		{Slug: "payment-transaction", Name: "Payment & Transaction", TagType: "payment"},
		{Slug: "accessibility-facilities", Name: "Accessibility & Facilities", TagType: "facility"},
		{Slug: "bed-sizes", Name: "Bed Sizes", TagType: "facility"},
		{Slug: "room-features", Name: "Room Features", TagType: "facility"},
		{Slug: "service-availability", Name: "Service & Availability", TagType: "service"},
		{Slug: "experience-environment", Name: "Experience & Environment", TagType: "characteristic"},
		{Slug: "dietary-special-needs", Name: "Dietary & Special Needs", TagType: "characteristic"},
		{Slug: "audience-usage", Name: "Audience & Usage", TagType: "characteristic"},
	}

	groupIDMap := make(map[string]int64)
	groupTypeMap := make(map[string]string)
	for i, g := range groups {
		gid, err := upsertTagGroup(db, g, i+1)
		if err != nil {
			log.Fatalf("Failed upserting tag group %s: %v", g.Slug, err)
		}
		groupIDMap[g.Slug] = gid
		groupTypeMap[g.Slug] = g.TagType
	}

	tags := []tagSeed{
		// Accommodation Type
		{Slug: "guesthouse", Name: "Guesthouse", Group: "accommodation-type"},
		{Slug: "hotel", Name: "Hotel", Group: "accommodation-type"},
		{Slug: "villa", Name: "Villa", Group: "accommodation-type"},
		{Slug: "homestay", Name: "Homestay", Group: "accommodation-type"},
		{Slug: "hostel", Name: "Hostel", Group: "accommodation-type"},
		{Slug: "bnb", Name: "B&B", Group: "accommodation-type"},
		{Slug: "apartments", Name: "Apartments", Group: "accommodation-type"},

		// Payment & Transaction
		{Slug: "cash", Name: "Cash", Group: "payment-transaction"},
		{Slug: "qris-accepted", Name: "QRIS accepted", Group: "payment-transaction"},
		{Slug: "debit-card", Name: "Debit card", Group: "payment-transaction"},
		{Slug: "credit-card", Name: "Credit card", Group: "payment-transaction"},
		{
			Slug: "e-wallet-accepted", Name: "E-wallet accepted", Group: "payment-transaction",
			Children: []tagSeed{
				{Slug: "gopay", Name: "Gopay", Group: "payment-transaction"},
				{Slug: "shopeepay", Name: "ShopeePay", Group: "payment-transaction"},
				{Slug: "dana", Name: "DANA", Group: "payment-transaction"},
				{Slug: "ovo", Name: "OVO", Group: "payment-transaction"},
				{Slug: "linkaja", Name: "LinkAja", Group: "payment-transaction"},
			},
		},

		// Accessibility & Facilities
		{Slug: "wheelchair-accessible", Name: "Wheelchair accessible", Group: "accessibility-facilities"},
		{Slug: "accessible-toilet", Name: "Accessible toilet", Group: "accessibility-facilities"},
		{Slug: "parking-available", Name: "Parking available", Group: "accessibility-facilities"},
		{Slug: "motorcycle-parking", Name: "Motorcycle parking", Group: "accessibility-facilities"},
		{Slug: "bicycle-parking", Name: "Bicycle parking", Group: "accessibility-facilities"},
		{Slug: "near-public-transport", Name: "Near public transport", Group: "accessibility-facilities"},
		{Slug: "restaurant", Name: "Restaurant", Group: "accessibility-facilities"},
		{Slug: "lounge", Name: "Lounge", Group: "accessibility-facilities"},
		{Slug: "laundry", Name: "Laundry", Group: "accessibility-facilities"},
		{Slug: "conference-meeting-room", Name: "Conference & Meeting Room", Group: "accessibility-facilities"},
		{Slug: "pool", Name: "Pool", Group: "accessibility-facilities"},
		{Slug: "room-service", Name: "Room Service", Group: "accessibility-facilities"},
		{Slug: "fitness-centre", Name: "Fitness Centre", Group: "accessibility-facilities"},
		{Slug: "spa-wellness", Name: "Spa & Wellness", Group: "accessibility-facilities"},
		{Slug: "children-facilities", Name: "Children Facilities", Group: "accessibility-facilities"},
		{Slug: "24-hour-front-desk", Name: "24 Hour Front desk", Group: "accessibility-facilities"},
		{Slug: "late-check-out", Name: "Late Check-out", Group: "accessibility-facilities"},
		{Slug: "late-check-in", Name: "Late Check-in", Group: "accessibility-facilities"},
		{Slug: "elevator", Name: "Elevator", Group: "accessibility-facilities"},
		{Slug: "prayer-room-musholla", Name: "Prayer room (Musholla)", Group: "accessibility-facilities"},

		// Bed Sizes
		{Slug: "single-bed", Name: "Single Bed", Group: "bed-sizes"},
		{Slug: "double-bed", Name: "Double Bed", Group: "bed-sizes"},
		{Slug: "queen", Name: "Queen", Group: "bed-sizes"},
		{Slug: "king", Name: "King", Group: "bed-sizes"},

		// Room Features
		{Slug: "air-conditioner", Name: "Air-Conditioner", Group: "room-features"},
		{Slug: "inner-bathroom", Name: "Inner Bathroom", Group: "room-features"},
		{Slug: "smart-tv", Name: "Smart TV", Group: "room-features"},
		{Slug: "smoking", Name: "Smoking", Group: "room-features"},
		{Slug: "non-smoking", Name: "Non-Smoking", Group: "room-features"},
		{Slug: "refrigerator", Name: "Refrigerator", Group: "room-features"},
		{Slug: "baby-cribs", Name: "Baby Cribs", Group: "room-features"},
		{Slug: "connecting-room", Name: "Connecting Room", Group: "room-features"},
		{Slug: "hairdryer", Name: "Hairdryer", Group: "room-features"},

		// Service & Availability
		{Slug: "reservation-available", Name: "Reservation available", Group: "service-availability"},
		{Slug: "walk-in-available", Name: "Walk-in available", Group: "service-availability"},
		{Slug: "pre-order-available", Name: "Pre-order available", Group: "service-availability"},
		{Slug: "home-service", Name: "Home service", Group: "service-availability"},
		{Slug: "delivery-available", Name: "Delivery available", Group: "service-availability"},
		{Slug: "pickup-available", Name: "Pickup available", Group: "service-availability"},
		{Slug: "on-site-service", Name: "On-site service", Group: "service-availability"},
		{Slug: "delivery-only", Name: "Delivery Only", Group: "service-availability"},
		{Slug: "dine-in-only", Name: "Dine-in Only", Group: "service-availability"},
		{Slug: "pick-up-only", Name: "Pick-up Only", Group: "service-availability"},

		// Experience & Environment
		{
			Slug: "pet-friendly", Name: "Pet friendly", Group: "experience-environment",
			Children: []tagSeed{
				{Slug: "small-breed-only", Name: "Small breed only", Group: "experience-environment"},
				{Slug: "all-breed", Name: "All breed", Group: "experience-environment"},
				{Slug: "cat-only", Name: "Cat only", Group: "experience-environment"},
				{Slug: "dogs-only", Name: "Dogs only", Group: "experience-environment"},
			},
		},
		{Slug: "indoor-smoking-area", Name: "Indoor smoking area", Group: "experience-environment"},
		{Slug: "outdoor-smoking-area", Name: "Outdoor smoking area", Group: "experience-environment"},
		{Slug: "env-non-smoking", Name: "Non-smoking", Group: "experience-environment"},
		{Slug: "wifi-available", Name: "Wi-Fi available", Group: "experience-environment"},
		{Slug: "outdoor-seating", Name: "Outdoor seating", Group: "experience-environment"},
		{Slug: "indoor-seating", Name: "Indoor seating", Group: "experience-environment"},
		{Slug: "air-conditioned", Name: "Air-conditioned", Group: "experience-environment"},
		{Slug: "quiet-place", Name: "Quiet place", Group: "experience-environment"},
		{Slug: "live-music", Name: "Live music", Group: "experience-environment"},
		{Slug: "aesthetic-instagrammable", Name: "Aesthetic / Instagrammable", Group: "experience-environment"},

		// Dietary & Special Needs
		{Slug: "halal-certified", Name: "Halal certified", Group: "dietary-special-needs"},
		{Slug: "non-halal", Name: "Non-Halal", Group: "dietary-special-needs"},
		{Slug: "vegetarian-friendly", Name: "Vegetarian friendly", Group: "dietary-special-needs"},
		{Slug: "vegan-friendly", Name: "Vegan friendly", Group: "dietary-special-needs"},
		{Slug: "gluten-free-options", Name: "Gluten-free options", Group: "dietary-special-needs"},
		{Slug: "sugar-free-options", Name: "Sugar-free options", Group: "dietary-special-needs"},
		{Slug: "healthy-options", Name: "Healthy options", Group: "dietary-special-needs"},

		// Audience & Usage
		{Slug: "family-friendly", Name: "Family friendly", Group: "audience-usage"},
		{Slug: "kids-friendly", Name: "Kids friendly", Group: "audience-usage"},
		{Slug: "suitable-for-groups", Name: "Suitable for groups", Group: "audience-usage"},
		{Slug: "solo-friendly", Name: "Solo-friendly", Group: "audience-usage"},
		{Slug: "work-friendly", Name: "Work-friendly", Group: "audience-usage"},
		{Slug: "meeting-friendly", Name: "Meeting-friendly", Group: "audience-usage"},
	}

	tagIDMap := make(map[string]int64)
	for i, t := range tags {
		if err := upsertTag(db, t, groupIDMap[t.Group], groupTypeMap[t.Group], nil, i+1, tagIDMap); err != nil {
			log.Fatalf("Failed upserting tag %s: %v", t.Slug, err)
		}
	}

	paymentFull := []string{"cash", "qris-accepted", "debit-card", "credit-card", "e-wallet-accepted", "gopay", "shopeepay", "dana", "ovo", "linkaja"}
	paymentBasic := []string{"cash", "debit-card", "credit-card"}
	accessStandard := []string{"wheelchair-accessible", "accessible-toilet", "parking-available", "motorcycle-parking", "bicycle-parking", "near-public-transport"}
	accessWithMusholla := append(append([]string{}, accessStandard...), "prayer-room-musholla")
	petFriendlyFull := []string{"pet-friendly", "small-breed-only", "all-breed", "cat-only", "dogs-only"}
	expFull := append(append([]string{}, petFriendlyFull...), "indoor-smoking-area", "outdoor-smoking-area", "env-non-smoking", "wifi-available", "outdoor-seating", "indoor-seating", "air-conditioned", "quiet-place", "live-music", "aesthetic-instagrammable")

	join := func(slices ...[]string) []string {
		var res []string
		for _, s := range slices {
			res = append(res, s...)
		}
		return res
	}

	categories := []categoryRuleSeed{
		{
			Slug: "akomodasi-perjalanan", Name: "Akomodasi Perjalanan",
			TagSlugs: join(
				[]string{"guesthouse", "hotel", "villa", "homestay", "hostel", "bnb", "apartments"},
				paymentFull,
				[]string{"wheelchair-accessible", "parking-available", "motorcycle-parking", "bicycle-parking", "near-public-transport", "restaurant", "lounge", "laundry", "conference-meeting-room", "pool", "room-service", "fitness-centre", "spa-wellness", "children-facilities", "24-hour-front-desk", "late-check-out", "late-check-in", "elevator", "prayer-room-musholla"},
				[]string{"single-bed", "double-bed", "queen", "king"},
				[]string{"air-conditioner", "inner-bathroom", "smart-tv", "smoking", "non-smoking", "refrigerator", "baby-cribs", "connecting-room", "hairdryer"},
			),
		},
		{
			Slug: "alat-tulis-buku", Name: "Alat Tulis & Buku",
			TagSlugs: join(paymentFull, accessWithMusholla, []string{"pre-order-available", "delivery-available", "pickup-available", "delivery-only", "pick-up-only"}),
		},
		{
			Slug: "bahan-makanan-kebutuhan", Name: "Bahan Makanan & Kebutuhan",
			TagSlugs: join(paymentFull, accessWithMusholla, []string{"pre-order-available", "delivery-available", "pickup-available", "delivery-only", "pick-up-only"}),
		},
		{
			Slug: "elektronik", Name: "Elektronik",
			TagSlugs: join(paymentFull, accessStandard, []string{"pre-order-available", "delivery-available", "pickup-available"}),
		},
		{
			Slug: "hadiah-souvenir", Name: "Hadiah & Souvenir",
			TagSlugs: join(paymentFull, accessStandard, []string{"pre-order-available", "delivery-available", "pickup-available", "delivery-only", "pick-up-only"}),
		},
		{
			Slug: "hewan-peliharaan", Name: "Hewan Peliharaan",
			TagSlugs: join(paymentFull, accessStandard, []string{"delivery-available", "pickup-available", "delivery-only", "pick-up-only"}),
		},
		{
			Slug: "hiburan-hobi", Name: "Hiburan & Hobi",
			TagSlugs: join(paymentFull, accessWithMusholla, []string{"reservation-available", "home-service", "on-site-service"}, expFull),
		},
		{
			Slug: "jasa", Name: "Jasa",
			TagSlugs: join(paymentFull, []string{"reservation-available", "delivery-available", "pickup-available", "delivery-only", "home-service", "on-site-service"}),
		},
		{
			Slug: "kecantikan", Name: "Kecantikan",
			TagSlugs: join(paymentFull, accessWithMusholla, []string{"reservation-available", "home-service", "on-site-service"}),
		},
		{
			Slug: "kesehatan-medis", Name: "Kesehatan & Medis",
			TagSlugs: join(paymentFull, accessWithMusholla, []string{"reservation-available", "home-service", "on-site-service"}),
		},
		{
			Slug: "layanan-kebersihan", Name: "Layanan Kebersihan",
			TagSlugs: paymentFull,
		},
		{
			Slug: "layanan-profesional", Name: "Layanan Profesional",
			TagSlugs: join(paymentFull, []string{"reservation-available", "home-service", "on-site-service"}),
		},
		{
			Slug: "makanan-minuman", Name: "Makanan & Minuman",
			TagSlugs: join(
				[]string{"halal-certified", "non-halal", "vegetarian-friendly", "vegan-friendly", "gluten-free-options", "sugar-free-options", "healthy-options"},
				paymentFull,
				accessWithMusholla,
				[]string{"reservation-available", "walk-in-available", "pre-order-available", "home-service", "delivery-available", "pickup-available", "on-site-service", "delivery-only", "dine-in-only", "pick-up-only"},
				expFull,
				[]string{"family-friendly", "kids-friendly", "suitable-for-groups", "solo-friendly", "work-friendly", "meeting-friendly"},
			),
		},
		{
			Slug: "mode-pakaian", Name: "Mode & Pakaian",
			TagSlugs: join(paymentFull, accessWithMusholla),
		},
		{
			Slug: "olahraga-kebugaran", Name: "Olahraga & Kebugaran",
			TagSlugs: join(paymentFull, accessWithMusholla, []string{"reservation-available", "home-service", "on-site-service"}),
		},
		{
			Slug: "otomotif", Name: "Otomotif",
			TagSlugs: join(paymentFull, []string{"indoor-smoking-area", "outdoor-smoking-area", "env-non-smoking", "wifi-available", "air-conditioned"}, accessWithMusholla, []string{"reservation-available", "home-service", "on-site-service"}),
		},
		{
			Slug: "pendidikan-keterampilan", Name: "Pendidikan & Keterampilan",
			TagSlugs: join(paymentFull, []string{"air-conditioned", "wifi-available", "reservation-available", "home-service", "on-site-service"}),
		},
		{
			Slug: "perlengkapan-rumah", Name: "Perlengkapan Rumah",
			TagSlugs: join(paymentFull, accessStandard),
		},
		{
			Slug: "properti", Name: "Properti",
			TagSlugs: join(paymentBasic, accessWithMusholla),
		},
		{
			Slug: "tempat-tinggal", Name: "Tempat Tinggal",
			TagSlugs: join(paymentBasic, accessWithMusholla),
		},
	}

	activeCatSlugs := make([]string, 0, len(categories))
	activeTagSlugs := make([]string, 0, len(tagIDMap))
	for s := range tagIDMap {
		activeTagSlugs = append(activeTagSlugs, s)
	}

	// Deactivate old rules first, then upsert active rules
	_ = db.Model(&models.CategoryTagRule{}).Where("1 = 1").Update("is_active", false).Error

	for i, c := range categories {
		activeCatSlugs = append(activeCatSlugs, c.Slug)
		catID, err := upsertCategory(db, c.Slug, c.Name, i+1)
		if err != nil {
			log.Fatalf("Failed upserting category %s: %v", c.Name, err)
		}
		for idx, tSlug := range c.TagSlugs {
			tagID, ok := tagIDMap[tSlug]
			if !ok {
				log.Fatalf("Unknown tag slug %s in category %s", tSlug, c.Slug)
			}
			rule := models.CategoryTagRule{
				CategoryID: catID,
				TagID:      tagID,
				SortOrder:  idx + 1,
				IsActive:   true,
			}
			if err := db.Where("category_id = ? AND tag_id = ?", catID, tagID).
				Assign(map[string]interface{}{"sort_order": idx + 1, "is_active": true}).
				FirstOrCreate(&rule).Error; err != nil {
				log.Fatalf("Failed upserting rule (%s, %s): %v", c.Slug, tSlug, err)
			}
		}
	}

	// Soft-deactivate legacy categories & tags not in the new canonical list (never delete/truncate!)
	if len(activeCatSlugs) > 0 {
		_ = db.Model(&models.Category{}).Where("slug IS NULL OR slug NOT IN ?", activeCatSlugs).Update("is_active", false).Error
	}
	if len(activeTagSlugs) > 0 {
		_ = db.Model(&models.Tag{}).Where("slug IS NULL OR slug NOT IN ?", activeTagSlugs).Update("is_active", false).Error
	}

	if err := meta.SeedIndonesiaRegions(db); err != nil {
		log.Printf("⚠️ Failed seeding Indonesia regions: %v", err)
	}

	_ = strings.TrimSpace("")
	log.Printf("Seeding completed: %d categories, %d tag groups, %d canonical tags.", len(categories), len(groups), len(tagIDMap))
}
