package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yourusername/weather-api-redis/internal/config"
	"github.com/yourusername/weather-api-redis/internal/model"
)

// FetchWeatherFromProvider fetches weather data from OpenWeatherMap API
func FetchWeatherFromProvider(location string) (*model.WeatherResponse, error) {
	apiKey := config.GetOpenWeatherMapAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("OPENWEATHERMAP_API_KEY environment variable not set")
	}
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", location, apiKey)
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}
	// mapping data from API to our model
	var data struct {
		Name string `json:"name"`
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			TempMin   float64 `json:"temp_min"`
			TempMax   float64 `json:"temp_max"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
			SeaLevel  int     `json:"sea_level"`
			GrndLevel int     `json:"grnd_level"`
		} `json:"main"`
		Weather []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
	}
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
