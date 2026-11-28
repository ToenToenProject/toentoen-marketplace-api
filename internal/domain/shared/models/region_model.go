package models

// Province represents an Indonesian province (38 provinces).
type Province struct {
	ProvinceCode string `gorm:"primaryKey;column:province_code;size:10" json:"province_code"`
	Name         string `gorm:"column:name;not null;index" json:"name"`
}

func (Province) TableName() string {
	return "provinces"
}

// City represents an Indonesian regency/city (514 kabupaten/kota).
type City struct {
	CityCode     string `gorm:"primaryKey;column:city_code;size:10" json:"city_code"`
	ProvinceCode string `gorm:"column:province_code;size:10;not null;index" json:"province_code"`
	Name         string `gorm:"column:name;not null;index" json:"name"`
}

func (City) TableName() string {
	return "cities"
}
