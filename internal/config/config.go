package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var once sync.Once
var logger *zap.SugaredLogger
var loggerOnce sync.Once

func initConfig() {
	once.Do(func() {
		root, err := getProjectRoot()
		if err != nil {
			GetLogger().Fatalw("Erciror finding project root", "error", err)
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(root)
		err = viper.ReadInConfig()
		if err != nil {
			GetLogger().Fatalw("Error reading config file", "error", err)
		}
	})
}

func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

func GetOpenWeatherApiUrl() string {
	initConfig()
	return viper.GetString("openweathermap.api_url")
}

func GetOpenWeatherMapAPIKey() string {
	_ = godotenv.Load()
	return os.Getenv("OPENWEATHERMAP_API_KEY")
}

func GetRedisAddr() string {
	initConfig()
	return viper.GetString("redis.addr")
}

func GetServerPort() string {
	initConfig()
	return viper.GetString("server.port")
}

func GetCacheExpiration() string {
	initConfig()
	return viper.GetString("cache.expiration")
}

func GetServerTimeout(key string) string {
	initConfig()
	return viper.GetString("server." + key)
}

func GetTestRedisMockPort() string {
	initConfig()
	return viper.GetString("test.redis_mock_port")
}

func GetTestServerPort() string {
	initConfig()
	return viper.GetString("test.server_port")
}

// ReloadConfigForTest resets the config singleton and reloads Viper config. Use only in tests.
func ReloadConfigForTest() {
	once = sync.Once{}
	initConfig()
}

func GetLogger() *zap.SugaredLogger {
	loggerOnce.Do(func() {
		l, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		logger = l.Sugar()
	})
	return logger
}

// GetRateLimiterCleanupTimeout returns the rate limiter cleanup timeout as a time.Duration.
// Defaults to 3m if not set or invalid.
func GetRateLimiterCleanupTimeout() time.Duration {
	initConfig()
	durStr := viper.GetString("rate_limiter.cleanup_timeout")
	if durStr == "" {
		durStr = "3m"
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return 3 * time.Minute
	}
	return dur
}

// GetGlobalRateLimiterConfig returns the rate and burst for the global rate limiter from config.
func GetGlobalRateLimiterConfig() (rate float64, burst int) {
	initConfig()
	rate = viper.GetFloat64("rate_limiter.global.rate")
	if rate == 0 {
		rate = 10
	}
	burst = viper.GetInt("rate_limiter.global.burst")
	if burst == 0 {
		burst = 10
	}
	return
}

// GetParamRateLimiterConfig returns the rate and burst for the param rate limiter from config.
func GetParamRateLimiterConfig() (rate float64, burst int) {
	initConfig()
	rate = viper.GetFloat64("rate_limiter.param.rate")
	if rate == 0 {
		rate = 2
	}
	burst = viper.GetInt("rate_limiter.param.burst")
	if burst == 0 {
		burst = 2
	}
	return
}
