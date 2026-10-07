package handlers

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rtspviewer/internal/config"
	"rtspviewer/internal/services"
	"rtspviewer/internal/websocket"
)

func TestFrontendSharesOriginWithoutExposingFilesOrProtectedAPI(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"index.html": "<html>viewer</html>", "assets/main.js": "console.log('viewer')", ".env": "private"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{MaxStreams: 6, MaxClients: 4, FrontendDir: directory, APIToken: "test-token", AllowedOrigins: []string{"https://viewer.example"}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	hubs := websocket.NewManager(4)
	manager := services.NewStreamManager(cfg, idleRunner{}, hubs, log)
	t.Cleanup(manager.Shutdown)
	router := Router(cfg, idleRunner{}, manager, hubs, log)
	for _, test := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/", 200}, {"HEAD", "/", 200}, {"GET", "/assets/main.js", 200},
		{"GET", "/assets/", 404}, {"GET", "/.env", 404}, {"GET", "/missing", 404},
		{"POST", "/", 405}, {"GET", "/api/health", 200}, {"GET", "/api/streams", 401},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(test.method, test.path, nil))
		if w.Code != test.status {
			t.Errorf("%s %s: got %d want %d", test.method, test.path, w.Code, test.status)
		}
		if strings.Contains(w.Body.String(), "private") {
			t.Fatal("static server exposed a hidden file")
		}
		if test.path == "/" && test.method == "GET" && (!strings.Contains(w.Body.String(), "<html>viewer</html>") || w.Header().Get("Cache-Control") != "no-cache") {
			t.Fatal("frontend document or cache policy was incorrect")
		}
	}
}
