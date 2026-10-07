package middleware

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// withScope memasang scope store seperti yang dilakukan AuthMiddleware.
func withScope(all bool, ids ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ctxStoreAll, all)
		c.Set(ctxStoreIDs, ids)
	}
}

func run(r *gin.Engine, method, url, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, url, strings.NewReader(body)))
	return w
}

func TestScopeStoreQuery(t *testing.T) {
	cases := []struct {
		name      string
		all       bool
		ids       []string
		url       string
		wantCode  int
		wantStore string
	}{
		{"semua store, tanpa filter", true, nil, "/x", 200, ""},
		{"1 store, tanpa filter → dipaksa ke store itu", false, []string{"s1"}, "/x", 200, "s1"},
		{"1 store, filter store sendiri", false, []string{"s1"}, "/x?store_id=s1", 200, "s1"},
		{"filter store lain", false, []string{"s1"}, "/x?store_id=s2", 403, ""},
		{"beberapa store, tanpa filter", false, []string{"s1", "s2"}, "/x", 400, ""},
		{"tanpa store sama sekali", false, nil, "/x", 403, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/x", withScope(tc.all, tc.ids...), ScopeStoreQuery(), func(c *gin.Context) { c.String(200, c.Query("store_id")) })
			w := run(r, "GET", tc.url, "")
			if w.Code != tc.wantCode || (tc.wantCode == 200 && w.Body.String() != tc.wantStore) {
				t.Errorf("want %d %q, got %d %q", tc.wantCode, tc.wantStore, w.Code, w.Body.String())
			}
		})
	}
}

func TestRequireStoreParam(t *testing.T) {
	r := gin.New()
	r.PUT("/pricing/:store_id", withScope(false, "s1"), RequireStoreParam("store_id"), func(c *gin.Context) { c.String(200, "ok") })
	if w := run(r, "PUT", "/pricing/s1", ""); w.Code != 200 {
		t.Errorf("store sendiri harus 200, got %d", w.Code)
	}
	if w := run(r, "PUT", "/pricing/s2", ""); w.Code != 403 {
		t.Errorf("store lain harus 403, got %d", w.Code)
	}
}

func TestRequireStoreInBody(t *testing.T) {
	r := gin.New()
	r.POST("/bookings", withScope(false, "s1"), RequireStoreInBody(), func(c *gin.Context) {
		var body struct {
			StoreID string `json:"store_id"`
		}
		_ = c.ShouldBindJSON(&body) // body harus tetap bisa dibaca handler
		c.String(200, body.StoreID)
	})
	if w := run(r, "POST", "/bookings", `{"store_id":"s1"}`); w.Code != 200 || w.Body.String() != "s1" {
		t.Errorf("store sendiri harus 200 dan body utuh, got %d %q", w.Code, w.Body.String())
	}
	if w := run(r, "POST", "/bookings", `{"store_id":"s2"}`); w.Code != 403 {
		t.Errorf("store lain harus 403, got %d", w.Code)
	}
}

func TestRequireEntityStore(t *testing.T) {
	lookup := func(id string) (string, error) {
		switch id {
		case "b1":
			return "s1", nil
		case "b2":
			return "s2", nil
		}
		return "", errors.New("not found")
	}
	r := gin.New()
	r.PATCH("/bookings/:id/cancel", withScope(false, "s1"), RequireEntityStore("id", lookup), func(c *gin.Context) { c.String(200, "ok") })
	if w := run(r, "PATCH", "/bookings/b1/cancel", ""); w.Code != 200 {
		t.Errorf("booking di store sendiri harus 200, got %d", w.Code)
	}
	if w := run(r, "PATCH", "/bookings/b2/cancel", ""); w.Code != 403 {
		t.Errorf("booking store lain harus 403, got %d", w.Code)
	}
	if w := run(r, "PATCH", "/bookings/nope/cancel", ""); w.Code != 200 {
		t.Errorf("tidak ditemukan → diteruskan ke handler (404 di sana), got %d", w.Code)
	}
}
