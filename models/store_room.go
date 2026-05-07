package models

import "time"

type StoreRoom struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	StoreID        string     `gorm:"type:char(36);not null;index" json:"store_id"`
	RoomTemplateID uint       `gorm:"not null" json:"room_template_id"`
	UnitNumber     uint       `gorm:"not null" json:"unit_number"`
	Name           string     `gorm:"size:150;not null" json:"name"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	CreatedBy      *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy      *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy      *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at"`

	RoomTemplate RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}
