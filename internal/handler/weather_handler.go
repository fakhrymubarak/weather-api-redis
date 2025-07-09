package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/weather-api-redis/internal/service"
)

func WeatherHandler(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("location")
	if location == "" {
		http.Error(w, "Missing 'location' query parameter", http.StatusBadRequest)
		return
	}

	// Placeholder for future Redis caching logic
	weather, err := service.FetchWeatherFromProvider(location)
	if err != nil {
		http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weather)
}
