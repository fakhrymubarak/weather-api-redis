package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fakhrymubarak/weather-api-redis/internal/model"
	"github.com/fakhrymubarak/weather-api-redis/internal/service"
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

func (h *WeatherHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *WeatherHandler) HandleWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errMsg := "Method not allowed"
		w.Header().Set("Allow", http.MethodGet)
		h.writeJSONResponse(w, http.StatusMethodNotAllowed, model.Response{
			Error:   &errMsg,
			Message: "Error",
		})
		return
	}

	location := r.URL.Query().Get("location")
	if location == "" {
		errMsg := "Missing 'location' query parameter"
		h.writeJSONResponse(w, http.StatusBadRequest, model.Response{
			Error:   &errMsg,
			Message: "Error",
		})
		return
	}

	ctx := context.Background()
	weather, err := h.WeatherService.GetWeather(ctx, location)
	if err != nil {
		// Check for downstream city not found error
		if err.Error() == "city not found" || err.Error() == "location not found" {
			errMsg := err.Error()
			h.writeJSONResponse(w, http.StatusNotFound, model.Response{
				Error:   &errMsg,
				Message: "Error",
			})
			return
		}
		errMsg := "Failed to fetch weather data"
		h.writeJSONResponse(w, http.StatusInternalServerError, model.Response{
			Error:   &errMsg,
			Message: "Error",
		})
		return
	}

	h.writeJSONResponse(w, http.StatusOK, model.Response{
		Data:    weather,
		Message: "Success",
	})
}
