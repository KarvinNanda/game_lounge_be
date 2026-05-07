package models

import "time"

type StaffStore struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	StaffID   string    `gorm:"type:char(36);not null;index" json:"staff_id"`
	StoreID   string    `gorm:"type:char(36);not null;index" json:"store_id"`
	CreatedAt time.Time `json:"created_at"`

	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}
