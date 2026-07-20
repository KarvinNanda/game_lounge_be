package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders menambahkan HTTP security headers standar ke semua response.
// API ini hanya menyajikan JSON + static image, sehingga CSP ketat aman dipakai:
// - default-src 'none' + sandbox menetralkan script di file SVG yang diupload
//   (stored XSS) saat dibuka langsung di browser.
// - nosniff mencegah browser menebak MIME type file upload.
func SecurityHeaders() gin.HandlerFunc {
	enableHSTS := os.Getenv("ENABLE_HSTS") == "true"

	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; sandbox")
		h.Set("Cross-Origin-Resource-Policy", "cross-origin") // izinkan FE lain-origin memuat gambar
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if enableHSTS {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
