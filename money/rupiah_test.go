package money

import (
	"encoding/json"
	"testing"
)

func TestRupiah_ScanDecimalFromMySQL(t *testing.T) {
	cases := []struct {
		src  any
		want Rupiah
	}{
		{[]byte("50000.00"), 50000},
		{[]byte("2018000.00"), 2018000},
		{"64000.0", 64000},
		{[]byte("0.00"), 0},
		{[]byte("-1500.00"), -1500},
		{[]byte("12345"), 12345},
		{int64(7000), 7000},
		{float64(9000), 9000},
		{nil, 0}, // SUM() over no rows
	}
	for _, c := range cases {
		var r Rupiah
		if err := r.Scan(c.src); err != nil || r != c.want {
			t.Errorf("Scan(%#v) = %d, %v; want %d", c.src, r, err, c.want)
		}
	}
}

func TestRupiah_ScanRejectsFractionsAndJunk(t *testing.T) {
	for _, src := range []any{[]byte("50000.50"), "0.01", float64(10.5), []byte("abc"), []byte(""), true} {
		var r Rupiah
		if err := r.Scan(src); err == nil {
			t.Errorf("Scan(%#v) must fail, got %d", src, r)
		}
	}
}

func TestRupiah_Value(t *testing.T) {
	v, err := Rupiah(64000).Value()
	if err != nil || v != int64(64000) {
		t.Errorf("Value() = %#v, %v", v, err)
	}
}

func TestRupiah_JSON(t *testing.T) {
	b, _ := json.Marshal(struct {
		P Rupiah  `json:"p"`
		N *Rupiah `json:"n"`
	}{P: 64000})
	if string(b) != `{"p":64000,"n":null}` {
		t.Errorf("marshal = %s", b)
	}
	var in struct{ P Rupiah }
	if err := json.Unmarshal([]byte(`{"P":15000}`), &in); err != nil || in.P != 15000 {
		t.Errorf("unmarshal whole number: %d, %v", in.P, err)
	}
	if err := json.Unmarshal([]byte(`{"P":15000.5}`), &in); err == nil {
		t.Error("a fractional rupiah in a request must be rejected")
	}
}

func TestPercentOf_RoundsHalfUp(t *testing.T) {
	cases := []struct {
		amount Rupiah
		pct    float64
		want   Rupiah
	}{
		{10, 15, 2}, {30, 25, 8}, {3333, 15, 500}, {10001, 15, 1500},
		{33333, 15, 5000}, {66667, 15, 10000}, {1, 50, 1}, {1, 49, 0},
		{100000, 12.5, 12500}, {0, 20, 0},
	}
	for _, c := range cases {
		if got := PercentOf(c.amount, c.pct); got != c.want {
			t.Errorf("PercentOf(%d, %v) = %d, want %d", c.amount, c.pct, got, c.want)
		}
	}
}

func TestDivRound_HalfUp(t *testing.T) {
	cases := []struct {
		total Rupiah
		n     int64
		want  Rupiah
	}{
		{2018000, 23, 87739}, // 87739.13
		{10, 4, 3},           // 2.5 -> 3
		{9, 4, 2},            // 2.25
		{0, 5, 0},
		{100, 0, 0}, // no transactions
	}
	for _, c := range cases {
		if got := DivRound(c.total, c.n); got != c.want {
			t.Errorf("DivRound(%d, %d) = %d, want %d", c.total, c.n, got, c.want)
		}
	}
}

func TestPerHour(t *testing.T) {
	cases := []struct {
		perHour Rupiah
		minutes int
		want    Rupiah
	}{
		{15000, 60, 15000}, {15000, 30, 7500}, {15000, 90, 22500},
		{25000, 20, 8333},  // 8333.33
		{25000, 40, 16667}, // 16666.67
		{30000, 240, 120000}, {1, 30, 1}, {15000, 0, 0},
	}
	for _, c := range cases {
		if got := PerHour(c.perHour, c.minutes); got != c.want {
			t.Errorf("PerHour(%d, %d) = %d, want %d", c.perHour, c.minutes, got, c.want)
		}
	}
}
