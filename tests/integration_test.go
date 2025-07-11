package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yourusername/weather-api-redis/internal/handler"
	"github.com/yourusername/weather-api-redis/internal/model"
	"github.com/yourusername/weather-api-redis/internal/redis"
)

func TestWeatherHandler_Integration(t *testing.T) {
	// Set up test environment
	os.Setenv("OPENWEATHERMAP_API_KEY", "test_api_key")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	// Create handler
	weatherHandler := handler.NewWeatherHandler()

	tests := []struct {
		name           string
		location       string
		expectedStatus int
		expectedError  bool
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "Missing location parameter",
			location:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			checkResponse: func(t *testing.T, resp *http.Response) {
				// Check error message
				body := make([]byte, 100)
				resp.Body.Read(body)
				if string(body) == "" {
					t.Error("Expected error message for missing location")
				}
			},
		},
		{
			name:           "Invalid API key",
			location:       "London",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			checkResponse: func(t *testing.T, resp *http.Response) {
				// Should return internal server error due to invalid API key
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/weather?location=%s", tt.location), nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			weatherHandler.HandleWeather(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Check response if provided
			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Result())
			}
		})
	}
}

func TestWeatherService_Integration(t *testing.T) {
	// Set up test environment
	os.Setenv("OPENWEATHERMAP_API_KEY", "test_api_key")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	// Test service directly
	weatherHandler := handler.NewWeatherHandler()

	// Test with invalid API key (should fail)
	ctx := context.Background()
	_, err := weatherHandler.WeatherService.GetWeather(ctx, "London")
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
}

func TestRedisIntegration(t *testing.T) {
	// Test Redis connection
	client := redis.GetClient()
	ctx := redis.GetContext()

	// Test basic Redis operations
	testKey := "test:weather:integration"
	testData := &model.WeatherResponse{
		Location:    "Test City",
		Temperature: 25.0,
		Description: "sunny",
		Cached:      false,
	}

	// Marshal test data
	data, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Set test data
	err = client.Set(ctx, testKey, data, time.Minute).Err()
	if err != nil {
		t.Logf("Redis not available, skipping Redis tests: %v", err)
		return
	}

	// Get test data
	val, err := client.Get(ctx, testKey).Result()
	if err != nil {
		t.Errorf("Failed to get data from Redis: %v", err)
	}

	// Unmarshal and verify
	var retrieved model.WeatherResponse
	err = json.Unmarshal([]byte(val), &retrieved)
	if err != nil {
		t.Errorf("Failed to unmarshal data from Redis: %v", err)
	}

	if retrieved.Location != testData.Location {
		t.Errorf("Expected location %s, got %s", testData.Location, retrieved.Location)
	}

	// Clean up
	client.Del(ctx, testKey)
}

func TestModelStructures(t *testing.T) {
	// Test WeatherResponse marshaling/unmarshaling
	weather := &model.WeatherResponse{
		Location:    "Test City",
		Temperature: 20.5,
		Description: "cloudy",
		Cached:      true,
	}

	data, err := json.Marshal(weather)
	if err != nil {
		t.Errorf("Failed to marshal WeatherResponse: %v", err)
	}

	var unmarshaled model.WeatherResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal WeatherResponse: %v", err)
	}

	if unmarshaled.Location != weather.Location {
		t.Errorf("Expected location %s, got %s", weather.Location, unmarshaled.Location)
	}

	// Test OpenWeatherMapResponse
	owmResponse := &model.OpenWeatherMapResponse{
		Name: "London",
		Main: struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			TempMin   float64 `json:"temp_min"`
			TempMax   float64 `json:"temp_max"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
			SeaLevel  int     `json:"sea_level"`
			GrndLevel int     `json:"grnd_level"`
		}{
			Temp: 15.2,
		},
		Weather: []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		}{
			{
				ID:          800,
				Main:        "Clear",
				Description: "clear sky",
				Icon:        "01d",
			},
		},
	}

	data, err = json.Marshal(owmResponse)
	if err != nil {
		t.Errorf("Failed to marshal OpenWeatherMapResponse: %v", err)
	}

	var unmarshaledOWM model.OpenWeatherMapResponse
	err = json.Unmarshal(data, &unmarshaledOWM)
	if err != nil {
		t.Errorf("Failed to unmarshal OpenWeatherMapResponse: %v", err)
	}

	if unmarshaledOWM.Name != owmResponse.Name {
		t.Errorf("Expected name %s, got %s", owmResponse.Name, unmarshaledOWM.Name)
	}
}

func TestHandlerErrorCases(t *testing.T) {
	weatherHandler := handler.NewWeatherHandler()

	// Test with empty location
	req, _ := http.NewRequest(http.MethodGet, "/weather", nil)
	rr := httptest.NewRecorder()
	weatherHandler.HandleWeather(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}

	// Test with malformed query parameter
	req, _ = http.NewRequest(http.MethodGet, "/weather?location=", nil)
	rr = httptest.NewRecorder()
	weatherHandler.HandleWeather(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}
}

func BenchmarkWeatherHandler(b *testing.B) {
	weatherHandler := handler.NewWeatherHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/weather?location=London", nil)
		rr := httptest.NewRecorder()
		weatherHandler.HandleWeather(rr, req)
	}
}
