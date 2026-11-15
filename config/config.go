package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort            = 8080
	defaultMaxUploadSizeMB = 10
	defaultOllamaURL       = "http://localhost:11434"
	defaultOllamaModel     = "qwen3:4b"
	defaultOllamaTimeout   = 60
)

type Config struct {
	OpenAIAPIKey    string
	OpenAIModel     string
	Port            int
	MaxUploadSizeMB int64
	OllamaURL       string
	OllamaModel     string
	OllamaTimeout   time.Duration
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
	ollamaURL := strings.TrimSpace(stringEnv("OLLAMA_URL", defaultOllamaURL))
	parsedURL, err := url.ParseRequestURI(ollamaURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return Config{}, fmt.Errorf("OLLAMA_URL must be a valid http or https URL")
	}
	ollamaModel := strings.TrimSpace(stringEnv("OLLAMA_MODEL", defaultOllamaModel))
	if ollamaModel == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL must not be empty")
	}
	ollamaTimeoutSeconds, err := intEnv("OLLAMA_TIMEOUT_SECONDS", defaultOllamaTimeout)
	if err != nil {
		return Config{}, err
	}
	if ollamaTimeoutSeconds < 1 || ollamaTimeoutSeconds > 300 {
		return Config{}, fmt.Errorf("OLLAMA_TIMEOUT_SECONDS must be between 1 and 300")
	}

	return Config{
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:     os.Getenv("OPENAI_MODEL"),
		Port:            port,
		MaxUploadSizeMB: maxUploadSizeMB,
		OllamaURL:       ollamaURL,
		OllamaModel:     ollamaModel,
		OllamaTimeout:   time.Duration(ollamaTimeoutSeconds) * time.Second,
	}, nil
}

func stringEnv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
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
