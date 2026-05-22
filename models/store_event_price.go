package models

import "time"

// StoreEventPrice menyimpan harga event per hari untuk setiap store.
// Dipakai untuk kalkulasi proporsional: total = (price_per_day / 24) × duration_hours.
type StoreEventPrice struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	StoreID     string    `gorm:"type:char(36);not null;uniqueIndex" json:"store_id"`
	PricePerDay float64   `gorm:"type:decimal(12,2);not null;default:0" json:"price_per_day"`
	CreatedBy   *string   `gorm:"size:255" json:"created_by"`
	UpdatedBy   *string   `gorm:"size:255" json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}
