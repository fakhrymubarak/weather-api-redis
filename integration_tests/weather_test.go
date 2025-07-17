package integrationtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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

var ProvidedCities = []string{
	"Jakarta", "Surabaya", "Bandung", "Medan", "Semarang",
	"Makassar", "Palembang", "Denpasar", "Yogyakarta", "Balikpapan",
	"Malang", "Batam", "Pekanbaru", "Pontianak", "Manado",
	"Padang", "Bengkulu", "Kupang", "Mataram", "Jayapura",
}

func (suite *WeatherAPITestSuite) TestWeatherEndpointLimiter() {
	var resp *http.Response
	var err error
	suite.Run("Failed - Rate limit exceeded per unique params", func() {
		ResetRateLimiterForIntegration()
		for i := 0; i <= 5; i++ {
			req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=Makassar", nil)
			resp, err = suite.httpServer.Client().Do(req)
			assert.NoError(suite.T(), err)

			if i < 2 {
				assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
			} else {
				assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode)
			}
		}
		defer resp.Body.Close()
	})

	suite.Run("Failed - Rate limit exceeded per global request", func() {
		ResetRateLimiterForIntegration()
		t := suite.T()
		for i := 0; i <= 14; i++ {

			city := ProvidedCities[i]
			req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location="+city, nil)
			resp, err = suite.httpServer.Client().Do(req)
			assert.NoError(t, err)

			var response model.Response
			err := json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			if i < 10 {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "Success", response.Message)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
				assert.Equal(t, "Too Many Requests (global limit)", response.Message)
			}
		}
		defer resp.Body.Close()
	})
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
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Error)
				assert.Contains(t, *response.Error, "Missing 'location' query parameter")
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
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Error)
				assert.Contains(t, *response.Error, "Missing 'location' query parameter")
			},
		},
		{
			name: "Failed - Invalid API key",
			setupMockTest: func() {
				// Clear any cached data for this test
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:Makassar")

				// Set an invalid API key for this test
				os.Setenv("OPENWEATHERMAP_API_KEY", "invalid_key")
				config.ReloadConfigForTest()
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=Makassar", nil)
				return req
			},
			wantStatus: http.StatusInternalServerError,
			validate: func(t *testing.T, resp *http.Response) {
				// Restore a valid API key after a test
				os.Setenv("OPENWEATHERMAP_API_KEY", "test_api_key")
				config.ReloadConfigForTest()
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Error)
				assert.Contains(t, *response.Error, "Failed to fetch weather data")
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
			wantStatus: http.StatusNotFound,
			validate: func(t *testing.T, resp *http.Response) {
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Error)
				assert.Equal(t, "city not found", *response.Error)
			},
		},
		{
			name: "Failed - City not found (short/ambiguous)",
			setupMockTest: func() {
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:ja")
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=ja", nil)
				return req
			},
			wantStatus: http.StatusNotFound,
			validate: func(t *testing.T, resp *http.Response) {
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Error)
				assert.Equal(t, "city not found", *response.Error)
			},
		},
		{
			name: "Success - Valid location (cached)",
			setupMockTest: func() {
				// Clear cache before setting up cached data
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:Makassar")

				// Setup Redis with cached data
				cachedWeather := &model.WeatherResponse{
					Location:    "Makassar",
					Temperature: 15.2,
					Description: "clear sky",
					Cached:      true,
				}

				data, _ := json.Marshal(cachedWeather)
				client.Set(ctx, "weather:Makassar", data, time.Minute)
				time.Sleep(50 * time.Millisecond)
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=Makassar", nil)
				return req
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp *http.Response) {
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, "Success", response.Message)
				assert.NotNil(t, response.Data)

				var weatherData model.WeatherResponse
				dataBytes, _ := json.Marshal(response.Data)
				err = json.Unmarshal(dataBytes, &weatherData)
				assert.NoError(t, err)

				assert.Equal(t, "Makassar", weatherData.Location)
				assert.True(t, weatherData.Cached)
			},
		},
		{
			name: "Success - Valid location (not-cached)",
			setupMockTest: func() {
				// Clear cache before running a not-cached test
				client := redis.GetClient()
				ctx := redis.GetContext()
				client.Del(ctx, "weather:Makassar")
			},
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=Makassar", nil)
				return req
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp *http.Response) {
				var response model.Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, "Success", response.Message)
				assert.NotNil(t, response.Data)

				var weatherData model.WeatherResponse
				dataBytes, _ := json.Marshal(response.Data)
				err = json.Unmarshal(dataBytes, &weatherData)
				assert.NoError(t, err)

				assert.Equal(t, "Makassar", weatherData.Location)
				assert.False(t, weatherData.Cached)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ResetRateLimiterForIntegration()
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

		if slices.Contains(ProvidedCities, q) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"name":    q,
				"main":    map[string]interface{}{"temp": 15.2},
				"weather": []map[string]interface{}{{"description": "clear sky"}},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"cod": "404", "message": "city not found"}`))
	}))
}
