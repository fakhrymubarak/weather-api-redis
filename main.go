package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yourusername/weather-api-redis/internal/handler"
)

func main() {
	weatherHandler := handler.NewWeatherHandler()
	http.HandleFunc("/weather", weatherHandler.HandleWeather)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Weather API server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
