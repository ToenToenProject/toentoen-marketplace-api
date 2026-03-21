package models

import (
	"time"
)

type Business struct {
	BusinessID        int64          `gorm:"primaryKey;autoIncrement;column:business_id" json:"business_id"`
	AccountID         int64          `gorm:"column:account_id" json:"account_id"`
	Name              string         `gorm:"not null;column:name" json:"name"`
	Handle            *string        `gorm:"unique;column:handle" json:"handle"`
	Status            string         `gorm:"default:'draft';column:status" json:"status"`
	SubscriptionTier  string         `gorm:"default:'free';column:subscription_tier" json:"subscription_tier"`
	LogoURL           *string        `gorm:"column:logo_url" json:"logo_url"`
	Description       *string        `gorm:"column:description" json:"description"`
	
	Address           *string        `gorm:"column:address" json:"address"`
	Latitude          *float64       `gorm:"column:latitude" json:"latitude"`
	Longitude         *float64       `gorm:"column:longitude" json:"longitude"`
	
	PhoneContact      *string        `gorm:"column:phone_contact" json:"phone_contact"`
	WhatsappContact   *string        `gorm:"column:whatsapp_contact" json:"whatsapp_contact"`
	EmailContact      *string        `gorm:"column:email_contact" json:"email_contact"`
	WebsiteURL        *string        `gorm:"column:website_url" json:"website_url"`
	SocialInstagram   *string        `gorm:"column:social_instagram" json:"social_instagram"`
	SocialTiktok      *string        `gorm:"column:social_tiktok" json:"social_tiktok"`
	
	IsManuallyClosed  bool           `gorm:"default:false;column:is_manually_closed" json:"is_manually_closed"`
	IsActive          bool           `gorm:"default:true;column:is_active" json:"is_active"`
	CreatedAt         time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at" json:"updated_at"`

	Hours      []BusinessHour `gorm:"foreignKey:BusinessID" json:"hours"`
	Categories []Category     `gorm:"many2many:business_categories;foreignKey:BusinessID;joinForeignKey:BusinessID;References:CategoryID;joinReferences:CategoryID" json:"categories"`
	Tags       []Tag          `gorm:"many2many:business_tags;foreignKey:BusinessID;joinForeignKey:BusinessID;References:TagID;joinReferences:TagID" json:"tags"`
}

type BusinessHour struct {
	BusinessID int64   `gorm:"column:business_id" json:"business_id"`
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
	Address         string  `json:"address"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	PhoneContact    string  `json:"phone_contact"`
	WhatsappContact string  `json:"whatsapp_contact"`
	SocialInstagram string  `json:"social_instagram"`
	SocialTiktok    string  `json:"social_tiktok"`
	WebsiteURL      string  `json:"website_url"`
	LogoURL         string  `json:"logo_url"`

	CategoryIDs []int64             `json:"category_ids"`
	TagIDs      []int64             `json:"tag_ids"`
	Hours       []BusinessHourInput `json:"hours"`
}

type BusinessHourInput struct {
	DayOfWeek int    `json:"day_of_week"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed  bool   `json:"is_closed"`
}

// Data Transfer Object untuk Update Business (Menggunakan Pointer)
type UpdateBusinessInput struct {
	Name            *string  `json:"name"`
	Handle          *string  `json:"handle"`
	Description     *string  `json:"description"`
	Address         *string  `json:"address"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	PhoneContact    *string  `json:"phone_contact"`
	WhatsappContact *string  `json:"whatsapp_contact"`
	EmailContact    *string  `json:"email_contact"`
	WebsiteUrl      *string  `json:"website_url"`
	SocialInstagram *string  `json:"social_instagram"`
	SocialTiktok    *string  `json:"social_tiktok"`
	LogoUrl         *string  `json:"logo_url"`

	Hours       *[]BusinessHourInput `json:"hours"`
	CategoryIDs *[]int64             `json:"category_ids"`
	TagIDs      *[]int64             `json:"tag_ids"`
}