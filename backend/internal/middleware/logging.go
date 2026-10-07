package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"rtspviewer/pkg/response"
)

type recorder struct {
	http.ResponseWriter
	status int
}

func (w *recorder) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *recorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
func (w *recorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *recorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("HTTP server does not support WebSockets")
	}
	conn, rw, err := h.Hijack()
	if err == nil {
		w.status = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

func Logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &recorder{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("request panic", "path", r.URL.Path) // Do not log arbitrary panic values that may contain input secrets.
				if wrapped.status == 0 {
					response.Error(wrapped, http.StatusInternalServerError, "unexpected backend error")
				}
			}
			status := wrapped.status
			if status == 0 {
				status = http.StatusOK
			}
			log.Info("request", "method", r.Method, "path", r.URL.Path, "status", status, "duration", time.Since(started))
		}()
		next.ServeHTTP(wrapped, r)
	})
}
