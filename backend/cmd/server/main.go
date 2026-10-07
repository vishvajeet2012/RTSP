package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"rtspviewer/internal/config"
	"rtspviewer/internal/handlers"
	"rtspviewer/internal/services"
	"rtspviewer/internal/websocket"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		return err
	}
	runner := services.NewFFmpegService(cfg.FFmpegPath, cfg.InputTimeout, log)
	if err := runner.Check(); err != nil {
		log.Warn("FFmpeg unavailable; stream creation disabled", "error", err)
	}
	hubs := websocket.NewManager(cfg.MaxClients)
	streams := services.NewStreamManager(cfg, runner, hubs, log)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: handlers.Router(cfg, runner, streams, hubs, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("server listening", "port", cfg.Port, "environment", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()
	var serveErr error
	select {
	case <-ctx.Done():
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server failed", "error", err)
			serveErr = err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Shutdown closes the listener first. Active sockets are hijacked, so the manager closes them explicitly.
	httpDone := make(chan struct{})
	go func() {
		defer close(httpDone)
		if err := server.Shutdown(shutdown); err != nil {
			log.Warn("HTTP shutdown timeout")
			_ = server.Close()
		}
	}()
	streams.Shutdown()
	<-httpDone
	log.Info("shutdown complete")
	return serveErr
}
