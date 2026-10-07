package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port, FFmpegPath, APIToken, Environment, FrontendDir string
	AllowedOrigins, AllowedRTSPHosts                     []string
	MaxStreams, MaxClients, ReconnectAttempts            int
	InputTimeout                                         time.Duration
}

func Load() (Config, error) {
	origins := env("RENDER_EXTERNAL_URL", "http://localhost:5173,http://127.0.0.1:5173")
	c := Config{
		Port: env("PORT", "8080"), FFmpegPath: env("FFMPEG_PATH", "ffmpeg"),
		APIToken: os.Getenv("API_TOKEN"), Environment: env("APP_ENV", "development"),
		FrontendDir:      os.Getenv("FRONTEND_DIR"),
		AllowedOrigins:   split(env("ALLOWED_ORIGINS", origins)),
		AllowedRTSPHosts: split(os.Getenv("ALLOWED_RTSP_HOSTS")),
	}
	for _, field := range []struct {
		key                string
		target             *int
		fallback, min, max int
	}{
		{"MAX_STREAMS", &c.MaxStreams, 6, 1, 100},
		{"MAX_CLIENTS_PER_STREAM", &c.MaxClients, 20, 1, 1000},
		{"STREAM_RECONNECT_MAX_ATTEMPTS", &c.ReconnectAttempts, 5, 0, 20},
	} {
		v, err := integer(field.key, field.fallback, field.min, field.max)
		if err != nil {
			return c, err
		}
		*field.target = v
	}
	seconds, err := integer("RTSP_TIMEOUT_SECONDS", 20, 3, 120)
	if err != nil {
		return c, err
	}
	c.InputTimeout = time.Duration(seconds) * time.Second
	p, err := strconv.Atoi(c.Port)
	if err != nil || p < 1 || p > 65535 {
		return c, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if len(c.AllowedOrigins) == 0 {
		return c, fmt.Errorf("ALLOWED_ORIGINS must contain at least one origin")
	}
	for _, origin := range c.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return c, fmt.Errorf("ALLOWED_ORIGINS must contain exact HTTP(S) origins without trailing slashes")
		}
	}
	if c.Environment == "production" && (len(c.APIToken) < 24 || len(c.AllowedRTSPHosts) == 0) {
		return c, fmt.Errorf("production requires API_TOKEN (at least 24 characters) and ALLOWED_RTSP_HOSTS")
	}
	if c.FrontendDir != "" {
		info, err := os.Stat(filepath.Join(c.FrontendDir, "index.html"))
		if err != nil || info.IsDir() {
			return c, fmt.Errorf("FRONTEND_DIR must contain index.html")
		}
	}
	return c, nil
}

func (c Config) AllowsOrigin(origin string) bool {
	if origin == "" {
		return true
	} // Native clients have no browser Origin; API auth still applies.
	for _, allowed := range c.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func split(value string) []string {
	var result []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}
func integer(key string, fallback, min, max int) (int, error) {
	v, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil || v < min || v > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return v, nil
}
