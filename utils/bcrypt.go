package utils

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	// Bersihkan spasi atau karakter newline di awal/akhir
    p := strings.TrimSpace(password)
    h := strings.TrimSpace(hash)

	err := bcrypt.CompareHashAndPassword([]byte(h), []byte(p))
	return err == nil
}
