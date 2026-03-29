package models

import (
	"time"
)

type Business struct {
	BusinessID       string   `gorm:"primaryKey;column:business_id" json:"business_id"`
	AccountID        string   `gorm:"column:account_id" json:"account_id"`
	Handle           *string `gorm:"unique;column:handle" json:"handle"`
	LogoURL          *string `gorm:"column:logo_url" json:"logo_url"`
	BusinessName     string  `gorm:"not null;column:name" json:"name"`
	EmailContact     *string `gorm:"column:email_contact" json:"email_contact"`
	Status           string  `gorm:"default:'draft';column:status" json:"status"`
	SubscriptionTier string  `gorm:"default:'free';column:subscription_tier" json:"subscription_tier"`
	Description      *string `gorm:"column:description" json:"description"`

	Province      *string  `gorm:"column:province" json:"province"`
	City          *string  `gorm:"column:city" json:"city"`
	AddressDetail *string  `gorm:"column:address_detail" json:"address_detail"`
	Address       *string  `gorm:"column:address" json:"address"`
	Latitude      *float64 `gorm:"column:latitude" json:"latitude"`
	Longitude     *float64 `gorm:"column:longitude" json:"longitude"`

	StoreType *string    `gorm:"default:'regular';column:store_type" json:"store_type"`
	StartDate *time.Time `gorm:"column:start_date" json:"start_date"`
	EndDate   *time.Time `gorm:"column:end_date" json:"end_date"`

	PhoneContact    *string `gorm:"column:phone_contact" json:"phone_contact"`
	WhatsappContact *string `gorm:"column:whatsapp_contact" json:"whatsapp_contact"`
	SocialInstagram *string `gorm:"column:social_instagram" json:"social_instagram"`
	SocialTiktok    *string `gorm:"column:social_tiktok" json:"social_tiktok"`

	IsManuallyClosed bool      `gorm:"default:false;column:is_manually_closed" json:"is_manually_closed"`
	IsActive         bool      `gorm:"default:true;column:is_active" json:"is_active"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`

	Websites   []BusinessWebsite `gorm:"foreignKey:BusinessID" json:"websites"`
	Images     []BusinessImage   `gorm:"foreignKey:BusinessID" json:"images"`
	Hours      []BusinessHour    `gorm:"foreignKey:BusinessID" json:"hours"`
	Categories []Category        `gorm:"many2many:business_categories;foreignKey:BusinessID;joinForeignKey:BusinessID;References:CategoryID;joinReferences:CategoryID" json:"categories"`
	Tags       []Tag             `gorm:"many2many:business_tags;foreignKey:BusinessID;joinForeignKey:BusinessID;References:TagID;joinReferences:TagID" json:"tags"`
}

type BusinessImage struct {
	ImageID    int64  `gorm:"primaryKey;autoIncrement;column:image_id" json:"image_id"`
	BusinessID string  `gorm:"column:business_id" json:"business_id"`
	ImageURL   string `gorm:"column:image_url" json:"image_url"`
}

func (BusinessImage) TableName() string { return "business_images" }

type BusinessWebsite struct {
	WebsiteID  int64  `gorm:"primaryKey;autoIncrement;column:website_id" json:"website_id"`
	BusinessID string  `gorm:"column:business_id" json:"business_id"`
	URL        string `gorm:"column:url" json:"url"`
}

func (BusinessWebsite) TableName() string { return "business_websites" }

type BusinessHour struct {
	BusinessID string   `gorm:"column:business_id" json:"business_id"`
	DayOfWeek  int     `gorm:"column:day_of_week" json:"day_of_week"`
	OpenTime   *string `gorm:"column:open_time" json:"open_time"`
	CloseTime  *string `gorm:"column:close_time" json:"close_time"`
	IsClosed   bool    `gorm:"default:false;column:is_closed" json:"is_closed"`
}

func (BusinessHour) TableName() string { return "business_hours" }

type CreateBusinessInput struct {
	Name            string  `json:"name"`
	EmailContact    string  `json:"email_contact"`
	Description     string  `json:"description"`
	
	Province        string  `json:"province"`
	City            string  `json:"city"`
	AddressDetail   string  `json:"address_detail"`
	Address         string  `json:"address"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	
	StoreType       string  `json:"store_type"`
	StartDate       string  `json:"start_date"`
	EndDate         string  `json:"end_date"`

	PhoneContact    string   `json:"phone_contact"`
	WhatsappContact string   `json:"whatsapp_contact"`
	SocialInstagram string   `json:"social_instagram"`
	SocialTiktok    string   `json:"social_tiktok"`
	LogoURL         string   `json:"logo_url"`

	Websites        []string            `json:"websites"`
	Images          []string            `json:"images"`
	CategoryIDs     []int64             `json:"category_ids"`
	TagIDs          []int64             `json:"tag_ids"`
	Hours           []BusinessHourInput `json:"hours"`
}

type BusinessHourInput struct {
	DayOfWeek int    `json:"day_of_week"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed  bool   `json:"is_closed"`
}

type UpdateBusinessInput struct {
	Name            *string  `json:"name"`
	Handle          *string  `json:"handle"`
	Description     *string  `json:"description"`
	
	Province        *string  `json:"province"`
	City            *string  `json:"city"`
	AddressDetail   *string  `json:"address_detail"`
	Address         *string  `json:"address"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	
	StoreType       *string  `json:"store_type"`
	StartDate       *string  `json:"start_date"`
	EndDate         *string  `json:"end_date"`

	PhoneContact    *string  `json:"phone_contact"`
	WhatsappContact *string  `json:"whatsapp_contact"`
	EmailContact    *string  `json:"email_contact"`
	SocialInstagram *string  `json:"social_instagram"`
	SocialTiktok    *string  `json:"social_tiktok"`
	LogoUrl         *string  `json:"logo_url"`

	Websites    *[]string            `json:"websites"` 
	Images      *[]string            `json:"images"` 
	Hours       *[]BusinessHourInput `json:"hours"`
	CategoryIDs *[]int64             `json:"category_ids"`
	TagIDs      *[]int64             `json:"tag_ids"`
}