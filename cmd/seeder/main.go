package main

import (
	"log"
	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func saveCategory(db *gorm.DB, cat *models.Category, parentID *int64) error {
	children := cat.Children
	cat.Children = nil
	cat.ParentID = parentID
	if err := db.Create(cat).Error; err != nil {
		return err
	}
	for i := range children {
		if err := saveCategory(db, &children[i], &cat.CategoryID); err != nil {
			return err
		}
	}
	cat.Children = children
	return nil
}

func saveTag(db *gorm.DB, tag *models.Tag, parentID *int64) error {
	children := tag.Children
	tag.Children = nil
	tag.ParentID = parentID
	if err := db.Create(tag).Error; err != nil {
		return err
	}
	for i := range children {
		children[i].CategoryID = tag.CategoryID
		children[i].IsGlobal = tag.IsGlobal
		if err := saveTag(db, &children[i], &tag.TagID); err != nil {
			return err
		}
	}
	tag.Children = children
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: Tidak dapat memuat file .env, menggunakan environment system")
	}

	config.InitConfig()
	database.ConnectDB()
	db := database.DB

	log.Println("Memulai proses seeding Kategori dan Tag...")

	log.Println("Membersihkan tabel lama...")
	db.Exec("TRUNCATE TABLE business_has_tags, business_has_categories, customer_preferred_categories RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE business_tags, business_categories RESTART IDENTITY CASCADE;")

	categories := []models.Category{
		{
			Name: "Food & drink",
			Children: []models.Category{
				{Name: "Indonesian"}, {Name: "Western"}, {Name: "Italian"}, {Name: "Japanese"},
				{Name: "Korean"}, {Name: "Thai"}, {Name: "Vietnamese"}, {Name: "Chinese"},
				{Name: "Mediterranean"}, {Name: "Latin American"}, {Name: "African"}, {Name: "Middle-East"},
				{Name: "Indian"}, {Name: "Greek"}, {Name: "Fast food"}, {Name: "Street food"},
				{Name: "Pastry"}, {Name: "Bakery"}, {Name: "Desserts"}, {Name: "Coffee"},
				{Name: "Juices"}, {Name: "Boba"}, {Name: "Tea"}, {Name: "Catering"},
			},
		},
		{
			Name: "Retail & shopping",
			Children: []models.Category{
				{Name: "Grocery"},
				{
					Name: "Fashion & apparel",
					Children: []models.Category{
						{Name: "Clothing"}, {Name: "Boutique"}, {Name: "Footwear"}, {Name: "Bags & Luggage"},
						{Name: "Watches"}, {Name: "Accessories"}, {Name: "Department Store"}, {Name: "Eyewear"},
						{Name: "Tailor & Alterations"}, {Name: "Dressmaker"}, {Name: "Custom Suit"},
						{Name: "Uniform"}, {Name: "Alteration Service"}, {Name: "Wedding Dress"},
					},
				},
				{
					Name: "Electronics",
					Children: []models.Category{
						{Name: "Mobile Phones & Tablets"}, {Name: "Computer & Laptop"},
						{Name: "Accessories"}, {Name: "Home Electronics"}, {Name: "Repair Services"},
					},
				},
				{
					Name: "Home & living",
					Children: []models.Category{
						{Name: "Furniture"}, {Name: "Kitchenware"}, {Name: "Bathroom Supply"}, {Name: "Home Appliance"},
					},
				},
				{
					Name: "Gifts & souvenirs",
					Children: []models.Category{
						{Name: "Florist"}, {Name: "Hampers & Gift basket"}, {Name: "Handicraft"},
						{Name: "Collectibles"}, {Name: "Souvenirs"},
					},
				},
				{
					Name: "Books & stationery",
					Children: []models.Category{
						{Name: "Bookstore"}, {Name: "School supply"}, {Name: "Office supply"}, {Name: "Art & Craft"},
					},
				},
			},
		},
		{
			Name: "Beauty & Wellness",
			Children: []models.Category{
				{Name: "Hair salon"}, {Name: "Barbershop"}, {Name: "Nail & lash"},
				{Name: "Skincare & facial"}, {Name: "Spa & massage"}, {Name: "Wellness clinic"},
			},
		},
		{
			Name: "Fitness & Sports",
			Children: []models.Category{
				{Name: "Gym"}, {Name: "Yoga"}, {Name: "Pilates"}, {Name: "Sports center"},
				{
					Name: "Martial arts",
					Children: []models.Category{
						{Name: "Taekwondo"}, {Name: "Karate"}, {Name: "Pencak Silat"},
						{Name: "Brazilian Jiujitsu"}, {Name: "Boxing"}, {Name: "Muay Thai"},
						{Name: "Judo"}, {Name: "Wushu / Kung Fu"}, {Name: "Aikido"}, {Name: "MMA"},
					},
				},
				{Name: "Swimming pool"}, {Name: "Personal trainer"},
			},
		},
		{
			Name: "Services",
			Children: []models.Category{
				{
					Name: "Repair & maintenance",
					Children: []models.Category{
						{Name: "General repair"}, {Name: "Air Conditioner service"},
						{Name: "Electronics repair"}, {Name: "Computer & Laptop repair"},
						{Name: "Locksmith"}, {Name: "Plumbing"},
					},
				},
				{
					Name: "Cleaning service",
					Children: []models.Category{
						{Name: "Home cleaning"}, {Name: "Office cleaning"}, {Name: "Pool cleaning"},
					},
				},
				{
					Name: "Domestic Helper Agency",
					Children: []models.Category{
						{Name: "Babysitter"}, {Name: "Elderly Care"}, {Name: "Live-in Helper"}, {Name: "Part-time Helper"},
					},
				},
				{
					Name: "Laundry",
					Children: []models.Category{
						{Name: "Self-service laundry"}, {Name: "Commercial laundry"},
						{Name: "Shoe laundry"}, {Name: "Bag laundry"},
					},
				},
				{Name: "Printing & photocopy"},
				{
					Name: "Event services",
					Children: []models.Category{
						{Name: "Event organizer"}, {Name: "Wedding organizer"},
						{Name: "Event equipment rental"}, {Name: "Decoration service"}, {Name: "Catering service"},
					},
				},
				{
					Name: "Security",
					Children: []models.Category{
						{Name: "Security guard service"}, {Name: "VIP protection"},
						{Name: "Security system integrator"}, {Name: "CCTV installation & monitoring"}, {Name: "Cyber security"},
					},
				},
				{Name: "Towing Services"},
				{Name: "Insurance Service"},
			},
		},
		{
			Name: "Health & Medical",
			Children: []models.Category{
				{Name: "Clinic"}, {Name: "General practices"}, {Name: "Dental"},
				{Name: "Pharmacy"}, {Name: "Therapy"}, {Name: "Traditional medicine"}, {Name: "Hospital"},
			},
		},
		{
			Name: "Education & Courses",
			Children: []models.Category{
				{Name: "Academic tutoring"}, {Name: "Test preparation"}, {Name: "Private tutoring"},
				{
					Name: "Language courses",
					Children: []models.Category{
						{Name: "English"}, {Name: "Mandarin"}, {Name: "Korean"},
						{Name: "Japanese"}, {Name: "German"}, {Name: "French"}, {Name: "Other"},
					},
				},
				{
					Name: "Music lessons",
					Children: []models.Category{
						{Name: "Piano"}, {Name: "Guitar"}, {Name: "Violin"},
						{Name: "Vocal"}, {Name: "Drums"}, {Name: "Other"},
					},
				},
				{
					Name: "Skill training",
					Children: []models.Category{
						{Name: "Computer / IT"}, {Name: "Beauty & makeup"}, {Name: "Culinary & Baking"},
						{Name: "Digital Marketing"}, {Name: "Public Speaking"}, {Name: "Graphic design & multimedia"},
						{Name: "Coding & Software"}, {Name: "Photography & videography"}, {Name: "Fashion design"},
						{Name: "Barista academy"}, {Name: "UI/UX design"}, {Name: "AI & Data"},
					},
				},
				{Name: "Driving school"}, {Name: "Kid's education"},
			},
		},
		{
			Name: "Pet & Animal",
			Children: []models.Category{
				{Name: "Pet shop"}, {Name: "Grooming"}, {Name: "Veterinary"},
				{Name: "Pet hotel"}, {Name: "Pet trainer"},
			},
		},
		{
			Name: "Home & Property",
			Children: []models.Category{
				{Name: "Furniture"}, {Name: "Architect"}, {Name: "Interior Design"},
				{Name: "Contractor"}, {Name: "Real estate"},
				{
					Name: "Space & Property Rental",
					Children: []models.Category{
						{Name: "House"}, {Name: "Apartment"}, {Name: "Office & Co-Working space"},
						{Name: "Warehouse"}, {Name: "Industrial Space"}, {Name: "Event Venue"},
					},
				},
			},
		},
		{
			Name: "Automotive",
			Children: []models.Category{
				{Name: "Car Repair Workshop"}, {Name: "Car wash"}, {Name: "Motorcycle Workshop"},
				{Name: "Spare parts"}, {Name: "Body repair & paint"}, {Name: "Detailing"},
				{Name: "Accessories"}, {Name: "Dealership"}, {Name: "Mechanic"}, {Name: "Tire shop"},
			},
		},
		{
			Name: "Entertainment",
			Children: []models.Category{
				{Name: "Cinema"}, {Name: "Karaoke"}, {Name: "Game center"},
				{Name: "Hobby shop"}, {Name: "Community space"},
			},
		},
		{
			Name: "Travel & Accommodation",
			Children: []models.Category{
				{
					Name: "Hotel",
					Children: []models.Category{
						{Name: "1 - 5 stars (user can choose which star they want)"},
					},
				},
				{Name: "Guesthouse / Hostel"}, {Name: "Resort"}, {Name: "Villa"},
				{Name: "Travel agent"}, {Name: "Tour services"},
				{
					Name: "Vehicle rental",
					Children: []models.Category{
						{Name: "Car rental"}, {Name: "Motorcycle rental"},
					},
				},
			},
		},
		{
			Name: "Professional & Business",
			Children: []models.Category{
				{Name: "Coworking space"}, {Name: "Office services"},
				{
					Name: "Consultant",
					Children: []models.Category{
						{Name: "Tax"}, {Name: "Business"}, {Name: "Legal"},
						{Name: "Financial"}, {Name: "HR & Recruitment"}, {Name: "IT"},
					},
				},
				{Name: "Digital Marketing"}, {Name: "ISO & Certification"},
				{
					Name: "Accounting",
					Children: []models.Category{
						{Name: "Bookkeeping"}, {Name: "Tax accounting & filing"},
						{Name: "Payroll service"}, {Name: "Accounting firm"}, {Name: "Audit Service"},
					},
				},
				{
					Name: "Logistics",
					Children: []models.Category{
						{Name: "Courier & Parcel Delivery"}, {Name: "Freight forwarder"},
						{Name: "Trucking & land transportation"}, {Name: "Warehouse & storage"},
						{Name: "International shipping"}, {Name: "Custom clearance"},
						{Name: "Third party logistic"}, {Name: "Air freight"}, {Name: "Sea freight"},
					},
				},
			},
		},
	}

	log.Println("Menyimpan Kategori...")
	for i := range categories {
		if err := saveCategory(db, &categories[i], nil); err != nil {
			log.Fatalf("Gagal menyimpan kategori %s: %v", categories[i].Name, err)
		}
	}

	// Ambil ID kategori spesifik untuk relasi Tag
	var retailCat, beautyCat, insuranceCat models.Category
	db.Where("name = ? AND parent_id IS NULL", "Retail & shopping").First(&retailCat)
	db.Where("name = ? AND parent_id IS NULL", "Beauty & Wellness").First(&beautyCat)
	db.Where("name = ?", "Insurance Service").First(&insuranceCat)

	// Specific Tags (terikat kategori, is_global = false)
	specificTagGroups := []models.Tag{
		{
			Name:       "Retail & Shopping",
			CategoryID: &retailCat.CategoryID,
			IsGlobal:   false,
			Children: []models.Tag{
				{Name: "Men"}, {Name: "Women"}, {Name: "Kids"}, {Name: "Local brand"},
				{Name: "Import"}, {Name: "Secondhand"}, {Name: "Custom-made"}, {Name: "Vintage"},
			},
		},
		{
			Name:       "Beauty & Wellness",
			CategoryID: &beautyCat.CategoryID,
			IsGlobal:   false,
			Children: []models.Tag{
				{Name: "Haircut"}, {Name: "Coloring"}, {Name: "Treatment"}, {Name: "Massage"},
				{Name: "Facial"}, {Name: "Bridal"}, {Name: "Family Spa"}, {Name: "Couple Spa"},
				{Name: "Home Service"}, {Name: "Men-only"}, {Name: "Women-only"},
			},
		},
		{
			Name:       "Insurance Service",
			CategoryID: &insuranceCat.CategoryID,
			IsGlobal:   false,
			Children: []models.Tag{
				{Name: "Health insurance"}, {Name: "Life insurance"}, {Name: "General insurance"},
				{Name: "Vehicle insurance"}, {Name: "Sharia insurance"}, {Name: "Property insurance"},
				{Name: "Employee & Group insurance"},
			},
		},
	}

	// Global Tags (tidak terikat kategori, is_global = true)
	globalTagGroups := []models.Tag{
		{
			Name:     "System / Platform",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Verified merchant (NIB)"}, {Name: "New business"},
				{Name: "Trending nearby"}, {Name: "Highly rated"}, {Name: "Active today"},
			},
		},
		{
			Name:     "Experience & Environment",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Pet friendly"}, {Name: "Smoking area"}, {Name: "Non-smoking"},
				{Name: "Wi-Fi available"}, {Name: "Outdoor seating"}, {Name: "Indoor seating"},
				{Name: "Air-conditioned"}, {Name: "Quiet place"}, {Name: "Live music"},
				{Name: "Aesthetic / Instagrammable"},
			},
		},
		{
			Name:     "Audience & Usage",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Family friendly"}, {Name: "Kids friendly"}, {Name: "Suitable for groups"},
				{Name: "Solo-friendly"}, {Name: "Work-friendly (laptop-friendly)"},
				{Name: "Meeting-friendly"}, {Name: "Student-friendly"},
			},
		},
		{
			Name:     "Price & Value",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Budget-friendly"}, {Name: "Mid-range"}, {Name: "Premium"},
				{Name: "Student discount"}, {Name: "Bundle deals"}, {Name: "Subscription available"},
			},
		},
		{
			Name:     "Dietary & Special needs",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Halal certified"}, {Name: "Vegetarian friendly"}, {Name: "Vegan friendly"},
				{Name: "Gluten-free options"}, {Name: "Sugar-free options"}, {Name: "Healthy options"},
			},
		},
		{
			Name:     "Accessibility & Facilities",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Wheelchair accessible"}, {Name: "Accessible toilet"},
				{Name: "Parking available"}, {Name: "Motorcycle parking"},
				{Name: "Near public transport"}, {Name: "Prayer room (Musholla)"},
			},
		},
		{
			Name:     "Payment & Transaction",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Cash"}, {Name: "QRIS accepted"}, {Name: "Debit card"},
				{Name: "Credit card"}, {Name: "E-wallet accepted"}, {Name: "Cashless only"},
				{Name: "Pay-later available"},
			},
		},
		{
			Name:     "Service & Availability",
			IsGlobal: true,
			Children: []models.Tag{
				{Name: "Open 24 hours"}, {Name: "Reservation available"}, {Name: "Walk-in available"},
				{Name: "Fast service"}, {Name: "Pre-order available"}, {Name: "Home service"},
				{Name: "Delivery available"}, {Name: "Pickup available"}, {Name: "On-site service"},
				{Name: "Delivery Only"}, {Name: "Dine-in Only"}, {Name: "Pick-up Only"},
			},
		},
	}

	log.Println("Menyimpan Specific Tags...")
	for i := range specificTagGroups {
		if err := saveTag(db, &specificTagGroups[i], nil); err != nil {
			log.Fatalf("Gagal menyimpan specific tag %s: %v", specificTagGroups[i].Name, err)
		}
	}

	log.Println("Menyimpan Global Tags...")
	for i := range globalTagGroups {
		if err := saveTag(db, &globalTagGroups[i], nil); err != nil {
			log.Fatalf("Gagal menyimpan global tag %s: %v", globalTagGroups[i].Name, err)
		}
	}

	log.Println("✅ SEEDING BERHASIL! Seluruh Kategori dan Tag telah masuk ke database dengan hierarki yang benar.")
}
