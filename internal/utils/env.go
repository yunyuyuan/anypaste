package utils

import "os"

// EnvOr returns the value of environment variable key, or def if it is unset or empty.
func EnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
