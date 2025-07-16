package repository

import (
	"bytes"
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
	redisv9 "github.com/redis/go-redis/v9"
)

type mockRedisClient struct {
	getFunc func(ctx context.Context, key string) *redisv9.StringCmd
	setFunc func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redisv9.StringCmd {
	return m.getFunc(ctx, key)
}
func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
	return m.setFunc(ctx, key, value, expiration)
}

// Implement only the methods used in the repo
func (m *mockRedisClient) Close() error { return nil }

// ... other methods can panic if called ...

// Mock HTTP client
func newMockHTTPClient(fn func(req *http.Request) *http.Response) *http.Client {
	return &http.Client{
		Transport: RoundTripperFunc(fn),
	}
}

func TestGetWeather_CacheHit(t *testing.T) {
	cached := &model.WeatherResponse{
		Location:    "London",
		Temperature: 20.0,
		Description: "clear sky",
		Cached:      true,
	}
	b, _ := json.Marshal(cached)
	mockRedis := &mockRedisClient{
		getFunc: func(ctx context.Context, key string) *redisv9.StringCmd {
			cmd := redisv9.NewStringResult(string(b), nil)
			return cmd
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
			return redisv9.NewStatusResult("OK", nil)
		},
	}
	repo := &weatherRepository{
		redisClient: mockRedis,
		httpClient:  http.DefaultClient,
	}
	ctx := context.Background()
	weather, err := repo.GetWeather(ctx, "London")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !weather.Cached {
		t.Errorf("Expected Cached=true, got false")
	}
	if weather.Location != "London" {
		t.Errorf("Expected London, got %s", weather.Location)
	}
}

func TestGetWeather_CacheMiss_APISuccess(t *testing.T) {
	os.Setenv("OPENWEATHERMAP_API_KEY", "testkey")
	defer os.Unsetenv("OPENWEATHERMAP_API_KEY")
	mockRedis := &mockRedisClient{
		getFunc: func(ctx context.Context, key string) *redisv9.StringCmd {
			return redisv9.NewStringResult("", errors.New("cache miss"))
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
			return redisv9.NewStatusResult("OK", nil)
		},
	}
	mockResp := model.OpenWeatherMapResponse{
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
		}{Temp: 21.5},
		Weather: []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		}{{Description: "sunny"}},
	}
	b, _ := json.Marshal(mockResp)
	mockHTTP := newMockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(b)),
			Header:     make(http.Header),
		}
	})
	repo := &weatherRepository{
		redisClient: mockRedis,
		httpClient:  mockHTTP,
	}
	ctx := context.Background()
	weather, err := repo.GetWeather(ctx, "London")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if weather.Cached {
		t.Errorf("Expected Cached=false, got true")
	}
	if weather.Location != "London" {
		t.Errorf("Expected London, got %s", weather.Location)
	}
}

func TestGetWeather_CacheMiss_APIError(t *testing.T) {
	os.Setenv("OPENWEATHERMAP_API_KEY", "testkey")
	defer os.Unsetenv("OPENWEATHERMAP_API_KEY")
	mockRedis := &mockRedisClient{
		getFunc: func(ctx context.Context, key string) *redisv9.StringCmd {
			return redisv9.NewStringResult("", errors.New("cache miss"))
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
			return redisv9.NewStatusResult("OK", nil)
		},
	}
	mockHTTP := newMockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader("error")),
			Header:     make(http.Header),
		}
	})
	repo := &weatherRepository{
		redisClient: mockRedis,
		httpClient:  mockHTTP,
	}
	ctx := context.Background()
	_, err := repo.GetWeather(ctx, "London")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetWeather_CacheMiss_APIDecodeError(t *testing.T) {
	os.Setenv("OPENWEATHERMAP_API_KEY", "testkey")
	defer os.Unsetenv("OPENWEATHERMAP_API_KEY")
	mockRedis := &mockRedisClient{
		getFunc: func(ctx context.Context, key string) *redisv9.StringCmd {
			return redisv9.NewStringResult("", errors.New("cache miss"))
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
			return redisv9.NewStatusResult("OK", nil)
		},
	}
	mockHTTP := newMockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("not-json")),
			Header:     make(http.Header),
		}
	})
	repo := &weatherRepository{
		redisClient: mockRedis,
		httpClient:  mockHTTP,
	}
	ctx := context.Background()
	_, err := repo.GetWeather(ctx, "London")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetWeather_MissingAPIKey(t *testing.T) {
	os.Unsetenv("OPENWEATHERMAP_API_KEY")
	mockRedis := &mockRedisClient{
		getFunc: func(ctx context.Context, key string) *redisv9.StringCmd {
			return redisv9.NewStringResult("", errors.New("cache miss"))
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
			return redisv9.NewStatusResult("OK", nil)
		},
	}
	mockHTTP := newMockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Header:     make(http.Header),
		}
	})
	repo := &weatherRepository{
		redisClient: mockRedis,
		httpClient:  mockHTTP,
	}
	ctx := context.Background()
	_, err := repo.GetWeather(ctx, "London")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetFromCache_UnmarshalError(t *testing.T) {
	mockRedis := &mockRedisClient{
		getFunc: func(ctx context.Context, key string) *redisv9.StringCmd {
			return redisv9.NewStringResult("not-json", nil)
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redisv9.StatusCmd {
			return redisv9.NewStatusResult("OK", nil)
		},
	}
	repo := &weatherRepository{
		redisClient: mockRedis,
		httpClient:  http.DefaultClient,
	}
	ctx := context.Background()
	_, err := repo.getFromCache(ctx, "London")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}
