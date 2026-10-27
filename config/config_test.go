package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("MAX_UPLOAD_SIZE_MB", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")

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
