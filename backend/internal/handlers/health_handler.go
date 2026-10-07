package handlers

import (
	"net/http"
	"rtspviewer/internal/services"
	"rtspviewer/pkg/response"
)

func Health(runner services.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := runner.Check(); err != nil {
			response.JSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "ffmpeg": false})
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"status": "ok", "ffmpeg": true})
	}
}
