package service

import (
	"context"
	"errors"

	"github.com/fakhrymubarak/weather-api-redis/internal/model"
	"github.com/fakhrymubarak/weather-api-redis/internal/repository"
)

var (
	ErrWeatherService = errors.New("weather service error")
)

// WeatherServiceInterface defines the interface for weather service operations
type WeatherServiceInterface interface {
	GetWeather(ctx context.Context, location string) (*model.WeatherResponse, error)
}

// WeatherService handles weather-related business logic
type WeatherService struct {
	WeatherRepo repository.WeatherRepository
}

// Ensure the WeatherService implements WeatherServiceInterface
var _ WeatherServiceInterface = (*WeatherService)(nil)

// NewWeatherService creates a new weather service instance
func NewWeatherService(repo ...repository.WeatherRepository) WeatherServiceInterface {
	var weatherRepo repository.WeatherRepository
	if len(repo) > 0 && repo[0] != nil {
		weatherRepo = repo[0]
	} else {
		weatherRepo = repository.NewWeatherRepository()
	}
	return &WeatherService{
		WeatherRepo: weatherRepo,
	}
}

// GetWeather retrieves weather data for a given location
func (s *WeatherService) GetWeather(ctx context.Context, location string) (*model.WeatherResponse, error) {
	// Business logic can be added here (validation, transformation, etc.)
	return s.WeatherRepo.GetWeather(ctx, location)
}
