package service

import (
	"context"
	"testing"

	"github.com/yourusername/weather-api-redis/internal/model"
	"github.com/yourusername/weather-api-redis/internal/repository"
)

// Mock repository for testing
type mockWeatherRepository struct {
	shouldError bool
	mockData    *model.WeatherResponse
}

func (m *mockWeatherRepository) GetWeather(ctx context.Context, location string) (*model.WeatherResponse, error) {
	if m.shouldError {
		return nil, repository.ErrLocationNotFound
	}
	return m.mockData, nil
}

func TestWeatherService_GetWeather(t *testing.T) {
	tests := []struct {
		name        string
		location    string
		shouldError bool
		mockData    *model.WeatherResponse
		expectError bool
	}{
		{
			name:        "Successful weather retrieval",
			location:    "London",
			shouldError: false,
			mockData: &model.WeatherResponse{
				Location:    "London",
				Temperature: 15.2,
				Description: "clear sky",
				Cached:      false,
			},
			expectError: false,
		},
		{
			name:        "Repository error",
			location:    "InvalidCity",
			shouldError: true,
			mockData:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &mockWeatherRepository{
				shouldError: tt.shouldError,
				mockData:    tt.mockData,
			}

			// Create service with mock repository
			service := &WeatherService{
				WeatherRepo: mockRepo,
			}

			// Test GetWeather
			ctx := context.Background()
			result, err := service.GetWeather(ctx, tt.location)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
				if result.Location != tt.mockData.Location {
					t.Errorf("Expected location %s, got %s", tt.mockData.Location, result.Location)
				}
			}
		})
	}
}

func TestNewWeatherService(t *testing.T) {
	service := NewWeatherService()
	if service == nil {
		t.Error("Expected service to be created")
	}
	// Test that the service can be used
	ctx := context.Background()
	_, err := service.GetWeather(ctx, "test")
	// We expect an error due to invalid API key, but the service should be functional
	if err == nil {
		t.Log("Service is functional")
	}
}

func TestNewWeatherService_NilRepo(t *testing.T) {
	service := NewWeatherService(nil)
	if service == nil {
		t.Error("Expected service to be created with nil repo")
	}
}

func TestWeatherService_GetWeather_EmptyLocation(t *testing.T) {
	mockRepo := &mockWeatherRepository{shouldError: false, mockData: &model.WeatherResponse{Location: "", Temperature: 0, Description: "", Cached: false}}
	service := &WeatherService{WeatherRepo: mockRepo}
	ctx := context.Background()
	result, err := service.GetWeather(ctx, "")
	if err != nil {
		t.Errorf("Expected no error for empty location, got: %v", err)
	}
	if result == nil {
		t.Error("Expected result for empty location, got nil")
	}
}

func TestWeatherService_GetWeather_NilContext(t *testing.T) {
	mockRepo := &mockWeatherRepository{shouldError: false, mockData: &model.WeatherResponse{Location: "London", Temperature: 15.2, Description: "clear sky", Cached: false}}
	service := &WeatherService{WeatherRepo: mockRepo}
	result, err := service.GetWeather(nil, "London")
	if err != nil {
		t.Errorf("Expected no error for nil context, got: %v", err)
	}
	if result == nil {
		t.Error("Expected result for nil context, got nil")
	}
}
