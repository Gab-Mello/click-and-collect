package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env             string
	Port            int
	LogLevel        string
	DatabaseURL     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid PORT: %w", err)
	}

	return Config{
		Env:             getEnv("APP_ENV", "local"),
		Port:            port,
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
