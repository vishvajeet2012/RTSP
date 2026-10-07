package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"rtspviewer/internal/config"
	"rtspviewer/internal/models"
	"rtspviewer/internal/websocket"
)

var (
	ErrNotFound  = errors.New("stream not found")
	ErrDuplicate = errors.New("this RTSP URL has already been added")
	ErrLimit     = errors.New("stream limit reached; remove a stream before adding another")
	ErrShutdown  = errors.New("backend is shutting down")
)

type StreamSession struct {
	opMu    sync.Mutex // Serialize start/stop/restart/remove, including process reaping.
	mu      sync.RWMutex
	info    models.Stream
	rawURL  string
	hub     *websocket.Hub
	cancel  context.CancelFunc
	done    chan struct{}
	removed bool
}

func (s *StreamSession) snapshot() models.Stream {
	s.mu.RLock()
	info := s.info
	s.mu.RUnlock()
	info.Viewers = s.hub.Viewers()
	return info
}
func (s *StreamSession) status(status models.Status, lastError string, attempt int) {
	s.mu.Lock()
	s.info.Status = status
	s.info.LastError = lastError
	s.info.ReconnectAttempt = attempt
	s.mu.Unlock()
}

type StreamManager struct {
	mu       sync.RWMutex
	sessions map[string]*StreamSession
	byURL    map[string]string
	closing  bool
	ctx      context.Context
	cancel   context.CancelFunc
	runner   Runner
	hubs     *websocket.Manager
	cfg      config.Config
	log      *slog.Logger
}

func NewStreamManager(cfg config.Config, runner Runner, hubs *websocket.Manager, log *slog.Logger) *StreamManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamManager{sessions: make(map[string]*StreamSession), byURL: make(map[string]string), ctx: ctx, cancel: cancel, runner: runner, hubs: hubs, cfg: cfg, log: log}
}

func (m *StreamManager) Add(name, rawURL string) (models.Stream, error) {
	u, err := models.ValidateRTSPURL(rawURL)
	if err != nil {
		return models.Stream{}, err
	}
	name, err = models.ValidateName(name)
	if err != nil {
		return models.Stream{}, err
	}
	if len(m.cfg.AllowedRTSPHosts) > 0 {
		allowed := false
		for _, host := range m.cfg.AllowedRTSPHosts {
			if strings.EqualFold(host, u.Hostname()) {
				allowed = true
				break
			}
		}
		if !allowed {
			return models.Stream{}, fmt.Errorf("camera hostname is not in ALLOWED_RTSP_HOSTS")
		}
	}
	if err := m.runner.Check(); err != nil {
		return models.Stream{}, err
	}
	rawURL = u.String()
	id, err := NewID()
	if err != nil {
		return models.Stream{}, fmt.Errorf("generate stream ID: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing {
		return models.Stream{}, ErrShutdown
	}
	if _, exists := m.byURL[rawURL]; exists {
		return models.Stream{}, ErrDuplicate
	}
	if len(m.sessions) >= m.cfg.MaxStreams {
		return models.Stream{}, ErrLimit
	}
	if name == "" {
		name = fmt.Sprintf("Camera %d", len(m.sessions)+1)
	}
	s := &StreamSession{rawURL: rawURL, hub: m.hubs.Create(id), info: models.Stream{ID: id, Name: name, RTSPURL: models.SanitizeRTSPURL(rawURL), Hostname: u.Hostname(), Status: models.Idle, CreatedAt: time.Now().UTC()}}
	m.sessions[id] = s
	m.byURL[rawURL] = id
	m.startLocked(s)
	m.log.Info("stream created", "id", id, "name", name, "url", s.info.RTSPURL)
	return s.snapshot(), nil
}

func (m *StreamManager) session(id string) (*StreamSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.sessions[id]
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}
func (m *StreamManager) Get(id string) (models.Stream, error) {
	s, err := m.session(id)
	if err != nil {
		return models.Stream{}, err
	}
	return s.snapshot(), nil
}
func (m *StreamManager) List() []models.Stream {
	m.mu.RLock()
	sessions := make([]*StreamSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()
	result := make([]models.Stream, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, s.snapshot())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result
}

func (m *StreamManager) Start(id string) (models.Stream, error) {
	s, err := m.session(id)
	if err != nil {
		return models.Stream{}, err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.removed {
		return models.Stream{}, ErrNotFound
	}
	if m.ctx.Err() != nil {
		return models.Stream{}, ErrShutdown
	}
	if err := m.runner.Check(); err != nil {
		return models.Stream{}, err
	}
	if s.done != nil {
		select {
		case <-s.done:
			s.done = nil
		default:
			return s.snapshot(), nil
		}
	}
	m.startLocked(s)
	return s.snapshot(), nil
}

func (m *StreamManager) Stop(id string) (models.Stream, error) {
	s, err := m.session(id)
	if err != nil {
		return models.Stream{}, err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.removed {
		return models.Stream{}, ErrNotFound
	}
	m.stopLocked(s)
	return s.snapshot(), nil
}

func (m *StreamManager) Restart(id string) (models.Stream, error) {
	s, err := m.session(id)
	if err != nil {
		return models.Stream{}, err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.removed {
		return models.Stream{}, ErrNotFound
	}
	if m.ctx.Err() != nil {
		return models.Stream{}, ErrShutdown
	}
	if err := m.runner.Check(); err != nil {
		return models.Stream{}, err
	}
	m.stopLocked(s)
	m.startLocked(s)
	return s.snapshot(), nil
}

func (m *StreamManager) Remove(id string) error {
	s, err := m.session(id)
	if err != nil {
		return err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.removed {
		return ErrNotFound
	}
	// Keep the URL reserved until its old process has been fully reaped.
	m.stopLocked(s)
	s.removed = true
	m.mu.Lock()
	delete(m.sessions, id)
	delete(m.byURL, s.rawURL)
	m.mu.Unlock()
	m.hubs.Remove(id)
	m.log.Info("stream removed", "id", id)
	return nil
}

func (m *StreamManager) startLocked(s *StreamSession) {
	s.hub.Start()
	ctx, cancel := context.WithCancel(m.ctx)
	s.cancel = cancel
	done := make(chan struct{})
	s.done = done
	s.mu.Lock()
	now := time.Now().UTC()
	s.info.StartedAt = &now
	s.mu.Unlock()
	s.status(models.Connecting, "", 0)
	go func() { defer close(done); m.supervise(ctx, s) }()
}

func (m *StreamManager) stopLocked(s *StreamSession) {
	// Block late viewer subscriptions before cancelling and reaping FFmpeg.
	s.hub.Stop()
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		<-s.done
		s.done = nil
	}
	s.status(models.Stopped, "", 0)
}

func (m *StreamManager) supervise(ctx context.Context, s *StreamSession) {
	failures := 0
	for {
		started := time.Now()
		producedVideo := false
		m.log.Info("ffmpeg starting", "stream", s.info.ID, "attempt", failures)
		err := m.runner.Run(ctx, s.rawURL, func(data []byte) {
			if !producedVideo {
				producedVideo = true
				s.status(models.Live, "", 0)
			}
			s.hub.Broadcast(data)
		})
		s.hub.DisconnectAll()
		if ctx.Err() != nil {
			return
		}
		if producedVideo && time.Since(started) >= 30*time.Second {
			failures = 0
		}
		if err == nil {
			err = fmt.Errorf("RTSP stream ended")
		}
		m.log.Warn("ffmpeg disconnected", "stream", s.info.ID, "error", err.Error())
		if failures >= m.cfg.ReconnectAttempts || errors.Is(err, ErrFFmpegUnavailable) || errors.Is(err, ErrRTSPTLSUnavailable) {
			s.hub.Stop()
			s.status(models.Error, err.Error(), failures)
			return
		}
		failures++
		s.status(models.Reconnecting, err.Error(), failures)
		delay := min(time.Duration(1<<min(failures-1, 4))*time.Second, 15*time.Second)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (m *StreamManager) Shutdown() {
	m.mu.Lock()
	m.closing = true
	m.cancel()
	sessions := make([]*StreamSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()
	for _, s := range sessions {
		s.opMu.Lock()
		m.stopLocked(s)
		s.removed = true
		m.hubs.Remove(s.info.ID)
		s.opMu.Unlock()
	}
	m.mu.Lock()
	clear(m.sessions)
	clear(m.byURL)
	m.mu.Unlock()
}

func NewID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(id[:]), nil
}
