package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

// Config конфиг
type Config struct {
	ServerPort     string
	DatabaseURL    string
	KafkaBroker    string
	KafkaTopic     string
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	MaxFileSize    int64
	TempDir        string
	LoggerAppName  string
}

// LoadConfig загружает конфиг
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	maxFileSize, _ := strconv.ParseInt(getEnv("MAX_FILE_SIZE", "10485760"), 10, 64)
	minioUseSSL, _ := strconv.ParseBool(getEnv("MINIO_USE_SSL", "false"))

	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "3000"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgresql://user:pass@localhost:5432/images"),
		KafkaBroker:    getEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:     getEnv("KAFKA_TOPIC", "image-processing"),
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "images"),
		MinIOUseSSL:    minioUseSSL,
		MaxFileSize:    maxFileSize,
		TempDir:        getEnv("TEMP_DIR", "./tmp"),
		LoggerAppName:  getEnv("LOGGER_APP_NAME", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
