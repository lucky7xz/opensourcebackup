package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

// validName rejects path traversal and unexpected characters.
var validName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// downloadAgent handles GET /downloads/agent/{version}/{platform}
// Serves pre-built agent binaries from dist/agent/{version}/.
func (h *Handler) downloadAgent(w http.ResponseWriter, r *http.Request) {
	version := r.PathValue("version")
	platform := r.PathValue("platform")

	if !validName.MatchString(version) || !validName.MatchString(platform) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	filename := "opensourcebackup-agent-" + platform
	if platform == "windows-amd64" {
		filename += ".exe"
	}

	path := filepath.Join("dist", "agent", version, filename)
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "binary not available for this platform/version", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
}

// serveInstallScript handles GET /scripts/install-agent.{sh,ps1}
// Serves install scripts from the scripts/ directory.
func (h *Handler) serveInstallScript(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Path)
	if !validName.MatchString(name) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	path := filepath.Join("scripts", name)
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "script not found", http.StatusNotFound)
		return
	}
	if filepath.Ext(name) == ".ps1" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	}
	http.ServeFile(w, r, path)
}

// listDownloads handles GET /downloads/agent — returns available binaries as JSON.
func (h *Handler) listDownloads(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Version  string `json:"version"`
		Platform string `json:"platform"`
		URL      string `json:"url"`
	}

	base := r.URL.Scheme + "://" + r.Host
	if base == "://" {
		base = "http://" + r.Host
	}

	var entries []entry
	versions, err := os.ReadDir("dist/agent")
	if err != nil {
		versions = nil // dist/agent doesn't exist yet — return empty list
	}
	for _, v := range versions {
		if !v.IsDir() {
			continue
		}
		files, ferr := os.ReadDir(filepath.Join("dist", "agent", v.Name()))
		if ferr != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			platform := name
			// strip prefix and .exe suffix to get platform
			platform = regexp.MustCompile(`^opensourcebackup-agent-`).ReplaceAllString(platform, "")
			platform = regexp.MustCompile(`\.exe$`).ReplaceAllString(platform, "")
			entries = append(entries, entry{
				Version:  v.Name(),
				Platform: platform,
				URL:      base + "/downloads/agent/" + v.Name() + "/" + platform,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries) //nolint:errcheck
}
