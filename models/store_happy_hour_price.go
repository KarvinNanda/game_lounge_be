package models

import "time"

// StoreHappyHourPrice menyimpan harga per jam saat happy hour,
// per store per room template. UNIQUE (store_id, room_template_id).
type StoreHappyHourPrice struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	StoreID        string     `gorm:"type:char(36);not null;index" json:"store_id"`
	RoomTemplateID uint       `gorm:"not null" json:"room_template_id"`
	PricePerHour   float64    `gorm:"type:decimal(12,2);default:0" json:"price_per_hour"`
	CreatedBy      *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy      *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy      *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at"`

	RoomTemplate RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}
