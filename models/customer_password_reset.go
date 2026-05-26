package models

import "time"

// CustomerPasswordReset menyimpan token reset password untuk customer.
// Token 32-byte random hex, expire 15 menit, one-time use.
// ip_address dipakai untuk rate limiting (max 3 request/jam/IP).
type CustomerPasswordReset struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	CustomerID string     `gorm:"type:char(36);not null;index" json:"customer_id"`
	Token      string     `gorm:"size:64;uniqueIndex;not null" json:"token"`
	IPAddress  string     `gorm:"size:45;not null" json:"ip_address"`
	ExpiresAt  time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt     *time.Time `json:"used_at"`
	CreatedAt  time.Time  `json:"created_at"`

	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}
