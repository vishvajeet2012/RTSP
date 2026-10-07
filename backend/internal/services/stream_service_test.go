package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"rtspviewer/internal/config"
	"rtspviewer/internal/models"
	"rtspviewer/internal/websocket"
)

type controlledRunner struct {
	active    atomic.Int32
	calls     atomic.Int32
	maxActive atomic.Int32
	fail      bool
}

func (r *controlledRunner) Check() error { return nil }
func (r *controlledRunner) Run(ctx context.Context, _ string, onData func([]byte)) error {
	r.calls.Add(1)
	count := r.active.Add(1)
	defer r.active.Add(-1)
	for {
		old := r.maxActive.Load()
		if count <= old || r.maxActive.CompareAndSwap(old, count) {
			break
		}
	}
	if r.fail {
		return errors.New("camera unavailable")
	}
	onData([]byte{0x47, 0x00})
	<-ctx.Done()
	return ctx.Err()
}
func testManager(r Runner, attempts int) *StreamManager {
	cfg := config.Config{MaxStreams: 6, MaxClients: 4, ReconnectAttempts: attempts}
	return NewStreamManager(cfg, r, websocket.NewManager(4), slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func eventually(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not satisfied")
}

func TestLifecycleSharesOneProcess(t *testing.T) {
	r := &controlledRunner{}
	manager := testManager(r, 0)
	defer manager.Shutdown()
	stream, err := manager.Add("Office", "rtsp://admin:secret@localhost/live")
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { s, _ := manager.Get(stream.ID); return s.Status == models.Live })
	if _, err := manager.Add("Duplicate", "rtsp://admin:secret@LOCALHOST/live"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
	var group sync.WaitGroup
	for range 30 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := manager.Start(stream.ID); err != nil {
				t.Error(err)
			}
			_ = manager.List()
		}()
	}
	group.Wait()
	if r.calls.Load() != 1 {
		t.Fatalf("Start duplicated FFmpeg: %d", r.calls.Load())
	}
	if _, err := manager.Stop(stream.ID); err != nil {
		t.Fatal(err)
	}
	if r.active.Load() != 0 {
		t.Fatal("Stop did not wait for runner cleanup")
	}
	if _, err := manager.Restart(stream.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { return r.calls.Load() == 2 })
	if err := manager.Remove(stream.ID); err != nil {
		t.Fatal(err)
	}
	if len(manager.List()) != 0 || r.active.Load() != 0 {
		t.Fatal("Remove leaked session/process")
	}
	if r.maxActive.Load() != 1 {
		t.Fatal("multiple processes ran for the same stream")
	}
}

func TestConcurrentRestartAndShutdown(t *testing.T) {
	r := &controlledRunner{}
	manager := testManager(r, 0)
	stream, err := manager.Add("Office", "rtsp://localhost/live")
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for i := range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			if i%2 == 0 {
				_, _ = manager.Restart(stream.ID)
			} else {
				_, _ = manager.Stop(stream.ID)
			}
			_ = manager.List()
		}()
	}
	manager.Shutdown()
	group.Wait()
	if r.active.Load() != 0 || r.maxActive.Load() > 1 {
		t.Fatal("process lifecycle leaked/overlapped during shutdown")
	}
	if _, err := manager.Add("After shutdown", "rtsp://localhost/other"); !errors.Is(err, ErrShutdown) {
		t.Fatal("accepted a stream during shutdown")
	}
}

func TestExhaustedCameraIsError(t *testing.T) {
	r := &controlledRunner{fail: true}
	manager := testManager(r, 0)
	defer manager.Shutdown()
	stream, err := manager.Add("Offline", "rtsp://localhost/live")
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { s, _ := manager.Get(stream.ID); return s.Status == models.Error })
	if r.calls.Load() != 1 {
		t.Fatal("unbounded reconnect")
	}
}
