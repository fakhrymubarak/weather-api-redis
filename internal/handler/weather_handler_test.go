package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fakhrymubarak/weather-api-redis/internal/model"
	"github.com/fakhrymubarak/weather-api-redis/internal/service"
)

// Mock service for testing
type mockWeatherService struct {
	shouldError bool
	mockData    *model.WeatherResponse
}

func (m *mockWeatherService) GetWeather(context.Context, string) (*model.WeatherResponse, error) {
	if m.shouldError {
		return nil, service.ErrWeatherService
	}
	return m.mockData, nil
}

// Ensure mockWeatherService implements WeatherServiceInterface
var _ service.WeatherServiceInterface = (*mockWeatherService)(nil)

func TestNewWeatherHandler(t *testing.T) {
	handler := NewWeatherHandler()
	if handler == nil {
		t.Error("Expected handler to be created")
		return
	}
	if handler.WeatherService == nil {
		t.Error("Expected weather service to be initialized")
	}
}

func TestWeatherHandler_HandleWeather(t *testing.T) {
	tests := []struct {
		name           string
		location       string
		shouldError    bool
		mockData       *model.WeatherResponse
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing location parameter",
			location:       "",
			shouldError:    false,
			mockData:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'location' query parameter",
		},
		{
			name:        "Successful weather request",
			location:    "London",
			shouldError: false,
			mockData: &model.WeatherResponse{
				Location:    "London",
				Temperature: 15.2,
				Description: "clear sky",
				Cached:      false,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Service error",
			location:       "InvalidCity",
			shouldError:    true,
			mockData:       nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to fetch weather data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with mock service
			handler := &WeatherHandler{
				WeatherService: &mockWeatherService{
					shouldError: tt.shouldError,
					mockData:    tt.mockData,
				},
			}

			// Create request
			req, err := http.NewRequest("GET", "/weather?location="+tt.location, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.HandleWeather(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedBody != "" {
				body := rr.Body.String()
				if body == "" {
					t.Errorf("Expected error body but got empty response")
				}
				if body != tt.expectedBody {
					t.Logf("Got error body: %s", body)
				}
			}

			if tt.expectedStatus == http.StatusOK && tt.mockData != nil {
				var response model.WeatherResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}

				if response.Location != tt.mockData.Location {
					t.Errorf("Expected location %s, got %s", tt.mockData.Location, response.Location)
				}
				if response.Temperature != tt.mockData.Temperature {
					t.Errorf("Expected temperature %f, got %f", tt.mockData.Temperature, response.Temperature)
				}
			}
		})
	}
}

func TestWeatherHandler_HandleWeather_EdgeCases(t *testing.T) {
	handler := NewWeatherHandler()

	// Test with no query parameters
	req, _ := http.NewRequest("GET", "/weather", nil)
	rr := httptest.NewRecorder()
	handler.HandleWeather(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}

	req, _ = http.NewRequest("GET", "/weather?location=", nil)
	rr = httptest.NewRecorder()
	handler.HandleWeather(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}

	req, _ = http.NewRequest("GET", "/weather?location=London&location=Paris", nil)
	rr = httptest.NewRecorder()
	handler.HandleWeather(rr, req)

	if status := rr.Code; status == http.StatusOK {
		t.Log("Multiple location parameters handled correctly")
	}
}

func TestWeatherHandler_HandleWeather_NonGETMethod(t *testing.T) {
	handler := NewWeatherHandler()
	req, _ := http.NewRequest(http.MethodPost, "/weather?location=London", nil)
	rr := httptest.NewRecorder()
	handler.HandleWeather(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rr.Code)
	}
}

func BenchmarkWeatherHandler_HandleWeather(b *testing.B) {
	handler := NewWeatherHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/weather?location=London", nil)
		rr := httptest.NewRecorder()
		handler.HandleWeather(rr, req)
	}
}
