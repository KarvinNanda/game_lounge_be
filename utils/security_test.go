package utils

import (
	"bytes"
	"strings"
	"testing"
)

// ── JWT ───────────────────────────────────────────────────────

func TestJWT_RoundTrip_Staff(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-untuk-unit-test")

	token, err := GenerateJWT("staff-1", "admin", 2, false, []string{"booking.read"})
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}
	if claims.StaffID != "staff-1" || claims.Username != "admin" || claims.RoleID != 2 {
		t.Errorf("claims tidak sesuai: %+v", claims)
	}
	if len(claims.Permissions) != 1 || claims.Permissions[0] != "booking.read" {
		t.Errorf("permissions tidak sesuai: %v", claims.Permissions)
	}
}

func TestJWT_RoundTrip_Customer(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-untuk-unit-test")

	token, err := GenerateCustomerJWT("cust-1", "member")
	if err != nil {
		t.Fatalf("GenerateCustomerJWT error: %v", err)
	}

	claims, err := ValidateCustomerJWT(token)
	if err != nil {
		t.Fatalf("ValidateCustomerJWT error: %v", err)
	}
	if claims.CustomerID != "cust-1" || claims.CustomerType != "member" {
		t.Errorf("claims tidak sesuai: %+v", claims)
	}
}

func TestJWT_EmptySecret_Ditolak(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	if _, err := GenerateJWT("s", "u", 1, false, nil); err == nil {
		t.Error("GenerateJWT harus error saat JWT_SECRET kosong")
	}
	if _, err := GenerateCustomerJWT("c", "member"); err == nil {
		t.Error("GenerateCustomerJWT harus error saat JWT_SECRET kosong")
	}
	if _, err := ValidateJWT("token-apapun"); err == nil {
		t.Error("ValidateJWT harus error saat JWT_SECRET kosong")
	}
	if _, err := ValidateCustomerJWT("token-apapun"); err == nil {
		t.Error("ValidateCustomerJWT harus error saat JWT_SECRET kosong")
	}
}

func TestJWT_TokenDipalsukan_Ditolak(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-asli")
	token, _ := GenerateJWT("staff-1", "admin", 1, true, nil)

	// Validasi dengan secret berbeda → harus gagal
	t.Setenv("JWT_SECRET", "secret-penyerang")
	if _, err := ValidateJWT(token); err == nil {
		t.Error("token dengan signature dari secret lain harus ditolak")
	}
}

func TestJWT_StaffToken_BukanCustomerToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-untuk-unit-test")

	// Staff token dipakai di endpoint customer → harus ditolak
	staffToken, _ := GenerateJWT("staff-1", "admin", 1, true, nil)
	if _, err := ValidateCustomerJWT(staffToken); err == nil {
		t.Error("staff token tidak boleh lolos validasi customer")
	}
}

func TestJWT_TokenRusak_Ditolak(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-untuk-unit-test")

	token, _ := GenerateJWT("staff-1", "admin", 1, false, nil)
	tampered := token[:len(token)-4] + "XXXX" // rusak signature

	if _, err := ValidateJWT(tampered); err == nil {
		t.Error("token dengan signature rusak harus ditolak")
	}
}

// ── VerifyXenditWebhook ───────────────────────────────────────

func TestVerifyXenditWebhook_MockMode(t *testing.T) {
	// Kedua env kosong → mock mode → semua diterima
	t.Setenv("XENDIT_WEBHOOK_TOKEN", "")
	t.Setenv("XENDIT_SECRET_KEY", "")
	if !VerifyXenditWebhook("apapun") {
		t.Error("mock mode (kedua env kosong) harus menerima webhook")
	}
}

func TestVerifyXenditWebhook_FailClosed(t *testing.T) {
	// Secret key ada tapi webhook token lupa diset → TOLAK (fail-closed)
	t.Setenv("XENDIT_WEBHOOK_TOKEN", "")
	t.Setenv("XENDIT_SECRET_KEY", "xnd_production_abc")
	if VerifyXenditWebhook("apapun") {
		t.Error("webhook harus DITOLAK saat production tapi webhook token belum diset")
	}
}

func TestVerifyXenditWebhook_TokenBenar(t *testing.T) {
	t.Setenv("XENDIT_WEBHOOK_TOKEN", "rahasia-webhook-123")
	if !VerifyXenditWebhook("rahasia-webhook-123") {
		t.Error("token yang benar harus diterima")
	}
}

func TestVerifyXenditWebhook_TokenSalah(t *testing.T) {
	t.Setenv("XENDIT_WEBHOOK_TOKEN", "rahasia-webhook-123")
	cases := []string{"", "salah", "rahasia-webhook-12", "rahasia-webhook-1234"}
	for _, c := range cases {
		if VerifyXenditWebhook(c) {
			t.Errorf("token %q harus ditolak", c)
		}
	}
}

// ── sanitizeFolder ────────────────────────────────────────────

func TestSanitizeFolder_PathTraversal(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"stores", "stores"},
		{"room_templates", "room_templates"},
		{"../../etc", "etc"},          // traversal dihapus
		{"..\\windows", "windows"},    // backslash dihapus
		{"a/b/c", "abc"},              // slash dihapus
		{"  spaced  ", "spaced"},
		{"nama-folder_1", "nama-folder_1"},
		{"<script>", "script"},
		{"", ""},
		{"../..", ""},
	}
	for _, c := range cases {
		if got := sanitizeFolder(c.input); got != c.want {
			t.Errorf("sanitizeFolder(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── validateFileContent ───────────────────────────────────────

func pngBytes() []byte {
	// Magic bytes PNG + padding
	head := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	return append(head, bytes.Repeat([]byte{0}, 100)...)
}

func jpegBytes() []byte {
	head := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	return append(head, bytes.Repeat([]byte{0}, 100)...)
}

func TestValidateFileContent_PNGValid(t *testing.T) {
	if _, err := validateFileContent(bytes.NewReader(pngBytes()), ".png"); err != nil {
		t.Errorf("PNG asli dengan ekstensi .png harus lolos: %v", err)
	}
}

func TestValidateFileContent_JPEGValid(t *testing.T) {
	if _, err := validateFileContent(bytes.NewReader(jpegBytes()), ".jpg"); err != nil {
		t.Errorf("JPEG asli dengan ekstensi .jpg harus lolos: %v", err)
	}
}

func TestValidateFileContent_HTMLMenyamarJadiJPG(t *testing.T) {
	html := []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>")
	if _, err := validateFileContent(bytes.NewReader(html), ".jpg"); err == nil {
		t.Error("file HTML dengan ekstensi .jpg harus DITOLAK (stored XSS)")
	}
}

func TestValidateFileContent_PNGMenyamarJadiJPG(t *testing.T) {
	// Isi PNG tapi ekstensi .jpg → tolak (mismatch)
	if _, err := validateFileContent(bytes.NewReader(pngBytes()), ".jpg"); err == nil {
		t.Error("isi PNG dengan ekstensi .jpg harus ditolak")
	}
}

func TestValidateFileContent_SVGBersih(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="red"/></svg>`)
	got, err := validateFileContent(bytes.NewReader(svg), ".svg")
	if err != nil {
		t.Fatalf("SVG bersih harus lolos: %v", err)
	}
	if !bytes.Equal(got, svg) {
		t.Error("isi SVG yang dikembalikan harus utuh")
	}
}

func TestValidateFileContent_SVGDenganScript(t *testing.T) {
	berbahaya := []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(document.cookie)</script></svg>`,
		`<svg onload="alert(1)" xmlns="http://www.w3.org/2000/svg"></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><a href="javascript:alert(1)">x</a></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><body>x</body></foreignObject></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><image href="x" onerror="alert(1)"/></svg>`,
	}
	for i, s := range berbahaya {
		if _, err := validateFileContent(strings.NewReader(s), ".svg"); err == nil {
			t.Errorf("kasus %d: SVG berbahaya harus DITOLAK: %s", i, s[:50])
		}
	}
}

func TestValidateFileContent_SVGTanpaTagSVG(t *testing.T) {
	notSVG := `hello ini cuma text biasa`
	if _, err := validateFileContent(strings.NewReader(notSVG), ".svg"); err == nil {
		t.Error("file text tanpa tag <svg> harus ditolak")
	}
}
