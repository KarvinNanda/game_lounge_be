package utils

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type XenditInvoiceRequest struct {
	ExternalID  string  `json:"external_id"`
	Amount      float64 `json:"amount"`
	PayerEmail  string  `json:"payer_email"`
	Description string  `json:"description"`
	SuccessURL  string  `json:"success_redirect_url"`
	FailureURL  string  `json:"failure_redirect_url"`
	// InvoiceDuration (detik). Tanpa ini Xendit memakai default 24 jam, sehingga
	// customer bisa membayar setelah hold/intent di sisi kita sudah habis.
	InvoiceDuration int `json:"invoice_duration,omitempty"`
}

type XenditInvoiceResponse struct {
	ID         string `json:"id"`
	InvoiceURL string `json:"invoice_url"`
	Status     string `json:"status"`
}

// CreateXenditInvoice membuat invoice di Xendit.
// MOCK MODE: lihat XenditMockMode — return dummy URL untuk testing.
func CreateXenditInvoice(req XenditInvoiceRequest) (*XenditInvoiceResponse, error) {
	// ── MOCK MODE (dev tanpa API key) ────────────────────────────────────────
	if XenditMockMode() {
		mockID := fmt.Sprintf("mock-invoice-%s-%d", req.ExternalID, time.Now().Unix())
		mockURL := fmt.Sprintf("%s/payment/mock?invoice_id=%s&amount=%.0f",
			os.Getenv("APP_URL"), mockID, req.Amount)
		return &XenditInvoiceResponse{
			ID:         mockID,
			InvoiceURL: mockURL,
			Status:     "PENDING",
		}, nil
	}

	// ── REAL XENDIT (setelah API key tersedia) ────────────────────────────────
	payload, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "https://api.xendit.co/v2/invoices", bytes.NewBuffer(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(os.Getenv("XENDIT_SECRET_KEY"), "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("xendit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xendit invoice gagal: HTTP %d", resp.StatusCode)
	}
	var result XenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("xendit response tidak valid: %w", err)
	}
	if result.ID == "" || result.InvoiceURL == "" {
		return nil, fmt.Errorf("xendit response tanpa invoice id/url")
	}
	return &result, nil
}

// VerifyXenditWebhook memvalidasi X-CALLBACK-TOKEN dari Xendit.
// Menggunakan constant-time compare untuk mencegah timing attack.
// Verifikasi hanya di-skip pada mock mode penuh (kedua env Xendit kosong).
func VerifyXenditWebhook(token string) bool {
	expected := os.Getenv("XENDIT_WEBHOOK_TOKEN")
	if expected == "" {
		// Tanpa token hanya diterima di mock mode (dev, key kosong). Jika key ada
		// tapi token lupa diset, atau di production → TOLAK (fail-closed).
		return XenditMockMode()
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}
