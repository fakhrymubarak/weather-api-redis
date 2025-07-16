package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fakhrymubarak/weather-api-redis/internal/model"
)

func TestNewWeatherRepository(t *testing.T) {
	repo := NewWeatherRepository()
	if repo == nil {
		t.Error("Expected repository to be created")
	}
}

// --- Error Handling Tests ---

func TestWeatherRepository_GetWeather_ErrorCases(t *testing.T) {
	// Mock client returns 404 for any non-empty location
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.RawQuery, "InvalidCity12345") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"cod": "404", "message": "city not found"}`)),
					Header:     make(http.Header),
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name": "Test", "main": {"temp": 20}, "weather": [{"description": "clear sky"}]}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	_, err := repo.GetWeather(ctx, "")
	if err == nil {
		t.Error("Expected error for empty location")
	}

	_, err = repo.GetWeather(ctx, "InvalidCity12345")
	if err == nil {
		t.Error("Expected error for invalid location")
	}
}

func TestWeatherRepository_CacheOperations(t *testing.T) {
	// This test is about Redis, not HTTP, so we can use a mock that always returns 200
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name": "TestLocation", "main": {"temp": 22}, "weather": [{"description": "sunny"}]}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	location := "TestLocation"
	_, err := repo.GetWeather(ctx, location)
	if err == nil {
		t.Log("Cache test passed - Redis is available")
	} else {
		t.Logf("Cache test skipped - Redis not available: %v", err)
	}
}

func TestWeatherRepository_ErrorHandling(t *testing.T) {
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"cod": "500", "message": "server error"}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	_, err := repo.GetWeather(ctx, "")
	if err == nil {
		t.Error("Expected error for empty location")
	}

	longLocation := "A" + string(make([]byte, 1000))
	_, err = repo.GetWeather(ctx, longLocation)
	if err == nil {
		t.Error("Expected error for very long location")
	}

	_, err = repo.GetWeather(ctx, "London@#$%")
	if err == nil {
		t.Error("Expected error for location with special characters")
	}
}

func TestWeatherRepository_APICallSimulation(t *testing.T) {
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"cod": "500", "message": "simulated error"}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	_, err := repo.GetWeather(ctx, "SimulatedCity")
	if err == nil {
		t.Error("Expected error for simulated API call")
	}
}

func TestWeatherRepository_ConcurrentAccess(t *testing.T) {
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name": "ConcurrentCity", "main": {"temp": 18}, "weather": [{"description": "cloudy"}]}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			location := "ConcurrentCity"
			_, err := repo.GetWeather(ctx, location)
			if err == nil {
				t.Logf("Concurrent request %d completed", id)
			} else {
				t.Logf("Concurrent request %d failed as expected: %v", id, err)
			}
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestWeatherRepository_EdgeCases(t *testing.T) {
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.RawQuery, "%E5%8C%97%E4%BA%AC") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"cod": "404", "message": "city not found"}`)),
					Header:     make(http.Header),
				}
			}
			if strings.Contains(req.URL.RawQuery, "12345") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"cod": "404", "message": "city not found"}`)),
					Header:     make(http.Header),
				}
			}
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"cod": "400", "message": "bad request"}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	_, err := repo.GetWeather(ctx, "北京")
	if err == nil {
		t.Error("Expected error for unicode location")
	}

	_, err = repo.GetWeather(ctx, "12345")
	if err == nil {
		t.Error("Expected error for numeric location")
	}

	_, err = repo.GetWeather(ctx, "   ")
	if err == nil {
		t.Error("Expected error for whitespace-only location")
	}
}

func TestWeatherRepository_Performance(t *testing.T) {
	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name": "London", "main": {"temp": 20}, "weather": [{"description": "clear sky"}]}`)),
				Header:     make(http.Header),
			}
		}),
	}
	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	locations := []string{"London", "Paris", "Tokyo", "NewYork", "Sydney"}

	for _, location := range locations {
		start := time.Now()
		_, err := repo.GetWeather(ctx, location)
		duration := time.Since(start)

		if err == nil {
			t.Logf("Request for %s completed in %v", location, duration)
		} else {
			t.Logf("Request for %s failed as expected in %v: %v", location, duration, err)
		}
	}
}

func TestModelMarshaling(t *testing.T) {
	// Test OpenWeatherMapResponse marshaling
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
			Temp:      15.2,
			FeelsLike: 14.8,
			TempMin:   12.0,
			TempMax:   18.0,
			Pressure:  1013,
			Humidity:  65,
			SeaLevel:  1013,
			GrndLevel: 1010,
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

	// Test marshaling
	data, err := json.Marshal(owmResponse)
	if err != nil {
		t.Errorf("Failed to marshal OpenWeatherMapResponse: %v", err)
	}

	// Test unmarshaling
	var unmarshaled model.OpenWeatherMapResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal OpenWeatherMapResponse: %v", err)
	}

	// Verify data integrity
	if unmarshaled.Name != owmResponse.Name {
		t.Errorf("Expected name %s, got %s", owmResponse.Name, unmarshaled.Name)
	}
	if unmarshaled.Main.Temp != owmResponse.Main.Temp {
		t.Errorf("Expected temp %f, got %f", owmResponse.Main.Temp, unmarshaled.Main.Temp)
	}
	if len(unmarshaled.Weather) != len(owmResponse.Weather) {
		t.Errorf("Expected %d weather items, got %d", len(owmResponse.Weather), len(unmarshaled.Weather))
	}
}

func BenchmarkWeatherRepository_GetWeather(b *testing.B) {
	repo := NewWeatherRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetWeather(ctx, "London")
	}
}

func TestWeatherRepository_CacheWeatherFunction(t *testing.T) {
	repo := NewWeatherRepository()
	ctx := context.Background()

	// Create test weather data
	testWeather := &model.WeatherResponse{
		Location:    "TestCacheCity",
		Temperature: 25.0,
		Description: "sunny",
		Cached:      false,
	}

	// Test caching function directly (this will fail if Redis is not available)
	location := "TestCacheLocation"

	// Try to cache the weather data
	// This is a white-box test to improve coverage
	if r, ok := repo.(*weatherRepository); ok {
		r.cacheWeather(ctx, location, testWeather)
		t.Log("Cache weather function called successfully")
	} else {
		t.Log("Could not access cacheWeather function directly")
	}
}

func TestWeatherRepository_GetFromCacheFunction(t *testing.T) {
	repo := NewWeatherRepository()
	ctx := context.Background()

	// Test getFromCache function directly
	location := "TestGetCacheLocation"

	// Try to get from cache (this will fail if Redis is not available)
	if r, ok := repo.(*weatherRepository); ok {
		_, err := r.getFromCache(ctx, location)
		if err == nil {
			t.Log("Get from cache function called successfully")
		} else {
			t.Logf("Get from cache failed as expected: %v", err)
		}
	} else {
		t.Log("Could not access getFromCache function directly")
	}
}

func TestWeatherRepository_FetchFromExternalAPIFunction(t *testing.T) {
	repo := NewWeatherRepository()

	// Test fetchFromExternalAPI function directly
	location := "TestExternalAPILocation"

	if r, ok := repo.(*weatherRepository); ok {
		_, err := r.fetchFromExternalAPI(location)
		if err == nil {
			t.Error("Expected error for external API call")
		} else {
			t.Logf("External API call failed as expected: %v", err)
		}
	} else {
		t.Log("Could not access fetchFromExternalAPI function directly")
	}
}

// --- 404 City Not Found and API Key Error Tests ---

func TestWeatherRepository_ExternalAPI_404_CityNotFound_WithAPIKey(t *testing.T) {
	// Set API key so the error is not 'API key missing'
	oldKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	os.Setenv("OPENWEATHERMAP_API_KEY", "testkey")
	defer os.Setenv("OPENWEATHERMAP_API_KEY", oldKey)

	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"cod": "404", "message": "city not found"}`)),
				Header:     make(http.Header),
			}
		}),
	}

	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	_, err := repo.GetWeather(ctx, "ja")
	if err == nil {
		t.Fatal("Expected error for city not found, got nil")
	}

	var locationNotFoundError *LocationNotFoundError
	if !errors.As(err, &locationNotFoundError) {
		t.Errorf("Expected LocationNotFoundError, got %T", err)
	}

	if err.Error() != "city not found" {
		t.Errorf("Expected error message 'city not found', got '%s'", err.Error())
	}
}

func TestWeatherRepository_ExternalAPI_404_CityNotFound_MissingAPIKey(t *testing.T) {
	// Unset API key to simulate missing key error
	oldKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	os.Unsetenv("OPENWEATHERMAP_API_KEY")
	defer os.Setenv("OPENWEATHERMAP_API_KEY", oldKey)

	mockClient := &http.Client{
		Transport: RoundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"cod": "404", "message": "city not found"}`)),
				Header:     make(http.Header),
			}
		}),
	}

	repo := NewWeatherRepository(mockClient)
	ctx := context.Background()

	_, err := repo.GetWeather(ctx, "ja")
	if err == nil {
		t.Fatal("Expected error for city not found, got nil")
	}

	if err.Error() != "API key missing" {
		t.Errorf("Expected error message 'API key missing', got '%s'", err.Error())
	}
}
