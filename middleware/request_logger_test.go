package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRedactPathTokens(t *testing.T) {
	cases := map[string]string{
		"/api/customer/reset-password/abc123":          "/api/customer/reset-password/[REDACTED]",
		"/api/customer/reset-password/abc123/validate": "/api/customer/reset-password/[REDACTED]/validate",
		"/api/admin/admin-recovery/def456/reset":       "/api/admin/admin-recovery/[REDACTED]/reset",
		"/api/admin/admin-recovery/def456/validate":    "/api/admin/admin-recovery/[REDACTED]/validate",
		"/api/customer/bookings/42":                    "/api/customer/bookings/42",
		"/api/customer/forgot-password":                "/api/customer/forgot-password",
	}
	for in, want := range cases {
		if got := redactPathTokens(in); got != want {
			t.Errorf("redactPathTokens(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRequestLogger_DoesNotWriteToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	r := gin.New()
	r.Use(requestLoggerTo(&buf))
	r.GET("/api/customer/reset-password/:token/validate", func(c *gin.Context) { c.Status(http.StatusOK) })

	token := "9a21d1" + strings.Repeat("f", 58)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/customer/reset-password/"+token+"/validate", nil))

	if strings.Contains(buf.String(), token) {
		t.Errorf("token written to the access log: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "reset-password/[REDACTED]/validate") {
		t.Errorf("log line should keep the route shape: %s", buf.String())
	}
}
