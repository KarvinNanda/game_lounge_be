package utils

import (
	"strings"
	"unicode"
)

// substitutionMap menggantikan karakter umum dengan karakter "leet"-style
// agar password lebih kuat namun tetap mudah diingat.
var substitutionMap = map[rune]rune{
	'a': '@',
	'e': '3',
	'i': '1',
	'o': '0',
	's': '$',
	't': '7',
}

// GeneratePasswordFromName membuat password dari nama customer.
// Algoritma:
//  1. Ambil kata pertama dari nama (case-insensitive).
//  2. Terapkan substitusi karakter.
//  3. Kapitalisasi huruf pertama.
//  4. Tambahkan suffix "#Gl" (Game Lounge) di akhir agar panjang & ada simbol.
//
// Contoh: "Budi Santoso" → "Bud1#Gl"  (tidak ada 'a/e/i/o/s/t' di "Budi"
// kecuali 'i') → sebenarnya "Bud1#Gl"
func GeneratePasswordFromName(fullName string) string {
	// Ambil kata pertama
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "GameLounge#1"
	}
	word := strings.ToLower(parts[0])

	var sb strings.Builder
	for i, ch := range word {
		if i == 0 {
			// Huruf pertama selalu kapital
			sb.WriteRune(unicode.ToUpper(ch))
			continue
		}
		if sub, ok := substitutionMap[ch]; ok {
			sb.WriteRune(sub)
		} else {
			sb.WriteRune(ch)
		}
	}

	// Tambahkan suffix tetap agar selalu ada angka + simbol
	sb.WriteString("#Gl")

	return sb.String()
}
