package utils

import (
	"fmt"
	"html"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// ── SMTP core ─────────────────────────────────────────────────────────────────

type smtpConfig struct {
	Host       string
	FromEmail  string
	Password   string
	SenderName string
	Port       int
}

func loadSMTP() smtpConfig {
	cfg := smtpConfig{
		Host:       os.Getenv("SMTP_HOST"),
		FromEmail:  os.Getenv("SMTP_FROM"),
		Password:   os.Getenv("SMTP_PASSWORD"),
		SenderName: os.Getenv("SMTP_SENDER_NAME"),
	}
	if port, _ := strconv.Atoi(os.Getenv("SMTP_PORT")); port > 0 {
		cfg.Port = port
	} else {
		cfg.Port = 587
	}
	if cfg.SenderName == "" {
		// Ambil bagian sebelum @ sebagai display name (bsena692@gmail.com → bsena692)
		parts := strings.Split(cfg.FromEmail, "@")
		cfg.SenderName = parts[0]
	}
	return cfg
}

// dispatch mengirim raw HTML via SMTP — dipakai oleh SendEmail & SendHTMLEmail.
func dispatch(toEmail, toName, subject, htmlContent string) error {
	cfg := loadSMTP()
	auth := smtp.PlainAuth("", cfg.FromEmail, cfg.Password, cfg.Host)
	msg := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s <%s>\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		cfg.SenderName, cfg.FromEmail,
		toName, toEmail,
		subject,
		htmlContent,
	)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return smtp.SendMail(addr, auth, cfg.FromEmail, []string{toEmail}, []byte(msg))
}

// ── Public API ────────────────────────────────────────────────────────────────

// SendEmail menerima plain text dari template DB, konversi \n → <br>,
// lalu bungkus dalam card HTML generik sebelum dikirim.
func SendEmail(toEmail, toName, subject, body string) error {
	cfg := loadSMTP()
	htmlBody := strings.ReplaceAll(html.EscapeString(body), "\n", "<br>")
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f5f3ff;font-family:Arial,Helvetica,sans-serif">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f3ff;padding:32px 16px">
  <tr><td align="center">
  <table width="520" cellpadding="0" cellspacing="0" style="max-width:520px;background:white;border-radius:14px;overflow:hidden;box-shadow:0 4px 20px rgba(124,58,237,0.12)">
    <tr><td style="background:#7c3aed;padding:20px;text-align:center">
      <h2 style="margin:0;color:white;font-size:18px;letter-spacing:1px">🎮 QUANTUM GAMING CENTER</h2>
      <p style="margin:4px 0 0;color:#c4b5fd;font-size:12px">PlayStation Rental</p>
    </td></tr>
    <tr><td style="padding:28px 32px;font-size:14px;color:#374151;line-height:1.8">%s</td></tr>
    <tr><td style="padding:14px;text-align:center;border-top:1px solid #f3f4f6">
      <p style="margin:0;color:#d1d5db;font-size:11px">© %s</p>
    </td></tr>
  </table>
  </td></tr>
</table>
</body></html>`, htmlBody, cfg.SenderName)
	return dispatch(toEmail, toName, subject, htmlContent)
}

// SendHTMLEmail mengirim pre-built HTML langsung tanpa transformasi apapun.
// Dipakai oleh fungsi template hardcode (booking, voucher).
func SendHTMLEmail(toEmail, toName, subject, htmlContent string) error {
	return dispatch(toEmail, toName, subject, htmlContent)
}

// SendCustomerPasswordEmail fallback jika template DB belum tersedia.
func SendCustomerPasswordEmail(toEmail, customerName, password string) error {
	subject := "Selamat Datang di Quantum Gaming Center – Akun Member Anda"
	body := fmt.Sprintf(
		"Halo %s,\n\nAkun member Anda telah berhasil dibuat.\n\nEmail    : %s\nPassword : %s\n\nSilakan gunakan password di atas untuk login.\nDemi keamanan, segera ganti password setelah login pertama.\n\nSalam,\nTim Quantum Gaming Center",
		customerName, toEmail, password,
	)
	return SendEmail(toEmail, customerName, subject, body)
}

// BuildWelcomeEmailBody dipertahankan untuk backward compatibility.
func BuildWelcomeEmailBody(name, password string) string {
	return fmt.Sprintf("Halo %s!\n\nPassword kamu: %s\n\nSalam,\nTim Quantum Gaming Center", name, password)
}
