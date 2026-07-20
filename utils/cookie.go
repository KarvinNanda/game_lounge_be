package utils

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Nama & path cookie auth. Dua sistem auth memakai cookie terpisah agar staff
// dan customer bisa login bersamaan di browser yang sama tanpa bentrok.
//
// Path membatasi ke rute mana browser MENGIRIM cookie:
//   - staff_token hanya terkirim ke /api/admin/* — tidak pernah ikut
//     ke rute customer.
//   - customer_token hanya terkirim ke /api/customer/* — tidak pernah ikut
//     ke rute admin.
const (
	StaffCookieName    = "staff_token"
	StaffCookiePath    = "/api/admin"
	CustomerCookieName = "customer_token"
	CustomerCookiePath = "/api/customer"
)

// legacyCookiePaths adalah path yang pernah dipakai versi sebelumnya.
// Saat set/clear cookie, versi lama di path ini ikut dihapus agar tidak
// ada cookie duplikat di browser user pra-migrasi.
var legacyCookiePaths = []string{"/", "/api"}

// cookieSecure menentukan flag Secure pada cookie.
// Default true (production/HTTPS). Set COOKIE_SECURE=false untuk local dev via http.
func cookieSecure() bool {
	return os.Getenv("COOKIE_SECURE") != "false"
}

// StaffTokenMaxAge mengembalikan umur cookie staff dalam detik,
// mengikuti JWT_EXPIRED_HOURS (default 24 jam) agar sinkron dengan expiry JWT.
func StaffTokenMaxAge() int {
	hours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRED_HOURS"))
	if hours <= 0 {
		hours = 24
	}
	return hours * 3600
}

// CustomerTokenMaxAge = 7 hari, sinkron dengan expiry customer JWT.
func CustomerTokenMaxAge() int {
	return 7 * 24 * 3600
}

// clearLegacyCookies menghapus versi cookie di path-path lama
// (browser meng-key cookie per name+domain+path).
func clearLegacyCookies(c *gin.Context, name, activePath string) {
	for _, p := range legacyCookiePaths {
		if p != activePath {
			c.SetCookie(name, "", -1, p, "", cookieSecure(), true)
		}
	}
}

// SetAuthCookie memasang httpOnly auth cookie pada path tertentu.
// HttpOnly → JavaScript tidak bisa membaca token (mitigasi XSS).
// SameSite=Lax → cookie tidak dikirim pada cross-site request (mitigasi CSRF dasar).
func SetAuthCookie(c *gin.Context, name, token string, maxAgeSeconds int, path string) {
	c.SetSameSite(http.SameSiteLaxMode)
	clearLegacyCookies(c, name, path)
	c.SetCookie(name, token, maxAgeSeconds, path, "", cookieSecure(), true)
}

// ClearAuthCookie menghapus auth cookie (dipakai saat logout),
// termasuk versi di path legacy.
func ClearAuthCookie(c *gin.Context, name, path string) {
	c.SetSameSite(http.SameSiteLaxMode)
	clearLegacyCookies(c, name, path)
	c.SetCookie(name, "", -1, path, "", cookieSecure(), true)
}
