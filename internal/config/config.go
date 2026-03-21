package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	AppPort string

	DBHost string
	DBUser string
	DBPass string
	DBName string
	DBPort string

	RedisHost string

	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string

	SMTPHost    string
	SMTPPort    int
	SMTPEmail   string
	SMTPPass    string
	FonnteToken string
}

var Envs *AppConfig

func InitConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, reading from environtment variables...")
	}

	Envs = &AppConfig{
		AppPort:   getEnv("PORT", "3000"),
		RedisHost: getEnv("REDIS_HOST", "127.0.0.1:6379"),

		DBHost: getEnvMust("DB_HOST"),
		DBUser: getEnvMust("DB_USER"),
		DBPass: getEnvMust("DB_PASSWORD"),
		DBName: getEnvMust("DB_NAME"),
		DBPort: getEnv("DB_PORT", "5432"),

		JWTSecret:          getEnvMust("JWT_SECRET"),
		GoogleClientID:     getEnvMust("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: getEnvMust("GOOGLE_CLIENT_SECRET"),

		SMTPHost:    getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:    getEnvAsInt("SMTP_PORT", 587),
		SMTPEmail:   getEnvMust("SMTP_EMAIL"),
		SMTPPass:    getEnvMust("SMTP_PASSWORD"),
		FonnteToken: getEnvMust("FONNTE_TOKEN"),
	}
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return value
	}
	return fallback
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
