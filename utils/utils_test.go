package utils

import (
	"strings"
	"testing"
)

// ── GenerateRandomPassword ────────────────────────────────────

func TestGenerateRandomPassword_PanjangDanAlfabet(t *testing.T) {
	for i := 0; i < 50; i++ {
		p := GenerateRandomPassword()
		if len(p) != 14 {
			t.Fatalf("panjang harus 14, got %d (%q)", len(p), p)
		}
		for _, ch := range p {
			if !strings.ContainsRune(passwordAlphabet, ch) {
				t.Fatalf("karakter %q di luar alfabet", ch)
			}
		}
	}
}

func TestGenerateRandomPassword_TidakBerulang(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		p := GenerateRandomPassword()
		if seen[p] {
			t.Fatalf("password berulang: %q", p)
		}
		seen[p] = true
	}
}

// ── RenderTemplate ────────────────────────────────────────────

func TestRenderTemplate_GantiSatuVariabel(t *testing.T) {
	tmpl := "Halo {{nama_customer}}, selamat datang!"
	got := RenderTemplate(tmpl, map[string]string{"nama_customer": "Budi"})
	want := "Halo Budi, selamat datang!"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTemplate_GantiBanyakVariabel(t *testing.T) {
	tmpl := "Voucher {{kode}} berlaku sampai {{tanggal}}"
	got := RenderTemplate(tmpl, map[string]string{
		"kode":    "PROMO2025",
		"tanggal": "31 Des 2025",
	})
	want := "Voucher PROMO2025 berlaku sampai 31 Des 2025"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTemplate_VariabelTidakAda(t *testing.T) {
	// Key tidak ada di vars → placeholder tetap di output
	tmpl := "Halo {{unknown}}"
	got := RenderTemplate(tmpl, map[string]string{})
	if got != "Halo {{unknown}}" {
		t.Errorf("got %q, want 'Halo {{unknown}}'", got)
	}
}

func TestRenderTemplate_TemplateKosong(t *testing.T) {
	got := RenderTemplate("", map[string]string{"key": "val"})
	if got != "" {
		t.Errorf("template kosong harus menghasilkan string kosong, got %q", got)
	}
}

func TestRenderTemplate_VarsKosong(t *testing.T) {
	tmpl := "Tidak ada variabel"
	got := RenderTemplate(tmpl, nil)
	if got != tmpl {
		t.Errorf("vars kosong harus mengembalikan template asli, got %q", got)
	}
}

// ── GetTemplateOrFallback ─────────────────────────────────────

func TestGetTemplateOrFallback_ReturnTemplate(t *testing.T) {
	val := "Isi template dari DB"
	got := GetTemplateOrFallback(&val, "fallback")
	if got != val {
		t.Errorf("got %q, want %q", got, val)
	}
}

func TestGetTemplateOrFallback_NilReturnFallback(t *testing.T) {
	got := GetTemplateOrFallback(nil, "fallback teks")
	if got != "fallback teks" {
		t.Errorf("nil template: got %q, want 'fallback teks'", got)
	}
}

func TestGetTemplateOrFallback_EmptyReturnFallback(t *testing.T) {
	empty := ""
	got := GetTemplateOrFallback(&empty, "fallback teks")
	if got != "fallback teks" {
		t.Errorf("template kosong: got %q, want 'fallback teks'", got)
	}
}
