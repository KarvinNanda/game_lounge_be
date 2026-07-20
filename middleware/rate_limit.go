package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit membatasi jumlah request per IP per endpoint dalam satu window
// (sliding window, in-memory). Dipakai untuk endpoint sensitif seperti login
// dan forgot-password agar tahan brute force.
//
// Catatan deploy: agar c.ClientIP() akurat di belakang reverse proxy,
// set env TRUSTED_PROXIES (lihat routes.SetupRouter).
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	visitors := make(map[string][]time.Time)

	// Janitor: bersihkan entry yang sudah kadaluwarsa agar memori tidak bocor.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-window)
			mu.Lock()
			for key, ts := range visitors {
				valid := ts[:0]
				for _, t := range ts {
					if t.After(cutoff) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(visitors, key)
				} else {
					visitors[key] = valid
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP() + "|" + c.FullPath()
		now := time.Now()
		cutoff := now.Add(-window)

		mu.Lock()
		ts := visitors[key]
		valid := ts[:0]
		for _, t := range ts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) >= maxRequests {
			visitors[key] = valid
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Terlalu banyak percobaan. Silakan coba lagi beberapa saat lagi.",
			})
			return
		}
		visitors[key] = append(valid, now)
		mu.Unlock()

		c.Next()
	}
}

// MaxBodySize membatasi ukuran request body untuk mencegah payload raksasa
// menghabiskan memori server.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
