package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"rtspviewer/internal/config"
	"rtspviewer/pkg/response"
)

func CORS(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")
		if !cfg.AllowsOrigin(origin) {
			response.Error(w, http.StatusForbidden, "origin is not allowed")
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Auth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health" {
			provided := ""
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				response.Error(w, http.StatusUnauthorized, "backend access token is required or invalid")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
