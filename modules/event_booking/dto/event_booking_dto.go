package dto

type CreateEventBookingRequest struct {
	StoreID          string `json:"store_id" binding:"required"`
	EventName        string `json:"event_name" binding:"required,min=2,max=150"`
	Description      string `json:"description"`                     // opsional — tampil ke customer
	CustomerName     string `json:"customer_name" binding:"required,min=2"`
	CustomerWhatsapp string `json:"customer_whatsapp"`
	CustomerEmail    string `json:"customer_email"`
	BookingDate      string `json:"booking_date" binding:"required"` // YYYY-MM-DD
	StartTime        string `json:"start_time"`                      // HH:MM — wajib jika duration_type = hourly
	EndTime          string `json:"end_time"`                        // HH:MM — wajib jika duration_type = hourly
	Notes            string `json:"notes"`

	// Opsi baru
	DurationType            string `json:"duration_type" binding:"required"` // "hourly" | "full_day"
	BookingScope            string `json:"booking_scope" binding:"required"` // "full_venue" | "per_room_type"
	SelectedRoomTemplateIDs []uint `json:"selected_room_template_ids"`       // untuk per_room_type
}

type CancelEventBookingRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

type EventBookingFilter struct {
	StoreID  string `form:"store_id"`
	Status   string `form:"status"`
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
	Page     int    `form:"page,default=1"`
	PerPage  int    `form:"per_page,default=20"`
}
