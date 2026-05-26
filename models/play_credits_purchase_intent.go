package models

import "time"

// PlayCreditsPurchaseIntent melacak pembelian credits yang sedang diproses.
// Dibuat saat customer klik "Bayar", dikonfirmasi setelah Xendit webhook masuk.
// Status: pending → paid (sukses) / expired / failed.
type PlayCreditsPurchaseIntent struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	CustomerID       string     `gorm:"type:char(36);not null;index" json:"customer_id"`
	StoreID          string     `gorm:"type:char(36);not null" json:"store_id"`
	PackageID        string     `gorm:"type:char(36);not null" json:"package_id"`
	Amount           float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status           string     `gorm:"type:enum('pending','paid','expired','failed');default:'pending'" json:"status"`
	XenditInvoiceID  *string    `gorm:"size:255" json:"xendit_invoice_id"`
	XenditInvoiceURL *string    `gorm:"size:500" json:"xendit_invoice_url"`
	ExpiresAt        time.Time  `json:"expires_at"`
	PaidAt           *time.Time `json:"paid_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	Customer Customer           `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Store    Store              `gorm:"foreignKey:StoreID"    json:"store,omitempty"`
	Package  PlayCreditsPackage `gorm:"foreignKey:PackageID"  json:"package,omitempty"`
}
