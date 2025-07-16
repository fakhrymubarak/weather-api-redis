package config

import (
	"os"
	"testing"
)

func TestGetOpenWeatherMapAPIKey(t *testing.T) {
	// Test with environment variable set
	expectedKey := "test_api_key_123"
	os.Setenv("OPENWEATHERMAP_API_KEY", expectedKey)
	defer os.Unsetenv("OPENWEATHERMAP_API_KEY")

	result := GetOpenWeatherMapAPIKey()
	if result != expectedKey {
		t.Errorf("Expected API key %s, got %s", expectedKey, result)
	}

	// Test with environment variable not set
	os.Unsetenv("OPENWEATHERMAP_API_KEY")
	result = GetOpenWeatherMapAPIKey()
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestGetRedisAddr(t *testing.T) {
	// Test with environment variable set
	expectedAddr := "localhost:6379"
	os.Setenv("REDIS_ADDR", expectedAddr)
	defer os.Unsetenv("REDIS_ADDR")

	result := GetRedisAddr()
	if result != expectedAddr {
		t.Errorf("Expected Redis addr %s, got %s", expectedAddr, result)
	}

	// Test with environment variable not set (should return default)
	os.Unsetenv("REDIS_ADDR")
	result = GetRedisAddr()
	if result != "localhost:6379" {
		t.Errorf("Expected default Redis addr localhost:6379, got %s", result)
	}
}
