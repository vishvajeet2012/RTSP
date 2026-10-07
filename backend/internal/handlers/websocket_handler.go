package handlers

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	"rtspviewer/internal/config"
	"rtspviewer/internal/models"
	"rtspviewer/internal/services"
	"rtspviewer/internal/websocket"
	"rtspviewer/pkg/response"
)

type ticket struct {
	streamID string
	expires  time.Time
}
type WebSocketHandler struct {
	streams  *services.StreamManager
	hubs     *websocket.Manager
	cfg      config.Config
	log      *slog.Logger
	upgrader ws.Upgrader
	mu       sync.Mutex
	tickets  map[string]ticket
}

func NewWebSocketHandler(streams *services.StreamManager, hubs *websocket.Manager, cfg config.Config, log *slog.Logger) *WebSocketHandler {
	return &WebSocketHandler{streams: streams, hubs: hubs, cfg: cfg, log: log, tickets: make(map[string]ticket), upgrader: ws.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 8192, HandshakeTimeout: 10 * time.Second, CheckOrigin: func(r *http.Request) bool { return cfg.AllowsOrigin(r.Header.Get("Origin")) }}}
}

// Browser WebSockets cannot set Authorization. Exchange it for a one-use, 60-second ticket.
func (h *WebSocketHandler) Ticket(w http.ResponseWriter, r *http.Request) {
	if _, err := h.streams.Get(r.PathValue("id")); err != nil {
		streamError(w, err)
		return
	}
	value, err := services.NewID()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "unable to issue viewer ticket")
		return
	}
	h.mu.Lock()
	for key, t := range h.tickets {
		if time.Now().After(t.expires) {
			delete(h.tickets, key)
		}
	}
	if len(h.tickets) >= 1000 {
		h.mu.Unlock()
		response.Error(w, http.StatusTooManyRequests, "too many pending viewer connections")
		return
	}
	h.tickets[value] = ticket{streamID: r.PathValue("id"), expires: time.Now().Add(time.Minute)}
	h.mu.Unlock()
	response.JSON(w, http.StatusOK, map[string]any{"ticket": value, "expiresIn": 60})
}

func (h *WebSocketHandler) consume(value, id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	t, ok := h.tickets[value]
	delete(h.tickets, value)
	return ok && t.streamID == id && time.Now().Before(t.expires)
}

func (h *WebSocketHandler) Serve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if h.cfg.APIToken != "" && !h.consume(r.URL.Query().Get("ticket"), id) {
		response.Error(w, http.StatusUnauthorized, "viewer ticket is required or expired")
		return
	}
	stream, err := h.streams.Get(id)
	if err != nil {
		streamError(w, err)
		return
	}
	if stream.Status == models.Stopped || stream.Status == models.Error {
		response.Error(w, http.StatusConflict, "restart this stream before connecting")
		return
	}
	hub, ok := h.hubs.Get(id)
	if !ok {
		response.Error(w, http.StatusNotFound, "stream not found")
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := websocket.NewClient(conn)
	if err := hub.Subscribe(client); err != nil {
		client.Close()
		return
	}
	defer hub.Unsubscribe(client)
	h.log.Info("viewer connected", "stream", id)
	client.Run()
	h.log.Info("viewer disconnected", "stream", id)
}
