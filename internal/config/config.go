package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	AppPort string

	DBHost            string
	DBUser            string
	DBPass            string
	DBName            string
	DBPort            string
	DBSSLMode         string
	EnableAutoMigrate bool

	DBMaxOpenConns           int
	DBMaxIdleConns           int
	DBConnMaxLifetimeMinutes int
	DBConnMaxIdleTimeMinutes int

	RedisHost string

	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string

	SMTPHost    string
	SMTPPort    int
	SMTPEmail   string
	SMTPPass    string

	MeiliHost      string
	MeiliMasterKey string

	AWSRegion      string
	AWSAccessKey   string
	AWSSecretKey   string
	AWSS3Bucket    string

	AppEnv                   string
	DataEncryptionKey        string
	BlindIndexKey            string
	RequirePhoneVerification bool

	WhatsAppAPIURL        string
	WhatsAppToken         string
	WhatsAppPhoneNumberID string
	WhatsAppVerifyToken   string
}

var Envs *AppConfig

func InitConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, reading from environtment variables...")
	}

	Envs = &AppConfig{
		AppPort:   getEnv("PORT", "3000"),
		AppEnv:    getEnv("APP_ENV", "development"),
		RedisHost: getEnv("REDIS_HOST", "127.0.0.1:6379"),

		DBHost:            getEnvMust("DB_HOST"),
		DBUser:            getEnvMust("DB_USER"),
		DBPass:            getEnvMust("DB_PASSWORD"),
		DBName:            getEnvMust("DB_NAME"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		EnableAutoMigrate: getEnvAsBool("ENABLE_AUTO_MIGRATE", true),

		DBMaxOpenConns:           getEnvAsInt("DB_MAX_OPEN_CONNS", 50),
		DBMaxIdleConns:           getEnvAsInt("DB_MAX_IDLE_CONNS", 15),
		DBConnMaxLifetimeMinutes: getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		DBConnMaxIdleTimeMinutes: getEnvAsInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 5),

		JWTSecret:          getEnvMust("JWT_SECRET"),
		GoogleClientID:     getEnvMust("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: getEnvMust("GOOGLE_CLIENT_SECRET"),

		DataEncryptionKey: getEnv("DATA_ENCRYPTION_KEY", "toentoen-secret-key-32bytes-long!"),
		BlindIndexKey:     getEnv("BLIND_INDEX_KEY", "toentoen-blind-index-key-32bytes!"),

		SMTPHost:    getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:    getEnvAsInt("SMTP_PORT", 587),
		SMTPEmail:   getEnvMust("SMTP_EMAIL"),
		SMTPPass:    getEnvMust("SMTP_PASSWORD"),

		MeiliHost:      getEnv("MEILI_HOST", "http://localhost:7700"),
		MeiliMasterKey: getEnvMust("MEILI_MASTER_KEY"),

		AWSRegion:    getEnvMust("AWS_REGION"),
		AWSAccessKey: getEnvMust("AWS_ACCESS_KEY_ID"),
		AWSSecretKey: getEnvMust("AWS_SECRET_ACCESS_KEY"),
		AWSS3Bucket:  getEnvMust("AWS_S3_BUCKET"),

		RequirePhoneVerification: getEnv("REQUIRE_PHONE_VERIFICATION", "false") == "true",

		WhatsAppAPIURL:        getEnv("WHATSAPP_API_URL", "https://graph.facebook.com/v20.0"),
		WhatsAppToken:         getEnv("WHATSAPP_TOKEN", ""),
		WhatsAppPhoneNumberID: getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
		WhatsAppVerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func getEnvMust(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		log.Fatalf("FATAL ERROR: Variable '%s' has not been set in the .env file", key)
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if value, err := strconv.Atoi(valStr); err == nil {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	valStr := getEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return fallback
	}
	return val
}
