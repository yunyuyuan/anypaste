package handler

import (
	"net/http"
	"strings"

	"yunyuyuan/anypaste/internal/clibin"
)

// RegisterCLIHandler serves the embedded CLI binaries publicly (no auth) at
// /cli/ so the /help page can link straight to them. Serving from the embedded
// FS (rather than a cli-dist directory on disk) keeps the server a single
// self-contained binary.
func RegisterCLIHandler(mux *http.ServeMux) error {
	binFS, err := clibin.FS()
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(binFS))
	mux.Handle("/cli/", http.StripPrefix("/cli/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		// No directory listing, and hide the committed placeholder / dotfiles.
		if name == "" || strings.HasPrefix(name, ".") {
			http.NotFound(w, r)
			return
		}
		f, err := binFS.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})))
	return nil
}
