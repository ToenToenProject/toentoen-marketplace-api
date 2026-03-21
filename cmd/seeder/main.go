package main

import (
	"log"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/models"
	"toentoen-backend/pkg/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: Tidak dapat memuat file .env, menggunakan environment system")
	}

	config.InitConfig()
	database.ConnectDB()
	db := database.DB

	log.Println("Memulai proses seeding Kategori dan Tag...")

	log.Println("Membersihkan tabel lama...")
	db.Exec("TRUNCATE TABLE tags, tag_groups, categories RESTART IDENTITY CASCADE;")

	categories := []models.Category{
		{
			Name: "Food & drink",
			Subcategories: []models.Category{
				{Name: "Restaurants"}, {Name: "Cafes"}, {Name: "Street food"}, {Name: "Bakery & desserts"},
				{Name: "Beverage shops"}, {Name: "Catering"}, {Name: "Indonesian food"}, {Name: "Western food"},
				{Name: "Asian food"}, {Name: "Fast food"}, {Name: "Vegetarian / Vegan"}, {Name: "Halal food"},
				{Name: "Coffee"}, {Name: "Tea"}, {Name: "Dessert"}, {Name: "Late-night food"},
			},
		},
		{
			Name: "Retail & shopping",
			Subcategories: []models.Category{
				{Name: "Grocery & minimarket"}, {Name: "Fashion & apparel"}, {Name: "Shoes & bags"},
				{Name: "Electronics"}, {Name: "Home & living"}, {Name: "Gifts & souvenirs"},
				{Name: "Books & stationery"}, {Name: "Men"}, {Name: "Women"}, {Name: "Kids"},
				{Name: "Local brand"}, {Name: "Import"}, {Name: "Secondhand"}, {Name: "Custom-made"},
			},
		},
		{
			Name: "Beauty & Wellness",
			Subcategories: []models.Category{
				{Name: "Hair salon"}, {Name: "Barbershop"}, {Name: "Nail & lash"}, {Name: "Skincare & facial"},
				{Name: "Spa & massage"}, {Name: "Wellness clinic"}, {Name: "Haircut"}, {Name: "Coloring"},
				{Name: "Treatment"}, {Name: "Massage"}, {Name: "Facial"}, {Name: "Bridal"},
				{Name: "Men-only"}, {Name: "Women-only"},
			},
		},
		{
			Name: "Fitness & Sports",
			Subcategories: []models.Category{
				{Name: "Gym"}, {Name: "Yoga & pilates"}, {Name: "Sports center"}, {Name: "Martial arts"},
				{Name: "Swimming pool"}, {Name: "Personal trainer"},
			},
		},
		{
			Name: "Services",
			Subcategories: []models.Category{
				{Name: "Repair & maintenance"}, {Name: "Cleaning service"}, {Name: "Laundry"},
				{Name: "Tailor"}, {Name: "Printing & photocopy"}, {Name: "Event services"},
			},
		},
		{
			Name: "Health & Medical",
			Subcategories: []models.Category{
				{Name: "Clinic"}, {Name: "Dental"}, {Name: "Pharmacy"}, {Name: "Therapy"}, {Name: "Traditional medicine"},
			},
		},
		{
			Name: "Education & Courses",
			Subcategories: []models.Category{
				{Name: "Tutoring"}, {Name: "Language courses"}, {Name: "Music lessons"}, {Name: "Skill training"}, {Name: "Kids education"},
			},
		},
		{
			Name: "Pet & Animal",
			Subcategories: []models.Category{
				{Name: "Pet shop"}, {Name: "Grooming"}, {Name: "Veterinary"}, {Name: "Pet hotel"}, {Name: "Pet training"},
			},
		},
		{
			Name: "Home & Property",
			Subcategories: []models.Category{
				{Name: "Furniture"}, {Name: "Interior"}, {Name: "Contractor"}, {Name: "Real estate agent"}, {Name: "Home services"},
			},
		},
		{
			Name: "Automotive",
			Subcategories: []models.Category{
				{Name: "Workshop"}, {Name: "Car wash"}, {Name: "Motorcycle service"}, {Name: "Spare parts"}, {Name: "Accessories"},
			},
		},
		{
			Name: "Entertainment & Lifestyle",
			Subcategories: []models.Category{
				{Name: "Cinema"}, {Name: "Karaoke"}, {Name: "Game center"}, {Name: "Hobby shop"}, {Name: "Community space"},
			},
		},
		{
			Name: "Travel & Accommodation",
			Subcategories: []models.Category{
				{Name: "Hotel"}, {Name: "Guesthouse"}, {Name: "Travel agent"}, {Name: "Tour services"}, {Name: "Vehicle rental"},
			},
		},
		{
			Name: "Professional & Business",
			Subcategories: []models.Category{
				{Name: "Coworking space"}, {Name: "Office services"}, {Name: "Consultant"}, {Name: "Legal & accounting"}, {Name: "Logistics"},
			},
		},
	}

	tagGroups := []models.TagGroup{
		{
			Name: "System / Platform tags",
			Tags: []models.Tag{
				{Name: "Verified merchant"}, {Name: "New business"}, {Name: "Trending nearby"}, {Name: "Highly rated"}, {Name: "Active today"},
			},
		},
		{
			Name: "Experience & Environment",
			Tags: []models.Tag{
				{Name: "Pet friendly"}, {Name: "Smoking area"}, {Name: "Non-smoking"}, {Name: "Wi-Fi available"},
				{Name: "Outdoor seating"}, {Name: "Indoor seating"}, {Name: "Air-conditioned"}, {Name: "Quiet place"},
				{Name: "Live music"}, {Name: "Aesthetic / Instagrammable"},
			},
		},
		{
			Name: "Audience & Usage",
			Tags: []models.Tag{
				{Name: "Family friendly"}, {Name: "Kids friendly"}, {Name: "Suitable for groups"}, {Name: "Solo-friendly"},
				{Name: "Work-friendly (laptop-friendly)"}, {Name: "Meeting-friendly"}, {Name: "Student-friendly"},
			},
		},
		{
			Name: "Price & Value",
			Tags: []models.Tag{
				{Name: "Budget-friendly"}, {Name: "Mid-range"}, {Name: "Premium"}, {Name: "Student discount"},
				{Name: "Bundle deals"}, {Name: "Subscription available"},
			},
		},
		{
			Name: "Dietary & Special needs",
			Tags: []models.Tag{
				{Name: "Halal certified"}, {Name: "Vegetarian friendly"}, {Name: "Vegan friendly"},
				{Name: "Gluten-free options"}, {Name: "Sugar-free options"}, {Name: "Healthy options"},
			},
		},
		{
			Name: "Accessibility & Facilities",
			Tags: []models.Tag{
				{Name: "Wheelchair accessible"}, {Name: "Accessible toilet"}, {Name: "Parking available"},
				{Name: "Motorcycle parking"}, {Name: "Near public transport"}, {Name: "Prayer room (Musholla)"},
			},
		},
		{
			Name: "Payment & Transaction",
			Tags: []models.Tag{
				{Name: "Cash"}, {Name: "QRIS accepted"}, {Name: "Debit card"}, {Name: "Credit card"},
				{Name: "E-wallet accepted"}, {Name: "Cashless only"}, {Name: "Pay-later available"},
			},
		},
		{
			Name: "Service & Availability",
			Tags: []models.Tag{
				{Name: "Open 24 hours"}, {Name: "Reservation available"}, {Name: "Walk-in available"},
				{Name: "Fast service"}, {Name: "Pre-order available"}, {Name: "Delivery available"},
				{Name: "Pickup available"}, {Name: "On-site service"},
			},
		},
	}

	log.Println("Menyimpan Kategori...")
	if err := db.Create(&categories).Error; err != nil {
		log.Fatalf("Gagal menyimpan kategori: %v", err)
	}

	log.Println("Menyimpan Tag...")
	if err := db.Create(&tagGroups).Error; err != nil {
		log.Fatalf("Gagal menyimpan tag: %v", err)
	}

	log.Println("✅ SEEDING BERHASIL! Seluruh Kategori dan Tag telah masuk ke database.")
}