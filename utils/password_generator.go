package utils

import (
	"crypto/rand"
	"math/big"
)

// passwordAlphabet tanpa karakter yang mirip (0/O, 1/l/I) supaya mudah diketik dari email.
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// GenerateRandomPassword membuat password acak 14 karakter dari crypto/rand
// (~81 bit entropi). Dipakai untuk password awal customer dan reset password staff.
//
// Sebelumnya password diturunkan dari nama ("Budi" → "Bud1#Gl"), sehingga siapa pun
// yang tahu nama korban bisa menebaknya.
func GenerateRandomPassword() string {
	const length = 14
	max := big.NewInt(int64(len(passwordAlphabet)))
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand tidak pernah gagal di OS yang didukung; jika gagal, jangan
			// lanjut dengan password lemah.
			panic("crypto/rand gagal: " + err.Error())
		}
		b[i] = passwordAlphabet[n.Int64()]
	}
	return string(b)
}
