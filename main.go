package main

import (
	"net/http"

	"github.com/fakhrymubarak/weather-api-redis/internal/config"
	"github.com/fakhrymubarak/weather-api-redis/internal/handler"
	"github.com/fakhrymubarak/weather-api-redis/internal/middleware"
)

func main() {
	middleware.StartRateLimiterCleanup()
	weatherHandler := handler.NewWeatherHandler()
	mux := http.NewServeMux()
	mux.Handle("/weather", middleware.RateLimitMiddleware(http.HandlerFunc(weatherHandler.HandleWeather)))

	port := config.GetServerPort()
	if port == "" {
		port = "8080"
	}
	config.GetLogger().Infow("Weather API server running", "port", port)
	config.GetLogger().Fatalw("Server exited", "error", http.ListenAndServe(":"+port, mux))
}
