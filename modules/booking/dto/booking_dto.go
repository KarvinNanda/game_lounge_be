package dto

// CreateBookingRequest payload untuk membuat booking baru.
type CreateBookingRequest struct {
	StoreID          string  `json:"store_id" binding:"required"`
	RoomID           string  `json:"room_id" binding:"required"`
	CustomerID       string  `json:"customer_id"`                           // kosong = walk-in anonymous
	CustomerName     string  `json:"customer_name" binding:"required,min=2"`
	CustomerWhatsapp string  `json:"customer_whatsapp"`
	CustomerEmail    string  `json:"customer_email"`
	BookingDate      string  `json:"booking_date" binding:"required"` // YYYY-MM-DD
	StartTime        string  `json:"start_time" binding:"required"`   // HH:MM
	EndTime          string  `json:"end_time" binding:"required"`     // HH:MM
	DurationHours    float64 `json:"duration_hours" binding:"required,min=0.5"`
	PaymentMethod    string  `json:"payment_method"`   // 'cash' | 'play_credits'
	PlayCreditID     string  `json:"play_credit_id"`   // wajib jika payment_method = 'play_credits'
	VoucherCode      string  `json:"voucher_code"`     // opsional
	Notes            string  `json:"notes"`
}

// CancelBookingRequest payload untuk membatalkan booking.
type CancelBookingRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

// BookingFilter parameter filter untuk list booking.
type BookingFilter struct {
	StoreID  string `form:"store_id"`
	RoomID   string `form:"room_id"`
	Status   string `form:"status"`
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
	Search   string `form:"search"`
	Page     int    `form:"page,default=1"`
	PerPage  int    `form:"per_page,default=20"`
}

// DashboardFilter parameter untuk kalender grid.
type DashboardFilter struct {
	StoreID string `form:"store_id" binding:"required"`
	Date    string `form:"date" binding:"required"` // YYYY-MM-DD
	RoomID  string `form:"room_id"`
}
