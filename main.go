package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yourusername/weather-api-redis/internal/handler"
)

func main() {
	http.HandleFunc("/weather", handler.WeatherHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Weather API server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
