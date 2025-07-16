package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

var once sync.Once

func initConfig() {
	once.Do(func() {
		root, err := getProjectRoot()
		if err != nil {
			log.Fatalf("Error finding project root: %v", err)
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(root)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatalf("Error reading config file: %v", err)
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

func GetOpenWeatherMapAPIKey() string {
	initConfig()
	return viper.GetString("openweathermap.api_key")
}

func GetOpenWeatherMapAPIURL() string {
	initConfig()
	return viper.GetString("openweathermap.api_url")
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
