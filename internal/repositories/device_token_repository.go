package repositories

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type DeviceToken struct { // Sesuaikan nama struct dengan gaya Anda, misal di models
	TokenID    int64     `gorm:"primaryKey"`
	AccountID  string    `gorm:"not null"`
	FCMToken   string    `gorm:"not null;unique"`
	DeviceType string    
	UpdatedAt  time.Time 
}

func SaveDeviceToken(db *gorm.DB, accountID, fcmToken, deviceType string) error {
	token := DeviceToken{
		AccountID:  accountID,
		FCMToken:   fcmToken,
		DeviceType: deviceType,
		UpdatedAt:  time.Now(),
	}

	// Upsert: Kalau token sudah ada, update account_id dan waktunya. Kalau belum, create baru.
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fcm_token"}},
		DoUpdates: clause.AssignmentColumns([]string{"account_id", "updated_at"}),
	}).Create(&token).Error
}

// Mengambil semua token milik owner yang relevan
func GetTokensByAccountIDs(db *gorm.DB, accountIDs []string) ([]string, error) {
	var tokens []string
	err := db.Model(&DeviceToken{}).Where("account_id IN ?", accountIDs).Pluck("fcm_token", &tokens).Error
	return tokens, err
}