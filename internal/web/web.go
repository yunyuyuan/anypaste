// Package web embeds the built frontend (copied to ./dist at image build time)
// and serves it as a single-page app.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// all: also embeds dotfiles. A placeholder dist/index.html is committed so the
// module builds even without a frontend build; the Docker build overwrites
// ./dist with the real Vite output before `go build`.
//
//go:embed all:dist
var embedded embed.FS

// Handler serves the embedded SPA. Existing files are served directly; any
// other path falls back to index.html so client-side routes (e.g. /help) work
// on a hard reload.
func Handler() (http.Handler, error) {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, err
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		// Real asset → serve it; otherwise treat as an SPA route.
		if f, err := dist.Open(name); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}), nil
}
