package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClampPerPage(t *testing.T) {
	r := gin.New()
	r.Use(ClampPerPage(100))
	r.GET("/x", func(c *gin.Context) { c.String(200, c.Query("per_page")) })
	for in, want := range map[string]string{"": "", "20": "20", "100": "100", "1000000": "100", "abc": "abc"} {
		w := httptest.NewRecorder()
		url := "/x"
		if in != "" {
			url += "?per_page=" + in
		}
		r.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Body.String() != want {
			t.Errorf("per_page=%q: want %q, got %q", in, want, w.Body.String())
		}
	}
}

func TestSVGAsAttachment(t *testing.T) {
	r := gin.New()
	r.Use(SVGAsAttachment())
	r.GET("/assets/img/*f", func(c *gin.Context) { c.String(200, "x") })
	for path, want := range map[string]string{"/assets/img/a.svg": "attachment", "/assets/img/A.SVG": "attachment", "/assets/img/a.png": ""} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if got := w.Header().Get("Content-Disposition"); got != want {
			t.Errorf("%s: want %q, got %q", path, want, got)
		}
	}
}
