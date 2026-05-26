package utils

import (
	"bytes"
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
}

type XenditInvoiceResponse struct {
	ID         string `json:"id"`
	InvoiceURL string `json:"invoice_url"`
	Status     string `json:"status"`
}

// CreateXenditInvoice membuat invoice di Xendit.
// MOCK MODE: jika XENDIT_SECRET_KEY kosong, return dummy URL untuk testing.
func CreateXenditInvoice(req XenditInvoiceRequest) (*XenditInvoiceResponse, error) {
	secretKey := os.Getenv("XENDIT_SECRET_KEY")

	// ── MOCK MODE (belum punya API key) ──────────────────────────────────────
	if secretKey == "" {
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
	httpReq.SetBasicAuth(secretKey, "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("xendit request failed: %w", err)
	}
	defer resp.Body.Close()

	var result XenditInvoiceResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

// VerifyXenditWebhook memvalidasi X-CALLBACK-TOKEN dari Xendit.
func VerifyXenditWebhook(token string) bool {
	expected := os.Getenv("XENDIT_WEBHOOK_TOKEN")
	if expected == "" {
		return true // mock mode: skip verification
	}
	return token == expected
}
