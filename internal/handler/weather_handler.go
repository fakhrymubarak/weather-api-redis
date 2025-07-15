package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yourusername/weather-api-redis/internal/service"
)

type WeatherHandler struct {
	WeatherService service.WeatherServiceInterface
}

func NewWeatherHandler(svc ...service.WeatherServiceInterface) *WeatherHandler {
	var weatherService service.WeatherServiceInterface
	if len(svc) > 0 && svc[0] != nil {
		weatherService = svc[0]
	} else {
		weatherService = service.NewWeatherService()
	}
	return &WeatherHandler{
		WeatherService: weatherService,
	}
}

func (h *WeatherHandler) HandleWeather(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("location")
	if location == "" {
		http.Error(w, "Missing 'location' query parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	weather, err := h.WeatherService.GetWeather(ctx, location)
	if err != nil {
		http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(weather)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusBadRequest)
		return
	}
}
