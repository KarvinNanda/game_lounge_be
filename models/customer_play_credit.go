package models

import "time"

// CustomerPlayCredit adalah credits yang dimiliki customer per pembelian.
// remaining_hours berkurang saat booking pakai credits atau admin adjust.
// expires_at = purchased_at + validity_days dari package.
type CustomerPlayCredit struct {
	ID              string     `gorm:"type:char(36);primaryKey" json:"id"`
	CustomerID      string     `gorm:"type:char(36);not null;index" json:"customer_id"`
	PackageID       string     `gorm:"type:char(36);not null;index" json:"package_id"`
	TotalHours      float64    `gorm:"type:decimal(8,1);not null" json:"total_hours"`
	RemainingHours  float64    `gorm:"type:decimal(8,1);not null" json:"remaining_hours"`
	PurchasedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"purchased_at"`
	ExpiresAt       time.Time  `gorm:"not null" json:"expires_at"`
	PaymentMethod   string     `gorm:"type:enum('manual','xendit');default:'manual'" json:"payment_method"`
	PaymentAmount   *float64   `gorm:"type:decimal(12,2)" json:"payment_amount"`
	XenditInvoiceID *string    `gorm:"size:255" json:"xendit_invoice_id"`
	Notes           *string    `gorm:"type:text" json:"notes"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedBy       *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy       *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy       *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"deleted_at"`

	Customer Customer           `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Package  PlayCreditsPackage `gorm:"foreignKey:PackageID" json:"package,omitempty"`
}
