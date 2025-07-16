package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"github.com/yourusername/weather-api-redis/internal/config"
	"github.com/yourusername/weather-api-redis/internal/model"
	"github.com/yourusername/weather-api-redis/internal/redis"
)

// Custom error types
var (
	ErrLocationNotFound = errors.New("location not found")
	ErrAPIKeyMissing    = errors.New("API key missing")
	ErrExternalAPI      = errors.New("external API error")
)

// WeatherRepository defines the interface for weather data access
type WeatherRepository interface {
	GetWeather(ctx context.Context, location string) (*model.WeatherResponse, error)
}

// weatherRepository implements WeatherRepository
type weatherRepository struct {
	redisClient *redisv9.Client
	httpClient  *http.Client
}

// NewWeatherRepository creates a new weather repository instance
func NewWeatherRepository(httpClient ...*http.Client) WeatherRepository {
	client := http.DefaultClient
	if len(httpClient) > 0 && httpClient[0] != nil {
		client = httpClient[0]
	}
	return &weatherRepository{
		redisClient: redis.GetClient(),
		httpClient:  client,
	}
}

// GetWeather retrieves weather data, checking cache first, then external API
func (r *weatherRepository) GetWeather(ctx context.Context, location string) (*model.WeatherResponse, error) {
	// Try to get from cache first
	if cached, err := r.getFromCache(ctx, location); err == nil {
		fmt.Println("[DEBUG] Cache hit for:", location)
		return cached, nil
	} else {
		fmt.Println("[DEBUG] Cache miss for:", location, "error:", err)
	}

	// If not in cache, fetch from external API
	weather, err := r.fetchFromExternalAPI(location)
	if err != nil {
		fmt.Println("[DEBUG] External API error for:", location, "error:", err)
		return nil, err
	}
	fmt.Println("[DEBUG] Fetched from API:", location)

	// Cache the result
	r.cacheWeather(ctx, location, weather)

	return weather, nil
}

// getFromCache retrieves weather data from Redis cache
func (r *weatherRepository) getFromCache(ctx context.Context, location string) (*model.WeatherResponse, error) {
	cacheKey := "weather:" + location

	val, err := r.redisClient.Get(ctx, cacheKey).Result()
	if err != nil {
		fmt.Println("[DEBUG] Redis get error for:", cacheKey, "error:", err)
		return nil, err
	}

	fmt.Println("[DEBUG] Redis get success for:", cacheKey, "value:", val)

	var weather model.WeatherResponse
	if err := json.Unmarshal([]byte(val), &weather); err != nil {
		fmt.Println("[DEBUG] Unmarshal error for:", cacheKey, "error:", err)
		return nil, err
	}

	weather.Cached = true
	return &weather, nil
}

// fetchFromExternalAPI retrieves weather data from OpenWeatherMap API
func (r *weatherRepository) fetchFromExternalAPI(location string) (*model.WeatherResponse, error) {
	fmt.Println("[DEBUG] Fetching from external API for:", location)
	apiKey := config.GetOpenWeatherMapAPIKey()
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	apiURL := config.GetOpenWeatherApiUrl()
	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", apiURL, location, apiKey)
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, ErrExternalAPI
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrLocationNotFound
		}
		return nil, ErrExternalAPI
	}

	var data model.OpenWeatherMapResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	weather := &model.WeatherResponse{
		Location:    data.Name,
		Temperature: data.Main.Temp,
		Description: "",
		Cached:      false,
	}

	if len(data.Weather) > 0 {
		weather.Description = data.Weather[0].Description
	}

	return weather, nil
}

// cacheWeather stores weather data in Redis cache
func (r *weatherRepository) cacheWeather(ctx context.Context, location string, weather *model.WeatherResponse) {
	cacheKey := "weather:" + location

	if b, err := json.Marshal(weather); err == nil {
		dur, err := time.ParseDuration(config.GetCacheExpiration())
		if err != nil {
			dur = 10 * time.Minute // fallback
		}
		_ = r.redisClient.Set(ctx, cacheKey, b, dur).Err()
	}
}
