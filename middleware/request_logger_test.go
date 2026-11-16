package middleware

import (
	"bytes"
	"encoding/json"
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

func TestRequestLogger_OneJSONLinePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	r := gin.New()
	r.Use(RequestID(), requestLoggerTo(&buf))
	r.GET("/api/admin/x", func(c *gin.Context) {
		c.Set("staff_id", "staff-1") // what AuthMiddleware sets
		c.Status(http.StatusNotFound)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/x?page=2", nil)
	req.Header.Set("X-Request-ID", "req-42")
	r.ServeHTTP(httptest.NewRecorder(), req)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d: %q", len(lines), buf.String())
	}
	var e map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &e); err != nil {
		t.Fatalf("not JSON: %v: %s", err, lines[0])
	}
	want := map[string]any{"msg": "request", "level": "WARN", "request_id": "req-42", "user_id": "staff-1",
		"user_type": "staff", "method": "GET", "path": "/api/admin/x", "status": float64(404)}
	for k, v := range want {
		if e[k] != v {
			t.Errorf("%s = %#v, want %#v", k, e[k], v)
		}
	}
	if _, ok := e["latency_ms"]; !ok {
		t.Error("latency_ms missing")
	}
}

func TestRequestLogger_CustomerUserAndLevels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	r := gin.New()
	r.Use(RequestID(), requestLoggerTo(&buf))
	r.GET("/ok", func(c *gin.Context) { c.Set("customer_id", "cust-1"); c.Status(200) })
	r.GET("/boom", func(c *gin.Context) { c.Status(500) })
	for _, p := range []string{"/ok", "/boom"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, p, nil))
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var ok, boom map[string]any
	json.Unmarshal([]byte(lines[0]), &ok)
	json.Unmarshal([]byte(lines[1]), &boom)
	if ok["level"] != "INFO" || ok["user_id"] != "cust-1" || ok["user_type"] != "customer" {
		t.Errorf("customer request: %v", ok)
	}
	if boom["level"] != "ERROR" || boom["user_id"] != nil {
		t.Errorf("500 must be ERROR and anonymous has no user_id: %v", boom)
	}
}
