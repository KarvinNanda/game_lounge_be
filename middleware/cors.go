package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	// ALLOWED_ORIGINS: comma-separated list, e.g.
	// "https://app.example.com,https://admin.example.com"
	// Jika kosong → default ke "*" (cocok untuk local dev).
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	allowedSet := map[string]bool{}
	if originsEnv != "" {
		for _, o := range strings.Split(originsEnv, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedSet[trimmed] = true
			}
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Auth memakai httpOnly cookie (credentials), dan browser MENOLAK
		// kombinasi "Access-Control-Allow-Origin: *" + credentials.
		// Karena itu origin selalu di-echo spesifik, tidak pernah wildcard
		// saat ada Origin header.
		var allowOrigin string
		if len(allowedSet) == 0 {
			// Dev mode: echo origin apapun agar cookie tetap terkirim
			if origin != "" {
				allowOrigin = origin
			} else {
				allowOrigin = "*"
			}
		} else if allowedSet[origin] {
			allowOrigin = origin
		}

		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
