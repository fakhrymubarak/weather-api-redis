package main

import (
	"net/http"

	"github.com/fakhrymubarak/weather-api-redis/internal/config"
	"github.com/fakhrymubarak/weather-api-redis/internal/handler"
)

func main() {
	weatherHandler := handler.NewWeatherHandler()
	http.HandleFunc("/weather", weatherHandler.HandleWeather)

	port := config.GetServerPort()
	if port == "" {
		port = "8080"
	}
	config.GetLogger().Infow("Weather API server running", "port", port)
	config.GetLogger().Fatalw("Server exited", "error", http.ListenAndServe(":"+port, nil))
}
