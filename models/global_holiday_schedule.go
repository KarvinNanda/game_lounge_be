package models

import "time"

// GlobalHolidaySchedule adalah tanggal merah yang berlaku untuk SEMUA cabang.
// Jika tanggal ada di sini → semua store ikut jam ini, happy hour tidak berlaku.
type GlobalHolidaySchedule struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Date      time.Time  `gorm:"type:date;not null;uniqueIndex" json:"date"`
	Name      string     `gorm:"size:150;not null" json:"name"`
	OpenTime  string     `gorm:"type:time;not null" json:"open_time"`
	CloseTime string     `gorm:"type:time;not null" json:"close_time"`
	CreatedBy *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at"`
}
