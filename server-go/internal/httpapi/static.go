package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"higoos/server-go/internal/platform"
)

func (a *API) staticAssets(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	if hasAPIPrefix(r.URL.Path) {
		platform.WriteError(w, r, http.StatusNotFound, "route_not_found", "api route not found")
		return
	}
	if a.staticDir == "" {
		http.NotFound(w, r)
		return
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "." {
		cleanPath = "index.html"
	}
	target := filepath.Join(a.staticDir, cleanPath)
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		target = filepath.Join(a.staticDir, "index.html")
		info, err = os.Stat(target)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
	}
	http.ServeFile(w, r, target)
}
