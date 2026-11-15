package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("MAX_UPLOAD_SIZE_MB", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OLLAMA_URL", "")
	t.Setenv("OLLAMA_MODEL", "")
	t.Setenv("OLLAMA_TIMEOUT_SECONDS", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Port != defaultPort {
		t.Errorf("Port = %d, want %d", got.Port, defaultPort)
	}
	if got.MaxUploadSizeMB != defaultMaxUploadSizeMB {
		t.Errorf("MaxUploadSizeMB = %d, want %d", got.MaxUploadSizeMB, defaultMaxUploadSizeMB)
	}
	if got.OllamaURL != defaultOllamaURL || got.OllamaModel != defaultOllamaModel {
		t.Errorf("Ollama defaults = %q, %q", got.OllamaURL, got.OllamaModel)
	}
	if got.OllamaTimeout.Seconds() != defaultOllamaTimeout {
		t.Errorf("OllamaTimeout = %v, want %d seconds", got.OllamaTimeout, defaultOllamaTimeout)
	}
}

func TestLoadRejectsInvalidOllamaSettings(t *testing.T) {
	tests := []struct{ name, key, value string }{
		{"invalid URL", "OLLAMA_URL", "not-a-url"},
		{"empty model", "OLLAMA_MODEL", " "},
		{"non-numeric timeout", "OLLAMA_TIMEOUT_SECONDS", "soon"},
		{"out-of-range timeout", "OLLAMA_TIMEOUT_SECONDS", "301"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.key, test.value)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}
		})
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		maxUploadMB string
	}{
		{name: "non-numeric port", port: "abc", maxUploadMB: "10"},
		{name: "out-of-range port", port: "70000", maxUploadMB: "10"},
		{name: "non-numeric upload size", port: "8080", maxUploadMB: "abc"},
		{name: "non-positive upload size", port: "8080", maxUploadMB: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)
			t.Setenv("MAX_UPLOAD_SIZE_MB", tt.maxUploadMB)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}
		})
	}
}
