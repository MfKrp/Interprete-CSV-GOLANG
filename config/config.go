package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultPort            = 8080
	defaultMaxUploadSizeMB = 10
)

type Config struct {
	OpenAIAPIKey    string
	OpenAIModel     string
	Port            int
	MaxUploadSizeMB int64
}

func Load() (Config, error) {
	port, err := intEnv("PORT", defaultPort)
	if err != nil {
		return Config{}, err
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}

	maxUploadSizeMB, err := int64Env("MAX_UPLOAD_SIZE_MB", defaultMaxUploadSizeMB)
	if err != nil {
		return Config{}, err
	}
	if maxUploadSizeMB < 1 {
		return Config{}, fmt.Errorf("MAX_UPLOAD_SIZE_MB must be greater than 0")
	}

	return Config{
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:     os.Getenv("OPENAI_MODEL"),
		Port:            port,
		MaxUploadSizeMB: maxUploadSizeMB,
	}, nil
}

func intEnv(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func int64Env(name string, fallback int64) (int64, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}
