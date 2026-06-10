package models

import "time"

// EventBooking adalah booking seluruh gedung (1 store) untuk sebuah event.
// Tidak ada room_id karena memblokir SEMUA ruangan di store tersebut.
// total_price dihitung: (price_per_day / 24) × duration_hours, dibulatkan ke Rp 1.000.
// ID dibuat via uuid.NewString() di service layer — tidak pakai default DB.
type EventBooking struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	StoreID          string     `gorm:"type:char(36);not null;index" json:"store_id"`
	EventName        string     `gorm:"size:150;not null" json:"event_name"`
	Description      *string    `gorm:"type:text" json:"description"` // deskripsi event (tampil ke customer)
	CustomerName     string     `gorm:"size:150;not null" json:"customer_name"`
	CustomerWhatsapp *string    `gorm:"size:20" json:"customer_whatsapp"`
	CustomerEmail    *string    `gorm:"size:150" json:"customer_email"`
	BookingDate      time.Time  `gorm:"type:date;not null" json:"booking_date"`
	StartTime        string     `gorm:"type:time;not null" json:"start_time"`
	EndTime          string     `gorm:"type:time;not null" json:"end_time"`
	DurationHours    float64    `gorm:"type:decimal(4,1);not null" json:"duration_hours"`
	PricePerDay      float64    `gorm:"type:decimal(12,2);not null" json:"price_per_day"`
	TotalPrice       float64    `gorm:"type:decimal(12,2);not null" json:"total_price"`
	Status           string     `gorm:"type:enum('upcoming','ongoing','completed','cancelled');default:'upcoming'" json:"status"`

	// Opsi durasi dan scope ruangan (admin create event booking)
	DurationType            string  `gorm:"type:enum('hourly','full_day');default:'hourly'" json:"duration_type"`
	BookingScope            string  `gorm:"type:enum('full_venue','per_room_type');default:'full_venue'" json:"booking_scope"`
	SelectedRoomTemplateIDs *string `gorm:"type:text" json:"selected_room_template_ids"` // JSON "[1,2,3]" atau NULL

	CancelReason     *string    `gorm:"type:text" json:"cancel_reason"`
	CancelledAt      *time.Time `json:"cancelled_at"`
	CancelledBy      *string    `gorm:"size:255" json:"cancelled_by"`
	Notes            *string    `gorm:"type:text" json:"notes"`
	CreatedBy        *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy        *string    `gorm:"size:255" json:"updated_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Field untuk customer web booking (nullable — admin booking tidak pakai ini)
	CustomerID       *string    `gorm:"type:char(36);index" json:"customer_id"`
	IsCustomerBooking bool      `gorm:"default:false" json:"is_customer_booking"`
	PaymentStatus    *string    `gorm:"type:enum('pending_payment','paid','failed')" json:"payment_status"`
	PaymentMethod    *string    `gorm:"size:50" json:"payment_method"`
	XenditInvoiceID  *string    `gorm:"size:255" json:"xendit_invoice_id"`
	XenditInvoiceURL *string    `gorm:"size:500" json:"xendit_invoice_url"`

	Store    Store     `gorm:"foreignKey:StoreID"    json:"store,omitempty"`
	Customer *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}
