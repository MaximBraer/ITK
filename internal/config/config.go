package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"ITK/pkg/postgres"

	"github.com/joho/godotenv"
)

type Config struct {
	Env        string
	HTTPServer HTTPServer
	DB         postgres.DBConfig
	Retry      RetryConfig
}

type RetryConfig struct {
	MaxAttempts  int
	BaseDelayMS  int
}

type HTTPServer struct {
	Address     string
	Timeout     time.Duration
	IdleTimeout time.Duration
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/local.env"
	}

	if err := godotenv.Load(configPath); err != nil {
		log.Printf("Warning: failed to load config file %s: %v. Using environment variables.", configPath, err)
	}

	timeout, err := time.ParseDuration(getEnv("HTTP_TIMEOUT", "5s"))
	if err != nil {
		timeout = 5 * time.Second
	}

	idleTimeout, err := time.ParseDuration(getEnv("HTTP_IDLE_TIMEOUT", "60s"))
	if err != nil {
		idleTimeout = 60 * time.Second
	}

	return &Config{
		Env: getEnv("ENV", "local"),
		HTTPServer: HTTPServer{
			Address:     getEnv("HTTP_ADDRESS", "0.0.0.0:8080"),
			Timeout:     timeout,
			IdleTimeout: idleTimeout,
		},
		DB: postgres.DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "wallet"),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 20),
			ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 3600),
		},
		Retry: RetryConfig{
			MaxAttempts: getEnvAsInt("RETRY_MAX_ATTEMPTS", 10),
			BaseDelayMS: getEnvAsInt("RETRY_BASE_DELAY_MS", 10),
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
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
