package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	Environment string
	APIKeys     []string
	RateLimit   int

	Dremio   DremioConfig
	BigQuery BigQueryConfig
	Redis    RedisConfig
}

type DremioConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Token    string
}

type BigQueryConfig struct {
	ProjectID   string
	DatasetID   string
	Credentials string // Path to service account JSON
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENV", "development"),
		APIKeys:     strings.Split(getEnv("API_KEYS", "demo-key-123"), ","),
		RateLimit:   getEnvAsInt("RATE_LIMIT", 100),

		Dremio: DremioConfig{
			Host:     getEnv("DREMIO_HOST", ""),
			Port:     getEnvAsInt("DREMIO_PORT", 31010),
			Username: getEnv("DREMIO_USERNAME", ""),
			Password: getEnv("DREMIO_PASSWORD", ""),
			Token:    getEnv("DREMIO_TOKEN", ""),
		},

		BigQuery: BigQueryConfig{
			ProjectID:   getEnv("BIGQUERY_PROJECT_ID", ""),
			DatasetID:   getEnv("BIGQUERY_DATASET_ID", ""),
			Credentials: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		},

		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return defaultValue
}