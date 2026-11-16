package middleware

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/gin-gonic/gin"
)

// Password reset and admin recovery links carry a bearer token in the URL path.
// Anyone who can read the access log could use an unused token, so it is masked.
var pathTokenRE = regexp.MustCompile(`(/(?:reset-password|admin-recovery)/)[^/?]+`)

func redactPathTokens(path string) string {
	return pathTokenRE.ReplaceAllString(path, "${1}[REDACTED]")
}

// RequestLogger is gin's access log with tokens removed from the path.
// Use it with gin.New() instead of gin.Default().
func RequestLogger() gin.HandlerFunc {
	return requestLoggerTo(os.Stdout)
}

func requestLoggerTo(w io.Writer) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Output: w,
		Formatter: func(p gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n%s",
				p.TimeStamp.Format("2006/01/02 - 15:04:05"),
				p.StatusCode, p.Latency, p.ClientIP, p.Method,
				redactPathTokens(p.Path), p.ErrorMessage)
		},
	})
}
