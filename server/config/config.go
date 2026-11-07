package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	GinMode       string
	MongoURI      string
	MongoDatabase string
	JWTSecret     string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	return Config{
		Port:          getEnv("PORT", "8080"),
		GinMode:       getEnv("GIN_MODE", "debug"),
		MongoURI:      getEnv("MONGO_URI", ""),
		MongoDatabase: getEnv("MONGO_DATABASE", "rfq_costing"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
