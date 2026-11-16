package utils

import "testing"

func TestXenditMockMode(t *testing.T) {
	cases := []struct {
		name, env, key string
		want           bool
	}{
		{"dev tanpa key → mock", "development", "", true},
		{"APP_ENV kosong tanpa key → mock", "", "", true},
		{"dev dengan key → real", "development", "xnd_dev_x", false},
		{"production tanpa key → TETAP bukan mock", "production", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("XENDIT_SECRET_KEY", tc.key)
			if got := XenditMockMode(); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestProductionConfigErrors(t *testing.T) {
	set := func(t *testing.T, kv map[string]string) {
		for k, v := range kv {
			t.Setenv(k, v)
		}
	}
	good := map[string]string{
		"APP_ENV":              "production",
		"ALLOWED_ORIGINS":      "https://admin.example.com",
		"XENDIT_SECRET_KEY":    "xnd_production_x",
		"XENDIT_WEBHOOK_TOKEN": "tok",
		"COOKIE_SECURE":        "",
		"APP_URL":              "https://app.example.com",
	}

	t.Run("config lengkap → tidak ada error", func(t *testing.T) {
		set(t, good)
		if errs := ProductionConfigErrors(); len(errs) != 0 {
			t.Errorf("tidak boleh ada error, got %v", errs)
		}
	})
	t.Run("bukan production → tidak dicek", func(t *testing.T) {
		set(t, good)
		t.Setenv("APP_ENV", "development")
		t.Setenv("ALLOWED_ORIGINS", "")
		if errs := ProductionConfigErrors(); len(errs) != 0 {
			t.Errorf("dev tidak dicek, got %v", errs)
		}
	})
	for _, missing := range []string{"ALLOWED_ORIGINS", "XENDIT_SECRET_KEY", "XENDIT_WEBHOOK_TOKEN", "APP_URL"} {
		t.Run(missing+" kosong → error", func(t *testing.T) {
			set(t, good)
			t.Setenv(missing, "")
			if errs := ProductionConfigErrors(); len(errs) != 1 {
				t.Errorf("want 1 error, got %v", errs)
			}
		})
	}
	t.Run("COOKIE_SECURE=false → error", func(t *testing.T) {
		set(t, good)
		t.Setenv("COOKIE_SECURE", "false")
		if errs := ProductionConfigErrors(); len(errs) != 1 {
			t.Errorf("want 1 error, got %v", errs)
		}
	})
}

func TestFnbEnabled(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"", false}, // default off: FnB is out of scope (decision 2026-11-16)
		{"false", false},
		{"yes", false},
		{"true", true},
	}
	for _, tc := range cases {
		t.Run("FNB_ENABLED="+tc.env, func(t *testing.T) {
			t.Setenv("FNB_ENABLED", tc.env)
			if got := FnbEnabled(); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
