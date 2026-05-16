package models

import "time"

// StorePackagePrice menyimpan harga paket normal per store per room template.
// duration_hours: standar = 1,3,5,8,10. is_custom=true untuk paket tambahan admin.
// UNIQUE (store_id, room_template_id, duration_hours).
type StorePackagePrice struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	StoreID        string     `gorm:"type:char(36);not null;index" json:"store_id"`
	RoomTemplateID uint       `gorm:"not null" json:"room_template_id"`
	DurationHours  uint       `gorm:"not null" json:"duration_hours"`
	Price          float64    `gorm:"type:decimal(12,2);default:0" json:"price"`
	IsCustom       bool       `gorm:"default:false" json:"is_custom"`
	CreatedBy      *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy      *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy      *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at"`

	RoomTemplate RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}
