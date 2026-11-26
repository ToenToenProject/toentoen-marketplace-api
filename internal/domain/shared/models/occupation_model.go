package models

// Occupation represents standardized Indonesian national ID (KTP) occupation records.
// IsActive is unexported from JSON responses and used strictly for database filtering.
type Occupation struct {
	ID       int64  `gorm:"primaryKey;column:id" json:"id"`
	Label    string `gorm:"type:varchar(255);uniqueIndex;column:label" json:"label"`
	IsActive bool   `gorm:"column:is_active;default:true" json:"-"`
}

func (Occupation) TableName() string {
	return "occupations"
}
