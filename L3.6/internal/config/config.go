package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

// Config конфиг
type Config struct {
	ServerPort  string
	DatabaseURL string
}

// LoadConfig загружает конфиг
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "3000"),
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://user:pass@localhost:5432/sales_tracker"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
