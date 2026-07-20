package utils

import "testing"

// ── GeneratePasswordFromName ──────────────────────────────────

func TestGeneratePasswordFromName_SubstitusiKarakter(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		// Huruf PERTAMA selalu uppercase (bukan disubstitusi), sisanya disubstitusi
		// 'i' → '1'
		{"Budi Santoso", "Bud1#Gl"},
		// 'A' (huruf pertama) → kapital 'A'; ndre → n,d,r,'e'→'3'
		{"Andre", "Andr3#Gl"},
		// 'S' (huruf pertama) → kapital 'S'; anti → '@','n','7','1'
		{"Santi", "S@n71#Gl"},
		// 'R' kapital; atna → '@','7','n','@'
		{"Ratna", "R@7n@#Gl"},
		// nama kosong → fallback
		{"", "GameLounge#1"},
		// 'T' kapital; es → '3','$'
		{"tes", "T3$#Gl"},
	}
	for _, c := range cases {
		got := GeneratePasswordFromName(c.name)
		if got != c.want {
			t.Errorf("GeneratePasswordFromName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestGeneratePasswordFromName_HanyaAmbilKataPertama(t *testing.T) {
	// Nama dengan banyak kata → hanya kata pertama yang diproses
	got := GeneratePasswordFromName("Ahmad Budi Santoso")
	want := GeneratePasswordFromName("Ahmad")
	if got != want {
		t.Errorf("nama multi-kata: got %q, want %q", got, want)
	}
}

func TestGeneratePasswordFromName_SuffixSelalu(t *testing.T) {
	// Setiap nama selalu diakhiri "#Gl"
	names := []string{"Citra", "Doni", "XYZ"}
	for _, name := range names {
		got := GeneratePasswordFromName(name)
		if len(got) < 3 {
			t.Errorf("hasil terlalu pendek untuk nama %q: %q", name, got)
		}
		suffix := got[len(got)-3:]
		if suffix != "#Gl" {
			t.Errorf("GeneratePasswordFromName(%q) = %q, suffix harus '#Gl'", name, got)
		}
	}
}

func TestGeneratePasswordFromName_HurufPertamaKapital(t *testing.T) {
	// Huruf pertama output selalu kapital
	cases := []string{"budi", "santi", "rizki"}
	for _, name := range cases {
		got := GeneratePasswordFromName(name)
		if len(got) == 0 {
			t.Fatal("output kosong")
		}
		first := rune(got[0])
		if first < 'A' || first > 'Z' {
			t.Errorf("GeneratePasswordFromName(%q) = %q: huruf pertama harus kapital", name, got)
		}
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
