package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

// Config конфиг
type Config struct {
	Port               string
	DBConnectionString string
	RedisAddr          string
	RedisPassword      string
	RabbitAddress      string
}

// Load загружаем конфиг
func Load() *Config {
	// Загружаем .env файл
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DBConnectionString: getEnv("DATABASE_URL", "postgresql://login:password@localhost:5432/"),
		RedisAddr:          getEnv("REDIS_ADDRESS", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", "redispassword"),
		RabbitAddress:      getEnv("RABBIT_ADDRESS", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
