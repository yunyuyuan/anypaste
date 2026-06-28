package handler

import (
	"encoding/json"
	"net/http"
	"yunyuyuan/anypaste/internal/auth"
)

// RegisterInitHandler exposes first-run setup endpoints (no auth):
//   GET  /status  -> {"initialized": bool}
//   POST /init    -> {"password": "..."} sets the admin password (only once)
func RegisterInitHandler(mux *http.ServeMux) {
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/init", initHandler)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"initialized": auth.Initialized()})
}

func initHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Setup is a one-time action: once a password exists, lock this endpoint.
	if auth.Initialized() {
		http.Error(w, "already initialized", http.StatusConflict)
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Password == "" {
		http.Error(w, "password required", http.StatusBadRequest)
		return
	}

	if err := auth.SetPassword(body.Password); err != nil {
		http.Error(w, "failed to set password", http.StatusInternalServerError)
		return
	}

	// Log the new admin straight in (default session length).
	token, err := auth.IssueToken(0)
	if err != nil {
		http.Error(w, "issue token failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"token": token})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
