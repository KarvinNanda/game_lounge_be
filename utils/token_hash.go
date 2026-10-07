package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken mengembalikan SHA-256 (hex) dari token reset password. Yang disimpan
// di DB hanya hash-nya: jika DB/backup/log bocor, isinya tidak bisa dipakai
// sebagai link reset. Token sudah 256-bit acak, jadi tidak perlu salt/bcrypt.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
