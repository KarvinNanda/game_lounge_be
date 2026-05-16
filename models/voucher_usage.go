package models

import "time"

// VoucherUsage mencatat pemakaian voucher per customer.
// UNIQUE (voucher_id, customer_id) → 1x per user.
// booking_id akan diisi saat modul Booking dibuat.
type VoucherUsage struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	VoucherID      string    `gorm:"type:char(36);not null;index" json:"voucher_id"`
	CustomerID     string    `gorm:"type:char(36);not null;index" json:"customer_id"`
	BookingID      *string   `gorm:"type:char(36)" json:"booking_id"`
	DiscountAmount float64   `gorm:"type:decimal(12,2);default:0" json:"discount_amount"`
	UsedAt         time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"used_at"`

	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}
