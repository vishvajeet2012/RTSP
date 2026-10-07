package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rtspviewer/internal/config"
	"rtspviewer/internal/services"
	"rtspviewer/internal/websocket"
)

type idleRunner struct{}

func (idleRunner) Check() error { return nil }
func (idleRunner) Run(ctx context.Context, _ string, _ func([]byte)) error {
	<-ctx.Done()
	return ctx.Err()
}

func testRouter(t *testing.T, token string) http.Handler {
	t.Helper()
	cfg := config.Config{MaxStreams: 6, MaxClients: 4, APIToken: token, AllowedOrigins: []string{"http://localhost:5173"}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	hubs := websocket.NewManager(4)
	manager := services.NewStreamManager(cfg, idleRunner{}, hubs, log)
	t.Cleanup(manager.Shutdown)
	return Router(cfg, idleRunner{}, manager, hubs, log)
}

func TestHealthAndMalformedRequests(t *testing.T) {
	router := testRouter(t, "")
	for _, test := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/api/health", "", 200}, {"GET", "/api/streams", "", 200},
		{"POST", "/api/streams", `{"rtspUrl":"https://example.com"}`, 400},
		{"POST", "/api/streams", `{`, 400},
		{"POST", "/api/streams", `{"rtspUrl":"rtsp://localhost/live","unknown":true}`, 400},
		{"POST", "/api/streams", `{"rtspUrl":"rtsp://localhost/live"} {}`, 400},
		{"GET", "/api/streams/missing", "", 404},
	} {
		r := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Errorf("%s %s: %d, %s", test.method, test.path, w.Code, w.Body.String())
		}
		if !json.Valid(w.Body.Bytes()) {
			t.Errorf("non-JSON response: %s", w.Body.String())
		}
	}
}

func TestAuthOriginsAndCredentialMasking(t *testing.T) {
	router := testRouter(t, "test-token")
	for _, test := range []struct {
		token, origin string
		want          int
	}{
		{"", "", 401}, {"test-token", "https://evil.example", 403}, {"test-token", "http://localhost:5173", 201},
	} {
		r := httptest.NewRequest("POST", "/api/streams", strings.NewReader(`{"name":"Office","rtspUrl":"rtsp://admin:supersecret@localhost/live?token=privatetoken"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+test.token)
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("got %d want %d: %s", w.Code, test.want, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "supersecret") || strings.Contains(w.Body.String(), "privatetoken") {
			t.Fatal("API leaked RTSP credentials")
		}
	}
}

func TestTicketsAreSingleUseAndScoped(t *testing.T) {
	h := &WebSocketHandler{tickets: make(map[string]ticket)}
	h.tickets["one"] = ticket{streamID: "camera-one", expires: future()}
	if !h.consume("one", "camera-one") || h.consume("one", "camera-one") {
		t.Fatal("ticket was reusable")
	}
	h.tickets["two"] = ticket{streamID: "camera-two", expires: future()}
	if h.consume("two", "camera-one") {
		t.Fatal("ticket granted another stream")
	}
}
