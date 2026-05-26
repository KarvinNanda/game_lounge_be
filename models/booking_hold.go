package models

import "time"

// BookingHold adalah reservasi sementara (15 menit) saat customer memulai pembayaran.
// Hold diciptakan sebelum Xendit invoice dikonfirmasi.
// Setelah pembayaran sukses (webhook PAID), hold dikonversi menjadi Booking permanen.
type BookingHold struct {
	ID               string    `gorm:"type:char(36);primaryKey" json:"id"`
	CustomerID       string    `gorm:"type:char(36);not null" json:"customer_id"`
	StoreID          string    `gorm:"type:char(36);not null" json:"store_id"`
	RoomID           string    `gorm:"type:char(36);not null" json:"room_id"`
	RoomTemplateID   uint      `gorm:"not null" json:"room_template_id"`
	BookingDate      time.Time `gorm:"type:date;not null" json:"booking_date"`
	StartTime        string    `gorm:"type:time;not null" json:"start_time"`
	EndTime          string    `gorm:"type:time;not null" json:"end_time"`
	DurationHours    float64   `gorm:"type:decimal(4,1)" json:"duration_hours"`
	BasePrice        float64   `gorm:"type:decimal(12,2)" json:"base_price"`
	TotalPrice       float64   `gorm:"type:decimal(12,2)" json:"total_price"`
	PriceBreakdown   string    `gorm:"type:json" json:"price_breakdown"`
	PaymentMethod    string    `gorm:"size:50;default:'xendit'" json:"payment_method"`
	XenditInvoiceID  *string   `gorm:"size:255" json:"xendit_invoice_id"`
	XenditInvoiceURL *string   `gorm:"size:500" json:"xendit_invoice_url"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`

	Customer     Customer     `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Store        Store        `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Room         StoreRoom    `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	RoomTemplate RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}

// IsExpired mengembalikan true jika hold sudah melewati ExpiresAt.
func (b *BookingHold) IsExpired() bool {
	return time.Now().After(b.ExpiresAt)
}
