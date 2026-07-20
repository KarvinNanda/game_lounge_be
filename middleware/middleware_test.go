package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newEngine membuat gin engine minimal dengan middleware yang dites.
func newEngine(mw ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	for _, m := range mw {
		r.Use(m)
	}
	r.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	r.POST("/login", func(c *gin.Context) { c.String(200, "ok") })
	return r
}

// ── SecurityHeaders ───────────────────────────────────────────

func TestSecurityHeaders_SemuaHeaderTerpasang(t *testing.T) {
	t.Setenv("ENABLE_HSTS", "true")
	r := newEngine(SecurityHeaders())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(w, req)

	expect := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'; sandbox",
		"Cross-Origin-Resource-Policy": "cross-origin",
		"Strict-Transport-Security":    "max-age=31536000; includeSubDomains",
	}
	for k, want := range expect {
		if got := w.Header().Get(k); got != want {
			t.Errorf("header %s = %q, want %q", k, got, want)
		}
	}
}

func TestSecurityHeaders_HSTSMatiSecaraDefault(t *testing.T) {
	t.Setenv("ENABLE_HSTS", "")
	r := newEngine(SecurityHeaders())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ping", nil))

	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS tidak boleh terpasang saat ENABLE_HSTS tidak diset")
	}
}

// ── CORSMiddleware ────────────────────────────────────────────

func TestCORS_DevMode_EchoOrigin(t *testing.T) {
	// Dev mode: origin di-echo (bukan "*") agar kompatibel dengan
	// credentials/cookie — browser menolak "*" + credentials.
	t.Setenv("ALLOWED_ORIGINS", "")
	r := newEngine(CORSMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://siapapun.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://siapapun.com" {
		t.Errorf("dev mode: Allow-Origin = %q, want origin di-echo", got)
	}
}

func TestCORS_DevMode_TanpaOrigin_Wildcard(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := newEngine(CORSMiddleware())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ping", nil))

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("tanpa Origin header: Allow-Origin = %q, want '*'", got)
	}
}

func TestCORS_Production_OriginTerdaftar(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	r := newEngine(CORSMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Errorf("origin terdaftar: Allow-Origin = %q, want origin itu sendiri", got)
	}
}

func TestCORS_Production_OriginAsing_Ditolak(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	r := newEngine(CORSMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://jahat.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origin asing: Allow-Origin harus kosong, got %q", got)
	}
}

func TestCORS_Preflight_OPTIONS204(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := newEngine(CORSMiddleware())
	r.OPTIONS("/ping", func(c *gin.Context) {}) // route agar tidak 404 duluan

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("preflight OPTIONS: status = %d, want 204", w.Code)
	}
}

// ── RateLimit ─────────────────────────────────────────────────

func TestRateLimit_BlokirSetelahBatas(t *testing.T) {
	r := newEngine(RateLimit(3, time.Minute))

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("request ke-%d harus 200, got %d", i, w.Code)
		}
	}

	// Request ke-4 → 429
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("request ke-4 harus 429, got %d", w.Code)
	}
}

func TestRateLimit_IPBerbeda_TidakSalingBlokir(t *testing.T) {
	r := newEngine(RateLimit(2, time.Minute))

	// IP pertama habiskan kuota
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
	}

	// IP kedua harus tetap bisa
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("IP berbeda harus tetap 200, got %d", w.Code)
	}
}

func TestRateLimit_EndpointBerbeda_KuotaTerpisah(t *testing.T) {
	limiter := RateLimit(1, time.Minute)
	r := gin.New()
	r.POST("/login", limiter, func(c *gin.Context) { c.String(200, "ok") })
	r.POST("/forgot", limiter, func(c *gin.Context) { c.String(200, "ok") })

	// Habiskan kuota /login
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)

	// /forgot masih harus bisa (kuota per endpoint)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/forgot", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("endpoint berbeda harus punya kuota terpisah, got %d", w.Code)
	}
}

// ── MaxBodySize ───────────────────────────────────────────────

func TestMaxBodySize_BodyBesar_Ditolak(t *testing.T) {
	r := gin.New()
	r.Use(MaxBodySize(10)) // 10 byte
	r.POST("/data", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/data", strings.NewReader(strings.Repeat("x", 100)))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("body 100 byte dengan limit 10 harus ditolak, got %d", w.Code)
	}
}

func TestMaxBodySize_BodyKecil_Lolos(t *testing.T) {
	r := gin.New()
	r.Use(MaxBodySize(1024))
	r.POST("/data", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/data", strings.NewReader("kecil"))
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("body kecil harus lolos, got %d", w.Code)
	}
}

// ── AuthMiddleware (cookie-based) ─────────────────────────────

func TestAuthMiddleware_TanpaCookie_401(t *testing.T) {
	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/protected", nil))
	if w.Code != 401 {
		t.Errorf("tanpa cookie harus 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_HeaderAuthorization_TidakDiterima(t *testing.T) {
	// Cookie-only: token via Authorization header TIDAK lagi diterima
	t.Setenv("JWT_SECRET", "test-secret")
	token, _ := utils.GenerateJWT("staff-1", "admin", 1, true, nil)

	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("token valid via header harus tetap 401 (cookie-only), got %d", w.Code)
	}
}

func TestAuthMiddleware_CookiePalsu_401(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.StaffCookieName, Value: "token.palsu.disini"})
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("cookie palsu harus 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_CookieValid_200(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	token, err := utils.GenerateJWT("staff-1", "admin", 1, true, nil)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}

	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.String(200, c.GetString("staff_username"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.StaffCookieName, Value: token})
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("cookie valid harus 200, got %d", w.Code)
	}
	if w.Body.String() != "admin" {
		t.Errorf("staff_username harus terpasang di context, got %q", w.Body.String())
	}
}

// ── CustomerAuth (cookie-based) ───────────────────────────────

func TestCustomerAuth_CookieValid_200(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	token, _ := utils.GenerateCustomerJWT("cust-1", "member")

	r := gin.New()
	r.GET("/me", CustomerAuth(), func(c *gin.Context) {
		c.String(200, c.GetString("customer_id"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/me", nil)
	req.AddCookie(&http.Cookie{Name: utils.CustomerCookieName, Value: token})
	r.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "cust-1" {
		t.Errorf("cookie customer valid harus 200 + customer_id di context, got %d %q", w.Code, w.Body.String())
	}
}

func TestCustomerAuth_StaffCookie_Ditolak(t *testing.T) {
	// Staff token di cookie customer → ditolak (claims customer_id kosong)
	t.Setenv("JWT_SECRET", "test-secret")
	staffToken, _ := utils.GenerateJWT("staff-1", "admin", 1, true, nil)

	r := gin.New()
	r.GET("/me", CustomerAuth(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/me", nil)
	req.AddCookie(&http.Cookie{Name: utils.CustomerCookieName, Value: staffToken})
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("staff token di endpoint customer harus 401, got %d", w.Code)
	}
}

// ── CSRF origin check ─────────────────────────────────────────

func TestCSRF_PostDariOriginAsing_403(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	token, _ := utils.GenerateJWT("staff-1", "admin", 1, true, nil)

	r := gin.New()
	r.POST("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })

	// Cookie valid TAPI Origin situs jahat → harus 403 (CSRF)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.StaffCookieName, Value: token})
	req.Header.Set("Origin", "https://jahat.com")
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("POST dengan cookie valid dari origin asing harus 403, got %d", w.Code)
	}
}

func TestCSRF_PostDariOriginTerdaftar_Lolos(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	token, _ := utils.GenerateJWT("staff-1", "admin", 1, true, nil)

	r := gin.New()
	r.POST("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.StaffCookieName, Value: token})
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("POST dari origin terdaftar harus 200, got %d", w.Code)
	}
}

func TestCSRF_GetTidakDicek(t *testing.T) {
	// GET bukan state-changing → origin asing tetap lolos CSRF check
	// (tetap butuh cookie valid untuk auth)
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	token, _ := utils.GenerateJWT("staff-1", "admin", 1, true, nil)

	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.StaffCookieName, Value: token})
	req.Header.Set("Origin", "https://jahat.com")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("GET tidak dicek CSRF, harus 200, got %d", w.Code)
	}
}
