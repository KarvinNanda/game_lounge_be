package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func serveWithID(t *testing.T, incoming string) (header, inContext string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) { inContext = c.GetString("request_id") })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if incoming != "" {
		req.Header.Set("X-Request-ID", incoming)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Header().Get("X-Request-ID"), inContext
}

func TestRequestID_GeneratedWhenMissing(t *testing.T) {
	h, ctx := serveWithID(t, "")
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(h) || h != ctx {
		t.Errorf("want a 32-hex id in header and context, got header %q context %q", h, ctx)
	}
}

func TestRequestID_KeepsSafeIncomingID(t *testing.T) {
	h, ctx := serveWithID(t, "fe-abc_123")
	if h != "fe-abc_123" || ctx != h {
		t.Errorf("safe incoming id must be kept, got %q / %q", h, ctx)
	}
}

func TestRequestID_ReplacesUnsafeIncomingID(t *testing.T) {
	for _, bad := range []string{`x" ,"level":"ERROR`, "has space", strings.Repeat("a", 65)} {
		h, _ := serveWithID(t, bad)
		if h == bad || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(h) {
			t.Errorf("unsafe id %q must be replaced, got %q", bad, h)
		}
	}
}
