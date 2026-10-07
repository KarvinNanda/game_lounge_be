package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ClampPerPage membatasi query per_page (semua endpoint list) agar client tidak
// bisa meminta jutaan baris sekaligus. Dipasang global supaya tiap controller
// tidak perlu mengulang validasi yang sama.
func ClampPerPage(max int) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Request.URL.Query()
		if n, err := strconv.Atoi(q.Get("per_page")); err == nil && n > max {
			q.Set("per_page", strconv.Itoa(max))
			c.Request.URL.RawQuery = q.Encode()
		}
		c.Next()
	}
}

// SVGAsAttachment: file .svg yang dibuka langsung di browser diunduh, bukan
// dirender (SVG bisa berisi script). Di <img src> SVG tetap tampil normal,
// karena <img> tidak pernah menjalankan script dan mengabaikan header ini.
func SVGAsAttachment() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".svg") {
			c.Header("Content-Disposition", "attachment")
		}
		c.Next()
	}
}
