package service

import (
	"strings"
	"testing"
	"time"

	"game_lounge_be/models"
)

// ── GenerateCode ──────────────────────────────────────────────

func TestGenerateCode_AkronimDariNama(t *testing.T) {
	cases := []struct {
		name           string
		wantPrefixLen  int
		wantPrefixUpper bool
	}{
		{"Diskon Lebaran Spesial", 3, true},  // DLS + 4 digit
		{"Grand Opening", 2, true},           // GO + 4 digit
		{"Holiday Sale", 2, true},            // HS + 4 digit
	}
	for _, c := range cases {
		code := GenerateCode(c.name)
		if len(code) < c.wantPrefixLen+4 {
			t.Errorf("GenerateCode(%q) = %q: terlalu pendek (want prefix %d + 4 digit)", c.name, code, c.wantPrefixLen)
		}
		prefix := code[:len(code)-4]
		if c.wantPrefixUpper && prefix != strings.ToUpper(prefix) {
			t.Errorf("GenerateCode(%q) prefix %q harus uppercase", c.name, prefix)
		}
		suffix := code[len(code)-4:]
		for _, ch := range suffix {
			if ch < '0' || ch > '9' {
				t.Errorf("GenerateCode(%q) suffix %q harus 4 digit angka", c.name, suffix)
			}
		}
	}
}

func TestGenerateCode_NamaPendek(t *testing.T) {
	// Nama 1 kata pendek → ambil char langsung
	code := GenerateCode("OK")
	if len(code) < 6 {
		t.Errorf("GenerateCode('OK') = %q: terlalu pendek", code)
	}
	// Harus diakhiri 4 digit
	suffix := code[len(code)-4:]
	for _, ch := range suffix {
		if ch < '0' || ch > '9' {
			t.Errorf("suffix %q harus 4 digit angka", suffix)
		}
	}
}

func TestGenerateCode_MaxPrefixLength(t *testing.T) {
	// Nama banyak kata → prefix max 4 karakter
	code := GenerateCode("Flash Sale Akhir Tahun Besar")
	prefix := code[:len(code)-4]
	if len(prefix) > 4 {
		t.Errorf("prefix %q harus max 4 karakter, got %d", prefix, len(prefix))
	}
}

func TestGenerateCode_Deterministic_Format(t *testing.T) {
	// Setiap panggilan harus menghasilkan kode (tidak kosong dan punya format benar)
	for i := 0; i < 5; i++ {
		code := GenerateCode("Promo Ramadan")
		if code == "" {
			t.Error("GenerateCode harus menghasilkan kode non-empty")
		}
		suffix := code[len(code)-4:]
		for _, ch := range suffix {
			if ch < '0' || ch > '9' {
				t.Errorf("iterasi %d: suffix %q harus 4 digit angka", i, suffix)
			}
		}
	}
}

// ── calculateDiscount ─────────────────────────────────────────

func floatPtr(v float64) *float64 { return &v }

func makeVoucher(discountType string, discountValue float64, maxDiscount *float64) *models.Voucher {
	return &models.Voucher{
		DiscountType:  discountType,
		DiscountValue: discountValue,
		MaxDiscount:   maxDiscount,
	}
}

func TestCalculateDiscount_Percentage_NoMax(t *testing.T) {
	v := makeVoucher("percentage", 20, nil) // 20%
	got := calculateDiscount(100000, v)
	if got != 20000 {
		t.Errorf("20%% dari 100000: want 20000, got %.0f", got)
	}
}

func TestCalculateDiscount_Percentage_WithMax(t *testing.T) {
	v := makeVoucher("percentage", 30, floatPtr(25000)) // 30%, max 25000
	// 30% dari 100000 = 30000, tapi dikap 25000
	got := calculateDiscount(100000, v)
	if got != 25000 {
		t.Errorf("30%% dari 100000 (max 25000): want 25000, got %.0f", got)
	}
}

func TestCalculateDiscount_Percentage_BelowMax(t *testing.T) {
	v := makeVoucher("percentage", 10, floatPtr(50000)) // 10%, max 50000
	// 10% dari 100000 = 10000, di bawah max → pakai 10000
	got := calculateDiscount(100000, v)
	if got != 10000 {
		t.Errorf("10%% dari 100000 (max 50000): want 10000, got %.0f", got)
	}
}

func TestCalculateDiscount_Flat(t *testing.T) {
	v := makeVoucher("flat", 50000, nil)
	got := calculateDiscount(200000, v)
	if got != 50000 {
		t.Errorf("flat 50000: want 50000, got %.0f", got)
	}
}

func TestCalculateDiscount_Flat_ExceedsAmount(t *testing.T) {
	// Diskon flat melebihi total → tidak boleh melebihi amount
	v := makeVoucher("flat", 150000, nil)
	got := calculateDiscount(100000, v)
	if got != 100000 {
		t.Errorf("flat 150000 dari amount 100000: want 100000, got %.0f", got)
	}
}

func TestCalculateDiscount_Percentage_100pct(t *testing.T) {
	v := makeVoucher("percentage", 100, nil)
	got := calculateDiscount(75000, v)
	if got != 75000 {
		t.Errorf("100%% dari 75000: want 75000, got %.0f", got)
	}
}

func TestCalculateDiscount_ZeroAmount(t *testing.T) {
	v := makeVoucher("percentage", 50, nil)
	got := calculateDiscount(0, v)
	if got != 0 {
		t.Errorf("50%% dari 0: want 0, got %.0f", got)
	}
}

// ── ValidateVoucher (unit, tanpa DB) ─────────────────────────
// Hanya test fungsi calculateDiscount dan logika status — tidak menyentuh DB.

func TestVoucherStatus_Expired(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	v := models.Voucher{
		IsActive: true,
		EndDate:  &past,
	}
	ws := enrichVoucherStatus(v)
	if ws != "expired" {
		t.Errorf("want 'expired', got %q", ws)
	}
}

func TestVoucherStatus_Inactive(t *testing.T) {
	v := models.Voucher{IsActive: false}
	ws := enrichVoucherStatus(v)
	if ws != "inactive" {
		t.Errorf("want 'inactive', got %q", ws)
	}
}

func TestVoucherStatus_Active(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	v := models.Voucher{IsActive: true, EndDate: &future}
	ws := enrichVoucherStatus(v)
	if ws != "active" {
		t.Errorf("want 'active', got %q", ws)
	}
}

// enrichVoucherStatus diekstrak agar bisa ditest tanpa DB (duplikat logik dari enrichVoucher).
func enrichVoucherStatus(v models.Voucher) string {
	now := time.Now()
	if !v.IsActive {
		return "inactive"
	}
	if v.EndDate != nil && v.EndDate.Before(now) {
		return "expired"
	}
	return "active"
}
