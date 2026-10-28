package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// spaHandler serves the built frontend from dir and falls back to index.html for
// client-side routes. Paths under /api and /healthz are never served from disk;
// unmatched API paths should 404 rather than return the SPA.
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") ||
			r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/healthz/") {
			http.NotFound(w, r)
			return
		}

		clean := path.Clean("/" + r.URL.Path)
		file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}

		// A missing path with a file extension is a missing asset, not a route.
		if strings.Contains(path.Base(clean), ".") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}
