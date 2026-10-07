package handlers

import (
	"log/slog"
	"net/http"
	"rtspviewer/internal/config"
	"rtspviewer/internal/middleware"
	"rtspviewer/internal/services"
	"rtspviewer/internal/websocket"
	"rtspviewer/pkg/response"
)

func Router(cfg config.Config, runner services.Runner, streams *services.StreamManager, hubs *websocket.Manager, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	h := NewStreamHandler(streams)
	sockets := NewWebSocketHandler(streams, hubs, cfg, log)
	mux.HandleFunc("GET /api/health", Health(runner))
	mux.HandleFunc("GET /api/streams", h.List)
	mux.HandleFunc("POST /api/streams", h.Create)
	mux.HandleFunc("GET /api/streams/{id}", h.Get)
	mux.HandleFunc("DELETE /api/streams/{id}", h.Delete)
	mux.HandleFunc("POST /api/streams/{id}/start", h.Start)
	mux.HandleFunc("POST /api/streams/{id}/stop", h.Stop)
	mux.HandleFunc("POST /api/streams/{id}/restart", h.Restart)
	mux.HandleFunc("POST /api/streams/{id}/ticket", sockets.Ticket)
	mux.HandleFunc("GET /ws/streams/{id}", sockets.Serve)
	if cfg.FrontendDir != "" {
		mux.Handle("/", Frontend(cfg.FrontendDir))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			response.Error(w, http.StatusNotFound, "endpoint not found")
		})
	}
	return middleware.Logging(log, middleware.CORS(cfg, middleware.Auth(cfg.APIToken, mux)))
}
