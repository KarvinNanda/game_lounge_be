package utils

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
)

// smtpConfig mengambil konfigurasi SMTP dari environment variables.
func smtpConfig() (host, from, password string, port int, err error) {
	host = os.Getenv("SMTP_HOST")
	from = os.Getenv("SMTP_FROM")
	password = os.Getenv("SMTP_PASSWORD")
	portStr := os.Getenv("SMTP_PORT")

	if host == "" || from == "" {
		err = fmt.Errorf("konfigurasi SMTP belum diset (SMTP_HOST / SMTP_FROM)")
		return
	}

	port = 587
	if p, e := strconv.Atoi(portStr); e == nil && p > 0 {
		port = p
	}
	return
}

// SendEmail mengirim email generik dengan dukungan HTML body.
// contentType: "text/plain" atau "text/html"
func SendEmail(toEmail, toName, subject, body string) error {
	host, from, smtpPassword, port, err := smtpConfig()
	if err != nil {
		return err
	}

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, toEmail, subject, body,
	))

	addr := fmt.Sprintf("%s:%d", host, port)
	var auth smtp.Auth
	if smtpPassword != "" {
		auth = smtp.PlainAuth("", from, smtpPassword, host)
	}

	return smtp.SendMail(addr, auth, from, []string{toEmail}, msg)
}

// SendCustomerPasswordEmail mengirim password awal customer via SMTP.
func SendCustomerPasswordEmail(toEmail, customerName, password string) error {
	subject := "Selamat Datang di Quantum Gaming Center – Akun Member Anda"
	body := fmt.Sprintf(`Halo %s,

Akun member Anda di Game Lounge telah berhasil dibuat.

Email    : %s
Password : %s

Silakan gunakan password di atas untuk login ke aplikasi kami.
Demi keamanan, kami sarankan Anda mengganti password setelah login pertama.

Salam,
Tim Game Lounge
`, customerName, toEmail, password)

	return SendEmail(toEmail, customerName, subject, body)
}
