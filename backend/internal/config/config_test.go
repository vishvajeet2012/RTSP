package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionRequiresProtectedConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_TOKEN", "")
	t.Setenv("ALLOWED_RTSP_HOSTS", "")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted anonymous unrestricted camera input")
	}
	t.Setenv("API_TOKEN", strings.Repeat("x", 32))
	t.Setenv("ALLOWED_RTSP_HOSTS", "camera.example")
	t.Setenv("ALLOWED_ORIGINS", "https://viewer.example")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowsOrigin("https://viewer.example") || cfg.AllowsOrigin("https://viewer.example.attacker.example") {
		t.Fatal("origin matching was not exact")
	}
}

func TestRenderOriginAndFrontendDirectory(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("RENDER_EXTERNAL_URL", "https://viewer.onrender.com")
	directory := t.TempDir()
	t.Setenv("FRONTEND_DIR", directory)
	if _, err := Load(); err == nil {
		t.Fatal("accepted a missing frontend build")
	}
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("<html></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || !cfg.AllowsOrigin("https://viewer.onrender.com") || cfg.AllowsOrigin("http://localhost:5173") {
		t.Fatalf("Render origin was not used as an exact default: %v", err)
	}
	t.Setenv("ALLOWED_ORIGINS", "https://custom.example")
	cfg, err = Load()
	if err != nil || !cfg.AllowsOrigin("https://custom.example") || cfg.AllowsOrigin("https://viewer.onrender.com") {
		t.Fatalf("explicit origins did not override the Render default: %v", err)
	}
}

func TestConfigurationRejectsMalformedLimitsAndOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("MAX_STREAMS", "0")
	if _, err := Load(); err == nil {
		t.Fatal("accepted invalid stream limit")
	}
	t.Setenv("MAX_STREAMS", "6")
	t.Setenv("ALLOWED_ORIGINS", "https://viewer.example/")
	if _, err := Load(); err == nil {
		t.Fatal("accepted an origin with a path")
	}
}
