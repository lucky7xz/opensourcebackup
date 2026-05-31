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
		// Default: web/dist relative to working directory
		if _, err := os.Stat("web/dist/index.html"); err == nil {
			return "web/dist"
		}
		return "" // not built yet — skip
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return "" // directory doesn't have index.html
	}
	return dir
}

// spaHandler serves a Single Page Application (React).
// Known static assets are served directly; everything else returns index.html
// so React Router can handle client-side navigation.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		// If the file exists → serve it
		if _, err := os.Stat(path); err == nil && !strings.HasSuffix(path, "/") {
			fs.ServeHTTP(w, r)
			return
		}
		// Otherwise → index.html (React Router takes over)
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
