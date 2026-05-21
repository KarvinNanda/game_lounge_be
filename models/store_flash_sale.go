package models

import "time"

// StoreFlashSale adalah flash sale per store per room template per rentang tanggal & jam.
// PricePerHour = harga per jam saat flash sale aktif (seperti happy hour tapi date-gated).
// Prioritas tertinggi dalam kalkulasi harga (di atas happy hour & normal).
type StoreFlashSale struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	StoreID        string     `gorm:"type:char(36);not null;index" json:"store_id"`
	RoomTemplateID uint       `gorm:"not null" json:"room_template_id"`
	Name           string     `gorm:"size:150;not null" json:"name"`
	Description    *string    `gorm:"type:text" json:"description"`
	PricePerHour   float64    `gorm:"type:decimal(12,2);not null;default:0" json:"price_per_hour"`
	DateFrom       time.Time  `gorm:"type:date;not null" json:"date_from"`
	DateTo         time.Time  `gorm:"type:date;not null" json:"date_to"`
	TimeFrom       string     `gorm:"type:time;not null" json:"time_from"`
	TimeTo         string     `gorm:"type:time;not null" json:"time_to"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	CreatedBy      *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy      *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy      *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at"`

	RoomTemplate RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}
