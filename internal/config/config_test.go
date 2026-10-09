package config

import (
	"os"
	"testing"
)

func TestLoadDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
}

func TestLoadPortFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.Port)
	}
}

func TestMain(m *testing.M) {
	_ = os.Unsetenv("PORT")
	os.Exit(m.Run())
}
