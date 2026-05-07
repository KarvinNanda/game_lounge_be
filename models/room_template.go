package models

import "time"

type RoomTemplate struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	CapacityMin uint       `gorm:"default:1" json:"capacity_min"`
	CapacityMax uint       `gorm:"not null" json:"capacity_max"`
	Description *string    `gorm:"type:text" json:"description"`
	ImageURL    *string    `gorm:"size:255" json:"image_url"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	CreatedBy   *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy   *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy   *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at"`

	Facilities []Facility `gorm:"many2many:room_template_facilities;" json:"facilities,omitempty"`
}
