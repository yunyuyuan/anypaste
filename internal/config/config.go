// Package config persists the two runtime secrets (JWT signing key and the
// admin password hash) in a JSON file instead of environment variables, so a
// container only needs a writable data volume — no secrets to inject.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type data struct {
	JwtSecret string `json:"jwt_secret"`
	AppPasswd string `json:"app_passwd"` // bcrypt hash; empty until the admin sets a password
}

// Store is the in-memory config backed by a file on disk. Safe for concurrent use.
type Store struct {
	mu   sync.RWMutex
	path string
	d    data
}

// Load reads the config file (an absent file is treated as empty), then ensures
// a JWT secret exists — generating and persisting a random one on first run.
func Load(path string) (*Store, error) {
	s := &Store{path: path}

	switch b, err := os.ReadFile(path); {
	case err == nil:
		if err := json.Unmarshal(b, &s.d); err != nil {
			return nil, err
		}
	case !os.IsNotExist(err):
		return nil, err
	}

	if s.d.JwtSecret == "" {
		secret, err := randomSecret()
		if err != nil {
			return nil, err
		}
		s.d.JwtSecret = secret
		if err := s.save(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) JwtSecret() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.d.JwtSecret
}

func (s *Store) AppPasswd() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.d.AppPasswd
}

// Initialized reports whether an admin password has been set.
func (s *Store) Initialized() bool {
	return s.AppPasswd() != ""
}

// SetAppPasswd stores the bcrypt hash and persists it.
func (s *Store) SetAppPasswd(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d.AppPasswd = hash
	return s.save()
}

// save writes the file. Callers hold the write lock (or run before serving).
func (s *Store) save() error {
	if dir := filepath.Dir(s.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(s.d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
