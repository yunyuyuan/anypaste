package handler

import (
	"net/http"
	"strings"
)

// helpText is a concise, curl-friendly CLI guide. {base} is replaced with the
// request's scheme://host so the printed URLs are copy-pasteable.
const helpText = `anypaste CLI
============

Download (pick your platform):

  Linux   x64      {base}/cli/anypaste-linux-amd64
  Linux   arm64    {base}/cli/anypaste-linux-arm64
  macOS   arm64    {base}/cli/anypaste-darwin-arm64
  macOS   x64      {base}/cli/anypaste-darwin-amd64
  Windows x64      {base}/cli/anypaste-windows-amd64.exe
  Windows arm64    {base}/cli/anypaste-windows-arm64.exe

Quick install (Linux / macOS, x64 shown):

  curl -fsSL {base}/cli/anypaste-linux-amd64 -o anypaste
  chmod +x anypaste
  sudo mv anypaste /usr/local/bin/

Usage:

  anypaste login --server {base}/api     Log in (prompts for the password)
  anypaste up -m "some text"             Create a text paste
  anypaste up ./report.pdf               Create a paste and upload a file
  anypaste up -m note ./f --expire 24h   ...with an expiry (e.g. 30m, 24h)
  anypaste ls                            List pastes
  anypaste down <id> -o ./out.pdf        Download a paste's file (id is enough)
  anypaste logout                        Forget the stored token

Run "anypaste help" for the full reference.
`

// RegisterHelpHandler serves the plain-text CLI guide at /help (no auth), meant
// for `curl <host>/help`.
func RegisterHelpHandler(mux *http.ServeMux) {
	mux.HandleFunc("/help", helpHandler)
}

func helpHandler(w http.ResponseWriter, r *http.Request) {
	base := requestBaseURL(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strings.ReplaceAll(helpText, "{base}", base)))
}

// requestBaseURL reconstructs scheme://host, honoring a reverse proxy's
// X-Forwarded-Proto when present.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
