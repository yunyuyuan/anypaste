package auth

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func GeneratePasswd(s string) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	fmt.Println(string(hash))
}

func VerifyPasswd(input string) error {
	return bcrypt.CompareHashAndPassword([]byte(os.Getenv("APP_PASSWD")), []byte(input))
}
