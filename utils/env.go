package utils

import "os"

// IsProduction bernilai true jika APP_ENV=production.
func IsProduction() bool {
	return os.Getenv("APP_ENV") == "production"
}

// XenditMockMode: invoice palsu + endpoint mock-confirm aktif.
// Hanya jika key Xendit kosong DAN bukan production — supaya deploy production
// yang lupa mengisi key tidak berubah menjadi "bayar gratis".
func XenditMockMode() bool {
	return os.Getenv("XENDIT_SECRET_KEY") == "" && !IsProduction()
}

// ProductionConfigErrors mengembalikan daftar config wajib yang salah saat
// APP_ENV=production. Server harus menolak start jika daftar ini tidak kosong,
// karena setiap item di bawah membuat sistem fail-open.
func ProductionConfigErrors() []string {
	if !IsProduction() {
		return nil
	}
	var errs []string
	if os.Getenv("ALLOWED_ORIGINS") == "" {
		errs = append(errs, "ALLOWED_ORIGINS kosong: CORS akan menerima origin apa pun dengan credentials")
	}
	if os.Getenv("XENDIT_SECRET_KEY") == "" {
		errs = append(errs, "XENDIT_SECRET_KEY kosong")
	}
	if os.Getenv("XENDIT_WEBHOOK_TOKEN") == "" {
		errs = append(errs, "XENDIT_WEBHOOK_TOKEN kosong: webhook tidak bisa diverifikasi")
	}
	if os.Getenv("COOKIE_SECURE") == "false" {
		errs = append(errs, "COOKIE_SECURE=false: auth cookie bisa terkirim lewat HTTP")
	}
	return errs
}
