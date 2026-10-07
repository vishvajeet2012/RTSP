package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	"rtspviewer/internal/config"
	"rtspviewer/internal/services"
	"rtspviewer/internal/websocket"
)

type countingRunner struct {
	runner        services.Runner
	calls, active atomic.Int32
}

func (r *countingRunner) Check() error { return r.runner.Check() }
func (r *countingRunner) Run(ctx context.Context, url string, output func([]byte)) error {
	r.calls.Add(1)
	r.active.Add(1)
	defer r.active.Add(-1)
	return r.runner.Run(ctx, url, output)
}

func TestRealRTSPFanoutAndCleanup(t *testing.T) {
	raw := os.Getenv("TEST_RTSP_URL")
	if raw == "" {
		t.Skip("set TEST_RTSP_URL to a running MediaMTX test publisher")
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{MaxStreams: 6, MaxClients: 4, InputTimeout: 10 * time.Second, AllowedOrigins: []string{"http://localhost:5173"}}
	runner := &countingRunner{runner: services.NewFFmpegService("ffmpeg", cfg.InputTimeout, log)}
	hubs := websocket.NewManager(4)
	manager := services.NewStreamManager(cfg, runner, hubs, log)
	server := httptest.NewServer(Router(cfg, runner, manager, hubs, log))
	defer server.Close()
	defer manager.Shutdown()
	stream, err := manager.Add("Integration camera", raw)
	if err != nil {
		t.Fatal(err)
	}
	connect := func() *ws.Conn {
		conn, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/streams/"+stream.ID, http.Header{"Origin": []string{"http://localhost:5173"}})
		if err != nil {
			t.Fatal(err)
		}
		return conn
	}
	first, second := connect(), connect()
	defer first.Close()
	defer second.Close()
	readVideo := func(conn *ws.Conn) {
		_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
		kind, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if kind != ws.BinaryMessage || len(data) < 188 || data[0] != 0x47 {
			t.Fatal("WebSocket did not carry MPEG transport stream packets")
		}
	}
	readVideo(first)
	readVideo(second)
	if _, err := manager.Start(stream.ID); err != nil {
		t.Fatal(err)
	}
	if runner.calls.Load() != 1 {
		t.Fatal("viewers created more than one FFmpeg process")
	}
	if _, err := manager.Stop(stream.ID); err != nil {
		t.Fatal(err)
	}
	if runner.active.Load() != 0 {
		t.Fatal("FFmpeg did not exit on Stop")
	}
	if _, err := manager.Restart(stream.ID); err != nil {
		t.Fatal(err)
	}
	third := connect()
	defer third.Close()
	readVideo(third)
	if runner.calls.Load() != 2 {
		t.Fatal("Restart did not start exactly one new process")
	}
	if err := manager.Remove(stream.ID); err != nil {
		t.Fatal(err)
	}
	if runner.active.Load() != 0 || len(manager.List()) != 0 {
		t.Fatal("Remove leaked a process or session")
	}
}
