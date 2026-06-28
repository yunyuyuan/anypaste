package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// Initialized reports whether an admin password has been set yet.
func Initialized() bool {
	return store.Initialized()
}

// SetPassword hashes the plaintext password and persists it to the config store.
func SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return store.SetAppPasswd(string(hash))
}

// VerifyPasswd checks the input against the stored bcrypt hash.
func VerifyPasswd(input string) error {
	return bcrypt.CompareHashAndPassword([]byte(store.AppPasswd()), []byte(input))
}
