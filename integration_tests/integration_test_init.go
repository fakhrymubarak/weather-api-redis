package integrationtest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fakhrymubarak/weather-api-redis/internal/config"
	"github.com/fakhrymubarak/weather-api-redis/internal/handler"
	"github.com/fakhrymubarak/weather-api-redis/internal/middleware"
	"github.com/fakhrymubarak/weather-api-redis/internal/repository"
	"github.com/fakhrymubarak/weather-api-redis/internal/service"

	"github.com/alicebob/miniredis/v2"
)

var (
	miniRedisMock *miniredis.Miniredis
)

func runTestServer() *httptest.Server {
	middleware.StartRateLimiterCleanup()
	return setupIntegrationTestServer()
}

type MockResponse struct {
	Code       int
	Body       *string
	ExactMatch bool
	Method     string
}

func createMockRedisServer() {
	miniRedisMock = miniredis.NewMiniRedis()
	err := miniRedisMock.StartAddr(config.GetRedisAddr())
	if err != nil {
		panic(err)
	}
}

func setupIntegrationTestServer() *httptest.Server {
	// Create a custom http.Client that points to the mock server
	mockClient := &http.Client{
		Transport: &http.Transport{
			Proxy:             nil,
			DialContext:       nil,
			ForceAttemptHTTP2: false,
			MaxIdleConns:      10,
		},
	}

	weatherRepo := repository.NewWeatherRepository(mockClient)
	weatherService := service.NewWeatherService(weatherRepo)
	weatherHandler := handler.NewWeatherHandler(weatherService)

	mux := http.NewServeMux()
	mux.Handle("/weather", middleware.RateLimitMiddleware(http.HandlerFunc(weatherHandler.HandleWeather)))

	srv := &http.Server{
		Addr:              config.GetServerPort(),
		Handler:           mux,
		ReadHeaderTimeout: parseDurationOrDefault(config.GetServerTimeout("read_header_timeout"), 15*time.Second),
		ReadTimeout:       parseDurationOrDefault(config.GetServerTimeout("read_timeout"), 15*time.Second),
		WriteTimeout:      parseDurationOrDefault(config.GetServerTimeout("write_timeout"), 10*time.Second),
		IdleTimeout:       parseDurationOrDefault(config.GetServerTimeout("idle_timeout"), 30*time.Second),
	}

	// Create a channel to communicate server startup
	serverErr := make(chan error, 1)

	// Start a server in a goroutine
	go func() {
		config.GetLogger().Infow("Starting Lookup Server", "port", "8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Return the test server immediately
	return httptest.NewServer(mux)
}

func parseDurationOrDefault(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

// ResetRateLimiterForIntegration Add a helper to reset the rate limiter before integration tests
func ResetRateLimiterForIntegration() {
	middleware.ResetVisitors()
}
