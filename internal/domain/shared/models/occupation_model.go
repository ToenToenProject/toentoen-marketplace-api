package models

// Occupation merepresentasikan data pekerjaan yang tercantum di KTP Indonesia.
// Field IsActive tidak di-expose ke response JSON (json:"-"), hanya digunakan
// untuk filter di level database.
type Occupation struct {
	ID       int64  `gorm:"primaryKey;column:id" json:"id"`
	Label    string `gorm:"type:varchar(255);uniqueIndex;column:label" json:"label"`
	IsActive bool   `gorm:"column:is_active;default:true" json:"-"`
}

func (Occupation) TableName() string {
	return "occupations"
}
