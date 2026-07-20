package utils

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setCookieHeaders menjalankan fn pada gin context kosong dan
// mengembalikan semua header Set-Cookie yang dihasilkan.
func setCookieHeaders(t *testing.T, fn func(c *gin.Context)) []string {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	fn(c)
	return w.Header().Values("Set-Cookie")
}

func TestSetAuthCookie_CustomerPath(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "true")
	headers := setCookieHeaders(t, func(c *gin.Context) {
		SetAuthCookie(c, CustomerCookieName, "tok123", 3600, CustomerCookiePath)
	})

	// Harus ada cookie aktif dengan Path=/api/customer
	var active string
	for _, h := range headers {
		if strings.Contains(h, "tok123") {
			active = h
		}
	}
	if active == "" {
		t.Fatalf("tidak ada Set-Cookie dengan token: %v", headers)
	}
	for _, want := range []string{"Path=/api/customer", "HttpOnly", "SameSite=Lax", "Secure"} {
		if !strings.Contains(active, want) {
			t.Errorf("Set-Cookie %q harus mengandung %q", active, want)
		}
	}
}

func TestSetAuthCookie_StaffPath(t *testing.T) {
	headers := setCookieHeaders(t, func(c *gin.Context) {
		SetAuthCookie(c, StaffCookieName, "tok456", 3600, StaffCookiePath)
	})

	var active string
	for _, h := range headers {
		if strings.Contains(h, "tok456") {
			active = h
		}
	}
	if active == "" || !strings.Contains(active, "Path=/api/admin") {
		t.Errorf("staff cookie harus Path=/api/admin, got %q", active)
	}
}

func TestSetAuthCookie_HapusLegacyPathRoot(t *testing.T) {
	// Saat set cookie baru, cookie era lama (Path=/) harus ikut dihapus
	// agar tidak ada duplikat untuk user pra-migrasi.
	headers := setCookieHeaders(t, func(c *gin.Context) {
		SetAuthCookie(c, CustomerCookieName, "tok789", 3600, CustomerCookiePath)
	})

	foundLegacyDelete := false
	for _, h := range headers {
		if strings.Contains(h, CustomerCookieName+"=;") &&
			strings.Contains(h, "Path=/;") &&
			strings.Contains(h, "Max-Age=0") {
			foundLegacyDelete = true
		}
	}
	if !foundLegacyDelete {
		t.Errorf("harus ada Set-Cookie penghapus untuk Path=/ legacy: %v", headers)
	}
}

func TestClearAuthCookie_HapusSemuaPath(t *testing.T) {
	headers := setCookieHeaders(t, func(c *gin.Context) {
		ClearAuthCookie(c, StaffCookieName, StaffCookiePath)
	})

	// Path aktif (/api/admin) + legacy ("/" dan "/api") = 3 Set-Cookie
	if len(headers) != 3 {
		t.Fatalf("clear harus menghasilkan 3 Set-Cookie (aktif + 2 legacy), got %d: %v", len(headers), headers)
	}
	for _, h := range headers {
		if !strings.Contains(h, "Max-Age=0") {
			t.Errorf("semua Set-Cookie saat clear harus Max-Age=0: %q", h)
		}
	}
	// Pastikan ketiga path tercakup
	joined := strings.Join(headers, "\n")
	for _, p := range []string{"Path=/;", "Path=/api;", "Path=/api/admin"} {
		if !strings.Contains(joined, p) {
			t.Errorf("clear harus mencakup %q, got:\n%s", p, joined)
		}
	}
}

func TestCookieSecure_DevMode(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	headers := setCookieHeaders(t, func(c *gin.Context) {
		SetAuthCookie(c, CustomerCookieName, "tokdev", 3600, CustomerCookiePath)
	})

	for _, h := range headers {
		if strings.Contains(h, "tokdev") && strings.Contains(h, "Secure") {
			t.Errorf("COOKIE_SECURE=false: flag Secure tidak boleh ada: %q", h)
		}
	}
}
