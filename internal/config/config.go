package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabaseURL  string
	AdminSecret  string
	ApiKey       string
	StoragePath  string
	AllowOrigins []string
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error reading it, using environment variables")
	}

	return &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/pixstore?sslmode=disable"),
		AdminSecret:  getEnv("ADMIN_SECRET", "change-me-in-prod"),
		ApiKey:       getEnv("API_KEY", "search-api-key"),
		StoragePath:  getEnv("STORAGE_PATH", "./storage"),
		AllowOrigins: strings.Split(getEnv("ALLOW_ORIGINS", "*"), ","),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
