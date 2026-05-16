package models

import "time"

// Booking adalah satu sesi bermain customer di sebuah ruangan.
// Tidak ada soft delete — booking adalah catatan keuangan permanen.
// Status dihitung otomatis berdasarkan waktu WIB saat ini.
type Booking struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	BookingCode      string     `gorm:"size:20;uniqueIndex;not null" json:"booking_code"`
	StoreID          string     `gorm:"type:char(36);not null;index" json:"store_id"`
	RoomID           string     `gorm:"type:char(36);not null;index" json:"room_id"`
	CustomerID       *string    `gorm:"type:char(36);index" json:"customer_id"`
	CustomerName     string     `gorm:"size:150;not null" json:"customer_name"`
	CustomerWhatsapp *string    `gorm:"size:20" json:"customer_whatsapp"`
	CustomerEmail    *string    `gorm:"size:150" json:"customer_email"`
	BookingDate      time.Time  `gorm:"type:date;not null" json:"booking_date"`
	StartTime        string     `gorm:"type:time;not null" json:"start_time"`
	EndTime          string     `gorm:"type:time;not null" json:"end_time"`
	DurationHours    float64    `gorm:"type:decimal(4,1);not null" json:"duration_hours"`
	PriceBreakdown   *string    `gorm:"type:json" json:"price_breakdown"`
	BasePrice        float64    `gorm:"type:decimal(12,2);not null" json:"base_price"`
	DiscountAmount   float64    `gorm:"type:decimal(12,2);default:0" json:"discount_amount"`
	TotalPrice       float64    `gorm:"type:decimal(12,2);not null" json:"total_price"`
	PaymentMethod    string     `gorm:"type:enum('cash','play_credits');default:'cash'" json:"payment_method"`
	PlayCreditID     *string    `gorm:"type:char(36)" json:"play_credit_id"`
	VoucherID        *string    `gorm:"type:char(36)" json:"voucher_id"`
	VoucherCode      *string    `gorm:"size:50" json:"voucher_code"`
	Status           string     `gorm:"type:enum('upcoming','ongoing','completed','cancelled');default:'upcoming'" json:"status"`
	CancelReason     *string    `gorm:"type:text" json:"cancel_reason"`
	CancelledAt      *time.Time `json:"cancelled_at"`
	CancelledBy      *string    `gorm:"size:255" json:"cancelled_by"`
	Notes            *string    `gorm:"type:text" json:"notes"`
	CreatedBy        *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy        *string    `gorm:"size:255" json:"updated_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	Store    Store     `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Room     StoreRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Customer *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}
