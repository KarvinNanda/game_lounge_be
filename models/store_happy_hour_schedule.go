package models

import "time"

// StoreHappyHourSchedule menyimpan rentang jam happy hour per store.
// Contoh: 10:00-15:00 dan 22:00-02:00.
// end_time boleh lebih kecil dari start_time (lintas tengah malam).
type StoreHappyHourSchedule struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	StoreID   string     `gorm:"type:char(36);not null;index" json:"store_id"`
	StartTime string     `gorm:"type:time;not null" json:"start_time"`
	EndTime   string     `gorm:"type:time;not null" json:"end_time"`
	CreatedBy *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at"`
}
