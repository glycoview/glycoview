package ui

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type assetsHandler struct {
	assets   http.Handler
	distDir  string
	fallback []byte
}

func newAssetsHandler() http.Handler {
	distDir := filepath.Join("frontend", "dist")
	index, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		index = []byte(`<!doctype html><html><head><meta charset="utf-8"><title>GlycoView</title></head><body><div id="root"></div></body></html>`)
	}
	return assetsHandler{
		assets:   http.FileServer(http.Dir(distDir)),
		distDir:  distDir,
		fallback: index,
	}
}

func (h assetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if shouldServeAsset(r.URL.Path) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		h.assets.ServeHTTP(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	index, err := os.ReadFile(filepath.Join(h.distDir, "index.html"))
	if err != nil {
		index = h.fallback
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}

func shouldServeAsset(requestPath string) bool {
	if requestPath == "/" || requestPath == "" {
		return false
	}
	if strings.HasPrefix(requestPath, "/assets/") {
		return true
	}
	base := path.Base(requestPath)
	return strings.Contains(base, ".")
}
