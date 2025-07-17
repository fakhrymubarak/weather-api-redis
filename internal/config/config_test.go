package config

import (
	"github.com/spf13/viper"
	"os"
	"testing"
	"time"
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

	// Test with an environment variable not set
	os.Unsetenv("OPENWEATHERMAP_API_KEY")
	result = GetOpenWeatherMapAPIKey()
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestGetRedisAddr(t *testing.T) {
	// Test with the environment variable set
	expectedAddr := "localhost:16379"
	result := GetRedisAddr()
	if result != expectedAddr {
		t.Errorf("Expected Redis addr %s, got %s", expectedAddr, result)
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
	want := "18080"
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

func TestGetRateLimiterCleanupTimeout(t *testing.T) {
	ReloadConfigForTest()
	want := time.Second // from config_test.yaml
	got := GetRateLimiterCleanupTimeout()
	if got != want {
		t.Errorf("Expected cleanup timeout %v, got %v", want, got)
	}

	// Test with config string error
	viper.Set("rate_limiter.cleanup_timeout", "9aslkdfjas")
	want = 3 * time.Minute
	got = GetRateLimiterCleanupTimeout()
	if got != want {
		t.Errorf("Expected cleanup timeout %v, got %v", want, got)
	}

	// Test without a config
	viper.Reset()
	want = 3 * time.Minute
	got = GetRateLimiterCleanupTimeout()
	if got != want {
		t.Errorf("Expected cleanup timeout %v, got %v", want, got)
	}

}

func TestGetGlobalRateLimiterConfig(t *testing.T) {
	ReloadConfigForTest()
	wantRate := 10.0 // from config_test.yaml
	wantBurst := 10
	rate, burst := GetGlobalRateLimiterConfig()
	if rate != wantRate {
		t.Errorf("Expected global rate %v, got %v", wantRate, rate)
	}
	if burst != wantBurst {
		t.Errorf("Expected global burst %v, got %v", wantBurst, burst)
	}
}

func TestGetParamRateLimiterConfig(t *testing.T) {
	ReloadConfigForTest()
	wantRate := 2.0 // from config_test.yaml
	wantBurst := 2
	rate, burst := GetParamRateLimiterConfig()
	if rate != wantRate {
		t.Errorf("Expected param rate %v, got %v", wantRate, rate)
	}
	if burst != wantBurst {
		t.Errorf("Expected param burst %v, got %v", wantBurst, burst)
	}
}

func TestGetRateLimiterCleanupTimeout_Default(t *testing.T) {
	// Temporarily move config_test.yaml out of the way to test default
	_ = os.Rename("../../config_test.yaml", "../../config_test.yaml.bak")
	defer os.Rename("../../config_test.yaml.bak", "../../config_test.yaml")
	ReloadConfigForTest()
	want := 3 * time.Minute
	got := GetRateLimiterCleanupTimeout()
	if got != want {
		t.Errorf("Expected default cleanup timeout %v, got %v", want, got)
	}
}

func TestGetGlobalRateLimiterConfig_Default(t *testing.T) {
	ReloadConfigForTest()
	wantRate := 10.0
	wantBurst := 10
	rate, burst := GetGlobalRateLimiterConfig()
	if rate != wantRate {
		t.Errorf("Expected default global rate %v, got %v", wantRate, rate)
	}
	if burst != wantBurst {
		t.Errorf("Expected default global burst %v, got %v", wantBurst, burst)
	}

	viper.Reset()
	rate, burst = GetGlobalRateLimiterConfig()
	if rate != wantRate {
		t.Errorf("Expected default global rate %v, got %v", wantRate, rate)
	}
	if burst != wantBurst {
		t.Errorf("Expected default global burst %v, got %v", wantBurst, burst)
	}
}

func TestGetParamRateLimiterConfig_Default(t *testing.T) {
	_ = os.Rename("../../config_test.yaml", "../../config_test.yaml.bak")
	defer os.Rename("../../config_test.yaml.bak", "../../config_test.yaml")
	ReloadConfigForTest()
	wantRate := 2.0
	wantBurst := 2
	rate, burst := GetParamRateLimiterConfig()
	if rate != wantRate {
		t.Errorf("Expected default param rate %v, got %v", wantRate, rate)
	}
	if burst != wantBurst {
		t.Errorf("Expected default param burst %v, got %v", wantBurst, burst)
	}

	viper.Reset()
	rate, burst = GetParamRateLimiterConfig()
	if rate != wantRate {
		t.Errorf("Expected default param rate %v, got %v", wantRate, rate)
	}
	if burst != wantBurst {
		t.Errorf("Expected default param burst %v, got %v", wantBurst, burst)
	}
}
