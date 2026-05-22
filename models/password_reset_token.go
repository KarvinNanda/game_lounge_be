package models

import "time"

// PasswordResetToken menyimpan token reset password untuk super admin.
// Token 32-byte random hex, expire 15 menit, one-time use.
// ip_address dipakai untuk rate limiting (max 3 request/jam/IP).
type PasswordResetToken struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	StaffID   string     `gorm:"type:char(36);not null;index" json:"staff_id"`
	Token     string     `gorm:"size:64;uniqueIndex;not null" json:"token"`
	IPAddress string     `gorm:"size:45;not null" json:"ip_address"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`

	Staff Staff `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}
