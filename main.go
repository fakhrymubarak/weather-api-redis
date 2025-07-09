package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// WeatherResponse represents the structure of the weather data returned by the API
type WeatherResponse struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Description string  `json:"description"`
	Cached      bool    `json:"cached"`
}

// fetchWeatherFromProvider simulates fetching weather data from an external provider
func fetchWeatherFromProvider(location string) (*WeatherResponse, error) {
	// TODO: Replace this mock with a real API call
	return &WeatherResponse{
		Location:    location,
		Temperature: 25.0,
		Description: "Sunny",
		Cached:      false,
	}, nil
}

// weatherHandler handles the /weather endpoint
func weatherHandler(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("location")
	if location == "" {
		http.Error(w, "Missing 'location' query parameter", http.StatusBadRequest)
		return
	}

	// Placeholder for future Redis caching logic
	// If Redis is enabled, check cache here
	weather, err := fetchWeatherFromProvider(location)
	if err != nil {
		http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weather)
}

func main() {
	http.HandleFunc("/weather", weatherHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Weather API server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
