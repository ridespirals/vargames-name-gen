package config

import (
	"os"
	"testing"
)

func TestLoad_MaxConcurrent(t *testing.T) {
	t.Setenv(EnvClientID, "test-id")
	t.Setenv(EnvClientSecret, "test-secret")
	t.Setenv(EnvMaxConcurrent, "8")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MaxConcurrent != 8 {
		t.Fatalf("MaxConcurrent = %d, want 8", cfg.MaxConcurrent)
	}
}

func TestLoad_MaxConcurrentDefault(t *testing.T) {
	t.Setenv(EnvClientID, "test-id")
	t.Setenv(EnvClientSecret, "test-secret")
	os.Unsetenv(EnvMaxConcurrent)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MaxConcurrent != DefaultMaxConcurrent {
		t.Fatalf("MaxConcurrent = %d, want default %d", cfg.MaxConcurrent, DefaultMaxConcurrent)
	}
}

func TestLoad_MaxConcurrentInvalidUsesDefault(t *testing.T) {
	t.Setenv(EnvClientID, "test-id")
	t.Setenv(EnvClientSecret, "test-secret")
	t.Setenv(EnvMaxConcurrent, "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MaxConcurrent != DefaultMaxConcurrent {
		t.Fatalf("MaxConcurrent = %d, want default %d", cfg.MaxConcurrent, DefaultMaxConcurrent)
	}
}
