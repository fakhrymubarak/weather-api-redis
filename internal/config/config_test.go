package config

import (
	"os"
	"testing"
)

func TestGetOpenWeatherMapAPIKey(t *testing.T) {
	// Test with the environment variable set
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
	// Test with the environment variable set
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

func TestGetOpenWeatherApiUrl(t *testing.T) {
	want := "https://api.openweathermap.org/data/2.5/weather"
	got := GetOpenWeatherApiUrl()
	if got != want {
		t.Errorf("Expected API URL %s, got %s", want, got)
	}
}

func TestGetServerPort(t *testing.T) {
	want := "8080"
	got := GetServerPort()
	if got != want {
		t.Errorf("Expected server port %s, got %s", want, got)
	}
}

func TestGetCacheExpiration(t *testing.T) {
	want := "10m"
	got := GetCacheExpiration()
	if got != want {
		t.Errorf("Expected cache expiration %s, got %s", want, got)
	}
}

func TestGetServerTimeout(t *testing.T) {
	want := "15s"
	got := GetServerTimeout("read_header_timeout")
	if got != want {
		t.Errorf("Expected read_header_timeout %s, got %s", want, got)
	}
}

func TestGetTestRedisMockPort(t *testing.T) {
	want := ":16379"
	got := GetTestRedisMockPort()
	if got != want {
		t.Errorf("Expected test redis mock port %s, got %s", want, got)
	}
}

func TestGetTestServerPort(t *testing.T) {
	want := ":8080"
	got := GetTestServerPort()
	if got != want {
		t.Errorf("Expected test server port %s, got %s", want, got)
	}
}

func TestReloadConfigForTest(t *testing.T) {
	// Should not panic or error
	ReloadConfigForTest()
}

func TestInitConfig_MissingConfigFile(t *testing.T) {
	// Temporarily move config.yaml out of the way
	_ = os.Rename("../../config.yaml", "../../config.yaml.bak")
	defer os.Rename("../../config.yaml.bak", "../../config.yaml")
	defer func() { recover() }() // Expect panic
	initConfig()
}

func TestGetProjectRoot_MissingGoMod(t *testing.T) {
	_ = os.Rename("../../go.mod", "../../go.mod.bak")
	defer os.Rename("../../go.mod.bak", "../../go.mod")
	_, err := getProjectRoot()
	if err == nil {
		t.Error("Expected error for missing go.mod, got nil")
	}
}
