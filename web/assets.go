package webassets

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Handler struct {
	assets   http.Handler
	distDir  string
	fallback []byte
}

func NewHandler() http.Handler {
	distDir := filepath.Join("web", "dist")
	index, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		index = []byte(`<!doctype html><html><head><meta charset="utf-8"><title>BScout</title></head><body><div id="root"></div></body></html>`)
	}
	return Handler{
		assets:   http.FileServer(http.Dir(distDir)),
		distDir:  distDir,
		fallback: index,
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
