package integrationtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/yourusername/weather-api-redis/internal/handler"
	"github.com/yourusername/weather-api-redis/internal/model"
	"github.com/yourusername/weather-api-redis/internal/redis"
)

type WeatherAPITestSuite struct {
	suite.Suite
	httpServer     *httptest.Server
	weatherHandler *handler.WeatherHandler
}

func (suite *WeatherAPITestSuite) SetupSuite() {
	// Set up test environment variables
	os.Setenv("OPENWEATHERMAP_API_KEY", "test_api_key")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	// Create weather handler
	suite.weatherHandler = handler.NewWeatherHandler()

	// Create test server with proper handler
	mux := http.NewServeMux()
	mux.HandleFunc("/weather", suite.weatherHandler.HandleWeather)
	suite.httpServer = httptest.NewServer(mux)
}

func (suite *WeatherAPITestSuite) TearDownSuite() {
	if suite.httpServer != nil {
		suite.httpServer.Close()
	}
}

func TestWeatherAPITestSuite(t *testing.T) {
	suite.Run(t, new(WeatherAPITestSuite))
}

func (suite *WeatherAPITestSuite) TestWeatherEndpoint() {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		wantStatus   int
		validate     func(t *testing.T, resp *http.Response)
	}{
		{
			name: "Failed - Missing location parameter",
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
	}

	for _, tt := range tests {
		suite.Suite.Run(tt.name, func() {
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

func (suite *WeatherAPITestSuite) TestWeatherServiceIntegration() {
	ctx := context.Background()

	// Test service directly
	_, err := suite.weatherHandler.WeatherService.GetWeather(ctx, "London")
	assert.Error(suite.T(), err, "Expected error with invalid API key")
}

func (suite *WeatherAPITestSuite) TestRedisIntegration() {
	// Test Redis connection
	client := redis.GetClient()
	ctx := redis.GetContext()

	// Test basic Redis operations
	testKey := "test:weather:integration"
	testData := &model.WeatherResponse{
		Location:    "Test City",
		Temperature: 25.0,
		Description: "sunny",
		Cached:      false,
	}

	// Marshal test data
	data, err := json.Marshal(testData)
	if err != nil {
		suite.T().Fatalf("Failed to marshal test data: %v", err)
	}

	// Set test data
	err = client.Set(ctx, testKey, data, time.Minute).Err()
	if err != nil {
		suite.T().Logf("Redis not available, skipping Redis tests: %v", err)
		return
	}

	// Get test data
	val, err := client.Get(ctx, testKey).Result()
	assert.NoError(suite.T(), err)

	// Unmarshal and verify
	var retrieved model.WeatherResponse
	err = json.Unmarshal([]byte(val), &retrieved)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), testData.Location, retrieved.Location)

	// Clean up
	client.Del(ctx, testKey)
}

func (suite *WeatherAPITestSuite) TestModelStructures() {
	// Test WeatherResponse marshaling/unmarshaling
	weather := &model.WeatherResponse{
		Location:    "Test City",
		Temperature: 20.5,
		Description: "cloudy",
		Cached:      true,
	}

	data, err := json.Marshal(weather)
	assert.NoError(suite.T(), err)

	var unmarshaled model.WeatherResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), weather.Location, unmarshaled.Location)

	// Test OpenWeatherMapResponse
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
			Temp: 15.2,
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

	data, err = json.Marshal(owmResponse)
	assert.NoError(suite.T(), err)

	var unmarshaledOWM model.OpenWeatherMapResponse
	err = json.Unmarshal(data, &unmarshaledOWM)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), owmResponse.Name, unmarshaledOWM.Name)
}

func (suite *WeatherAPITestSuite) TestConcurrentRequests() {
	// Test concurrent access to the API
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req, _ := http.NewRequest(http.MethodGet, suite.httpServer.URL+"/weather?location=ConcurrentCity", nil)
			resp, err := suite.httpServer.Client().Do(req)
			if err != nil {
				suite.T().Logf("Concurrent request %d failed: %v", id, err)
			} else {
				resp.Body.Close()
				suite.T().Logf("Concurrent request %d completed with status: %d", id, resp.StatusCode)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}
