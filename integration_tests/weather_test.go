package integrationtest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/fakhrymubarak/weather-api-redis/internal/config"
	"github.com/fakhrymubarak/weather-api-redis/internal/handler"
	"github.com/fakhrymubarak/weather-api-redis/internal/model"
	"github.com/fakhrymubarak/weather-api-redis/internal/redis"
	"github.com/fakhrymubarak/weather-api-redis/internal/repository"
	"github.com/fakhrymubarak/weather-api-redis/internal/service"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type WeatherAPITestSuite struct {
	suite.Suite
	httpServer *httptest.Server
	miniRedis  *miniredis.Miniredis
}

func (suite *WeatherAPITestSuite) SetupSuite() {
	createMockRedisServer()
	suite.miniRedis = miniRedisMock
	viper.Set("redis.addr", miniRedisMock.Addr())

	config.ReloadConfigForTest()
	redis.ResetClientForTest()
	// In SetupSuite, set the environment variable for the API key
	os.Setenv("OPENWEATHERMAP_API_KEY", "test_api_key")

	// Start a mock OpenWeatherMap API server
	mockOWM := mockOWMApi()
	// Set the API URL in Viper to the mock server's URL
	viper.Set("openweathermap.api_url", mockOWM.URL)
	viper.Set("openweathermap.api_key", "test_api_key")
	config.ReloadConfigForTest()

	// Remove the custom HTTP client and roundTripperFunc
	// Inject the default client into the repository
	weatherRepo := repository.NewWeatherRepository()
	weatherService := service.NewWeatherService(weatherRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/weather", handler.NewWeatherHandler(weatherService).HandleWeather)

	suite.httpServer = runTestServer()
}

func (suite *WeatherAPITestSuite) TearDownSuite() {
	if suite.httpServer != nil {
		suite.httpServer.Close()
	}
	if suite.miniRedis != nil {
		suite.miniRedis.Close()
	}
}

func TestWeatherAPITestSuite(t *testing.T) {
	suite.Run(t, new(WeatherAPITestSuite))
}

func (suite *WeatherAPITestSuite) TestWeatherEndpoint() {
	tests := []struct {
		name          string
		setupMockTest func()
		setupRequest  func() *http.Request
		wantStatus    int
		validate      func(t *testing.T, resp *http.Response)
	}{
		{
			name: "Failed - Missing location parameter",
			setupMockTest: func() {
				// No Redis setup needed for this test
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather", nil)
				return req
			},
			wantStatus: http.StatusBadRequest,
			validate: func(t *testing.T, resp *http.Response) {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "Missing 'location' query parameter")
			},
		},
		{
			name: "Failed - Empty location parameter",
			setupMockTest: func() {
				// No Redis setup needed for this test
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=", nil)
				return req
			},
			wantStatus: http.StatusBadRequest,
			validate: func(t *testing.T, resp *http.Response) {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "Missing 'location' query parameter")
			},
		},
		{
			name: "Failed - Invalid API key",
			setupMockTest: func() {
				// Clear any cached data for this test
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:London")

				// Set an invalid API key for this test
				os.Setenv("OPENWEATHERMAP_API_KEY", "invalid_key")
				config.ReloadConfigForTest()
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=London", nil)
				return req
			},
			wantStatus: http.StatusInternalServerError,
			validate: func(t *testing.T, resp *http.Response) {
				// Restore a valid API key after a test
				os.Setenv("OPENWEATHERMAP_API_KEY", "test_api_key")
				config.ReloadConfigForTest()
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "Failed to fetch weather data")
			},
		},
		{
			name: "Failed - Invalid location",
			setupMockTest: func() {
				// Clear any cached data for this test
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:InvalidCity12345")
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=InvalidCity12345", nil)
				return req
			},
			wantStatus: http.StatusInternalServerError,
			validate: func(t *testing.T, resp *http.Response) {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "Failed to fetch weather data")
			},
		},
		{
			name: "Success - Valid location (cached)",
			setupMockTest: func() {
				// Clear cache before setting up cached data
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:London")

				// Setup Redis with cached data
				cachedWeather := &model.WeatherResponse{
					Location:    "London",
					Temperature: 15.2,
					Description: "clear sky",
					Cached:      true,
				}

				data, _ := json.Marshal(cachedWeather)
				client.Set(ctx, "weather:London", data, time.Minute)
				time.Sleep(50 * time.Millisecond)
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=London", nil)
				return req
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp *http.Response) {
				var weather model.WeatherResponse
				err := json.NewDecoder(resp.Body).Decode(&weather)
				assert.NoError(t, err)
				assert.Equal(t, "London", weather.Location)
				assert.True(t, weather.Cached)
			},
		},
		{
			name: "Success - Valid location (not-cached)",
			setupMockTest: func() {
				// Clear cache before running a not-cached test
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:London")
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=London", nil)
				return req
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp *http.Response) {
				var weather model.WeatherResponse
				err := json.NewDecoder(resp.Body).Decode(&weather)
				assert.NoError(t, err)
				assert.Equal(t, "London", weather.Location)
				assert.False(t, weather.Cached)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMockTest()

			req := tt.setupRequest()

			resp, err := suite.httpServer.Client().Do(req)
			assert.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), tt.wantStatus, resp.StatusCode)

			if tt.validate != nil {
				tt.validate(suite.T(), resp)
			}
		})
	}
}

func mockOWMApi() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		apiKey := r.URL.Query().Get("appid")
		if apiKey != "test_api_key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"cod":401,"message":"Invalid API key"}`))
			return
		}
		if q == "London" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, err := os.ReadFile("integration_tests/openweathermap_mock_london.json")
			if err != nil {
				_, _ = w.Write([]byte(`{"name": "London", "main": {"temp": 15.2}, "weather": [{"description": "clear sky"}]}`))
				return
			}
			_, _ = w.Write(data)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"cod": "404", "message": "city not found"}`))
	}))

}
