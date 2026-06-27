package handler

import "net/http"

// CLIDir holds the cross-compiled CLI binaries served at /cli/.
const CLIDir = "cli-dist"

// RegisterCLIHandler serves the CLI binaries publicly (no auth) so the /help
// page can link straight to them.
func RegisterCLIHandler(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir(CLIDir))
	mux.Handle("/cli/", http.StripPrefix("/cli/", fs))
}
