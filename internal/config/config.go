package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func GetOpenWeatherMapAPIKey() string {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or failed to load .env")
	}
	return os.Getenv("OPENWEATHERMAP_API_KEY")
}

func GetRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}
