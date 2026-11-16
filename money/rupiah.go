package money

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Rupiah is an amount of money in whole rupiah. IDR has no minor unit in practice,
// so amounts are integers and money math never goes through float64.
//
// The DB columns stay decimal(12,2). go-sql-driver returns those as text such as
// "50000.00", which a plain int64 cannot Scan, so Rupiah implements sql.Scanner.
// A value with a non-zero fraction is an error: it means bad data, not something
// to round silently.
type Rupiah int64

func (r *Rupiah) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*r = 0 // e.g. SUM() over no rows
		return nil
	case int64:
		*r = Rupiah(v)
		return nil
	case float64:
		if v != math.Trunc(v) {
			return fmt.Errorf("rupiah: %v has a fraction", v)
		}
		*r = Rupiah(v)
		return nil
	case []byte:
		return r.parse(string(v))
	case string:
		return r.parse(v)
	}
	return fmt.Errorf("rupiah: cannot scan %T", src)
}

func (r *Rupiah) parse(s string) error {
	whole, frac, _ := strings.Cut(s, ".")
	if strings.Trim(frac, "0") != "" {
		return fmt.Errorf("rupiah: %q has a fraction", s)
	}
	n, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return fmt.Errorf("rupiah: %q: %w", s, err)
	}
	*r = Rupiah(n)
	return nil
}

func (r Rupiah) Value() (driver.Value, error) {
	return int64(r), nil
}

// PercentOf returns pct percent of amount, rounded half up to whole rupiah.
// pct may have up to 2 decimals (12.5 means 12.5%). Integer math only:
// amount * basis points / 10000.
func PercentOf(amount Rupiah, pct float64) Rupiah {
	bp := int64(math.Round(pct * 100))
	return Rupiah(divRoundHalfUp(int64(amount)*bp, 10000))
}

// DivRound divides total by n, rounded half up. n <= 0 returns 0.
func DivRound(total Rupiah, n int64) Rupiah {
	if n <= 0 {
		return 0
	}
	return Rupiah(divRoundHalfUp(int64(total), n))
}

// divRoundHalfUp rounds a/b to the nearest integer, halves away from zero (same as
// math.Round, which the float64 code used). b must be positive.
func divRoundHalfUp(a, b int64) int64 {
	if a < 0 {
		return -divRoundHalfUp(-a, b)
	}
	return (a + b/2) / b
}

// PerHour prices a duration in minutes at an hourly rate, rounded half up to
// whole rupiah. Each segment (happy hour, flash sale) is rounded on its own.
func PerHour(perHour Rupiah, minutes int) Rupiah {
	return Rupiah(divRoundHalfUp(int64(perHour)*int64(minutes), 60))
}

// Thousands writes the amount with Indonesian thousand separators: 31000 -> "31.000".
// For templates that already print "Rp" themselves.
func Thousands(r Rupiah) string {
	n := int64(r)
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, d := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	return sign + b.String()
}

// Format writes an amount for people: 31000 -> "Rp 31.000", -1500 -> "-Rp 1.500".
func Format(r Rupiah) string {
	if r < 0 {
		return "-Rp " + Thousands(-r)
	}
	return "Rp " + Thousands(r)
}
