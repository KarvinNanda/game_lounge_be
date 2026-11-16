package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/gin-gonic/gin"
)

// A client may send its own X-Request-ID (so FE and BE logs can be matched).
// Only short ids of letters, digits, '-' and '_' are kept: anything else could
// inject fake fields into the JSON log, so it is replaced.
var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// RequestID puts a request id in the context ("request_id") and in the
// X-Request-ID response header. Register it before RequestLogger.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !safeRequestID.MatchString(id) {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
