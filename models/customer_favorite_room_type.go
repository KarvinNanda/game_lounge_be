package models

import "time"

// CustomerFavoriteRoomType menyimpan relasi customer ↔ room template favorit.
type CustomerFavoriteRoomType struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CustomerID     string    `gorm:"type:char(36);not null;index" json:"customer_id"`
	RoomTemplateID uint      `gorm:"not null" json:"room_template_id"`
	CreatedAt      time.Time `json:"created_at"`

	// Relasi
	RoomTemplate RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}
