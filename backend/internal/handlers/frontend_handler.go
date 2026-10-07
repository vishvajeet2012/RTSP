package handlers

import (
	"io/fs"
	"net/http"
	"os"
	"strings"

	"rtspviewer/pkg/response"
)

// The optional built frontend shares the API origin in a single container deployment.
func Frontend(directory string) http.Handler {
	files := os.DirFS(directory)
	server := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		for _, part := range strings.Split(name, "/") {
			if strings.HasPrefix(part, ".") {
				response.Error(w, http.StatusNotFound, "endpoint not found")
				return
			}
		}
		info, err := fs.Stat(files, name)
		if err != nil || info.IsDir() {
			response.Error(w, http.StatusNotFound, "endpoint not found")
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if name == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		server.ServeHTTP(w, r)
	})
}
