package integrationtest

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/yourusername/weather-api-redis/internal/handler"
	"github.com/yourusername/weather-api-redis/internal/repository"
	"github.com/yourusername/weather-api-redis/internal/service"

	"github.com/alicebob/miniredis/v2"
)

var (
	miniRedisMock *miniredis.Miniredis
)

func runTestServer() *httptest.Server {
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
	err := miniRedisMock.StartAddr(":16379")
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

	mux := http.NewServeMux()
	mux.HandleFunc("/weather", handler.NewWeatherHandler(weatherService).HandleWeather)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	// Create a channel to communicate server startup
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Lookup Server on %s", "8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Return the test server immediately
	return httptest.NewServer(mux)
}
