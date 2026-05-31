package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// webUIDir returns the directory from which to serve the React app.
// Set WEB_UI_DIR env var (default: web/dist). Empty string = disabled.
func (h *Handler) webUIDir() string {
	dir := os.Getenv("WEB_UI_DIR")
	if dir == "" {
		if _, err := os.Stat("web/dist/index.html"); err == nil {
			return "web/dist"
		}
		return ""
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return ""
	}
	return dir
}

// spaHandler serves a Single Page Application (React).
// Static assets (JS, CSS, images) are served directly.
// All other paths return index.html so React Router handles navigation.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip leading slash and use clean relative path to avoid
		// filepath.Join dropping the dir prefix on absolute paths.
		rel := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		full := filepath.Join(dir, rel)

		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			// File exists → serve it directly (JS, CSS, assets, favicon)
			fs.ServeHTTP(w, r)
			return
		}
		// Not a file → serve index.html for React Router
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
