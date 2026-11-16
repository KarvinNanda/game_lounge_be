package middleware

import (
	"io"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

// Password reset and admin recovery links carry a bearer token in the URL path.
// Anyone who can read the access log could use an unused token, so it is masked.
var pathTokenRE = regexp.MustCompile(`(/(?:reset-password|admin-recovery)/)[^/?]+`)

func redactPathTokens(path string) string {
	return pathTokenRE.ReplaceAllString(path, "${1}[REDACTED]")
}

// RequestLogger writes one JSON event per request: request_id, user_id and
// user_type (staff or customer, set by the auth middlewares), method, path
// (tokens redacted, no query string), status and latency. Level follows the
// status: 5xx ERROR, 4xx WARN, else INFO. Use with gin.New() after RequestID().
func RequestLogger() gin.HandlerFunc {
	return requestLoggerTo(os.Stdout)
}

func requestLoggerTo(w io.Writer) gin.HandlerFunc {
	logger := slog.New(slog.NewJSONHandler(w, nil))
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}
		attrs := []slog.Attr{
			slog.String("request_id", c.GetString("request_id")),
			slog.String("method", c.Request.Method),
			slog.String("path", redactPathTokens(c.Request.URL.Path)),
			slog.Int("status", status),
			slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
			slog.String("client_ip", c.ClientIP()),
		}
		if id := c.GetString("staff_id"); id != "" {
			attrs = append(attrs, slog.String("user_id", id), slog.String("user_type", "staff"))
		} else if id := c.GetString("customer_id"); id != "" {
			attrs = append(attrs, slog.String("user_id", id), slog.String("user_type", "customer"))
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.String()))
		}
		logger.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}
