package models

import (
	"time"

	"gorm.io/gorm"

	"game_lounge_be/money"
)

// VoucherUsage mencatat pemakaian voucher per customer.
// UNIQUE (voucher_id, customer_id) → 1x per user.
// booking_id akan diisi saat modul Booking dibuat.
type VoucherUsage struct {
	ID             uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	VoucherID      string       `gorm:"type:char(36);not null;index" json:"voucher_id"`
	CustomerID     string       `gorm:"type:char(36);not null;index" json:"customer_id"`
	BookingID      *string      `gorm:"type:char(36)" json:"booking_id"`
	DiscountAmount money.Rupiah `gorm:"type:decimal(12,2);default:0" json:"discount_amount"`
	UsedAt         time.Time    `gorm:"default:CURRENT_TIMESTAMP(3)" json:"used_at"`

	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

// BeforeCreate fills used_at from the app clock. Without it GORM leaves the
// column to the DB default CURRENT_TIMESTAMP, which uses the DB server timezone
// while every other timestamp uses the app's (loc=Local in the DSN).
func (u *VoucherUsage) BeforeCreate(tx *gorm.DB) error {
	if u.UsedAt.IsZero() {
		u.UsedAt = time.Now()
	}
	return nil
}
