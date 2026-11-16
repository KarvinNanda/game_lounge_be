package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func fnbRoutes(r *gin.Engine) []string {
	var out []string
	for _, rt := range r.Routes() {
		if strings.Contains(rt.Path, "/fnb/") {
			out = append(out, rt.Method+" "+rt.Path)
		}
	}
	return out
}

func TestFnbRoutes_DisabledByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("FNB_ENABLED", "")
	r := SetupRouter()

	if got := fnbRoutes(r); len(got) != 0 {
		t.Errorf("FnB routes must not be registered when FNB_ENABLED is unset, got %v", got)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/fnb/menu", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /api/public/fnb/menu = %d, want 404", w.Code)
	}
}

func TestFnbRoutes_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("FNB_ENABLED", "true")
	r := SetupRouter()

	// 10 admin + 2 customer + 1 public.
	if got := fnbRoutes(r); len(got) != 13 {
		t.Errorf("want 13 FnB routes, got %d: %v", len(got), got)
	}
}
